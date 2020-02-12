package scheduler

import (
	"context"
	"os"
	"regexp"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	rp "nidavellir/services/repo"
	"nidavellir/services/store"
)

type JobManager struct {
	ctx     context.Context
	cancel  context.CancelFunc
	db      IStore
	queue   *JobQueue
	started bool
	// An array of completed jobs by the manager, this is primarily used for testing purposes
	CompletedJobs []int
}

// The manager holds a queue of job. Whenever there are new jobs, it will dispatch
// the job. At any one time, it can only run one job. Thus the jobs are queued.
func NewJobManager(db IStore) *JobManager {
	return &JobManager{
		queue:         NewTaskQueue(),
		db:            db,
		started:       false,
		CompletedJobs: []int{},
	}
}

// Starts watching for jobs and executing work
func (m *JobManager) Start() error {
	if !m.started {
		ctx, cancel := context.WithCancel(context.Background())
		m.started = true
		m.ctx = ctx
		m.cancel = cancel
		go m.dispatchWork()

		return nil
	}

	return errors.New("cannot start JobManager as it is already running")
}

// Stops all job and the job manager.
func (m *JobManager) Close() {
	if m.started {
		m.started = false
		m.cancel()
	}
}

// Adds a job into the manager queue. Jobs are saved as TaskGroups in the
// manager queue
func (m *JobManager) AddJob(source *store.Source, trigger string) error {
	job, err := m.db.AddJob(source.Id, trigger)
	if err != nil {
		return err
	}

	repo, err := rp.NewRepo(source.RepoUrl, source.UniqueName)
	if err != nil {
		return err
	}

	tg, err := NewTaskGroup(repo, m.ctx, source.Id, job.Id, source.NextTime)
	if err != nil {
		return err
	}

	extraEnv := source.SecretMap()
	extraEnv["task_date"] = source.NextTime.Format("2006-01-02 15:04:05")
	tg.AddEnvVar(extraEnv)

	m.queue.Enqueue(tg)
	return nil
}

// starts dispatching work
func (m *JobManager) dispatchWork() {
	ch := make(chan bool, 1)
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			if len(ch) == 0 && m.queue.HasJob() {
				ch <- true
				go m.dispatch(m.queue.Dequeue(), ch)
			}
		case <-ch:
			if m.queue.HasJob() {
				ch <- true
				go m.dispatch(m.queue.Dequeue(), ch)
			}
		case <-m.ctx.Done():
			return
		}
	}
}

// Executes the TaskGroup
func (m *JobManager) dispatch(taskGroup *TaskGroup, done <-chan bool) {
	defer func() { <-done }()

	if taskGroup == nil {
		return
	}
	taskDate := regexp.MustCompile(`\D`).ReplaceAllString(taskGroup.TaskDate, "-")
	file, err := os.OpenFile(logFilePath(taskGroup.Name, taskDate), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		log.Println(errors.Wrap(err, "could not open log file"))
		return
	}
	defer func() { _ = file.Close() }()

	source, job, err := m.retrieveWorkDetails(taskGroup)
	if err != nil {
		logOutput(file, err)
		return
	}

	err = m.initWork(source, job)
	if err != nil {
		logOutput(file, err)
		return
	}

	// Execute tasks and save logs if any
	logs, err := taskGroup.Execute()
	if err != nil {
		_ = m.failWork(source, job)
		logOutput(file, err)

		return
	}

	if err := m.completeWork(source, job); err != nil {
		logOutput(file, err)
	}
	logOutput(file, logs)
	m.CompletedJobs = append(m.CompletedJobs, job.Id)
}

// Fetches details about the job from the database
func (m *JobManager) retrieveWorkDetails(tg *TaskGroup) (*store.Source, *store.Job, error) {
	source, err := m.db.GetSource(tg.SourceId)
	if err != nil {
		return nil, nil, errors.Wrap(err, "source could not be retrieved from db")
	}

	job, err := m.db.GetJob(tg.JobId)
	if err != nil {
		return nil, nil, errors.Wrap(err, "job could not be retrieved from db")
	}

	return source, job, nil
}

// Announces that the job is completed
func (m *JobManager) completeWork(source *store.Source, job *store.Job) error {
	source.ToCompleted()
	if err := job.ToSuccessState(); err != nil {
		return err
	}

	return m.updateJobAndSourceStatus(source, job)

}

// Initializes the work
func (m *JobManager) initWork(source *store.Source, job *store.Job) error {
	source.ToRunning()
	if err := job.ToStartState(); err != nil {
		return err
	}

	return m.updateJobAndSourceStatus(source, job)
}

// Announces that the job has failed
func (m *JobManager) failWork(source *store.Source, job *store.Job) error {
	source.ToCompleted()
	if err := job.ToFailureState(); err != nil {
		return err
	}

	return m.updateJobAndSourceStatus(source, job)
}

// Updates the job status
func (m *JobManager) updateJobAndSourceStatus(source *store.Source, job *store.Job) error {
	if _, err := m.db.UpdateJob(*job); err != nil {
		return errors.Wrap(err, "could not update job status")
	}

	if _, err := m.db.UpdateSource(*source); err != nil {
		return errors.Wrap(err, "could not update source status")
	}

	return nil
}
