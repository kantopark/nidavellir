package scheduler

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"nidavellir/libs"
	"nidavellir/services/iofiles"
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
	// Path to folder/volume that stores task output and logs
	AppFolderPath string
}

// The manager holds a queue of job. Whenever there are new jobs, it will dispatch
// the job. At any one time, it can only run one job. Thus the jobs are queued.
// You should not be creating a JobManager, but should call NewScheduler which will
// create a JobManager internally.
func NewJobManager(db IStore, appFolderPath string) (*JobManager, error) {
	if !libs.PathExists(appFolderPath) {
		err := os.MkdirAll(appFolderPath, 0777)
		if err != nil {
			return nil, errors.Wrap(err, "could not create data folder")
		}
	}

	return &JobManager{
		queue:         NewJobQueue(),
		db:            db,
		started:       false,
		CompletedJobs: []int{},
		AppFolderPath: appFolderPath,
	}, nil
}

// Starts watching for jobs and executing work
func (m *JobManager) Start() {
	if !m.started {
		ctx, cancel := context.WithCancel(context.Background())
		m.started = true
		m.ctx = ctx
		m.cancel = cancel
		go m.dispatchWork()
	}
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

	repo, err := rp.NewRepo(source.RepoUrl, source.UniqueName, m.AppFolderPath)
	if err != nil {
		return err
	}

	tg, err := NewTaskGroup(repo, m.ctx, source.Id, job.Id, source.NextTime, m.AppFolderPath)
	if err != nil {
		return err
	}

	extraEnv := source.SecretMap()
	extraEnv["task_date"] = source.NextTime.Format("2006-01-02 15:04:05")
	tg.AddEnvVar(extraEnv)

	switch trigger {
	case store.TriggerManual:
		m.queue.EnqueueTop(tg)
	default:
		m.queue.Enqueue(tg)
	}

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
	defer func() {
		<-done

		data, err := json.MarshalIndent(struct {
			Name string `json:"name"`
			Date string `json:"date"`
		}{taskGroup.Name, taskGroup.TaskDate}, "", "")
		if err != nil {
			log.Printf("could not save task meta data: %s", err.Error())
		}
		path := iofiles.GetMetaFilePath(m.AppFolderPath, taskGroup.SourceId, taskGroup.JobId)
		_ = ioutil.WriteFile(path, data, 0666)
	}()

	if taskGroup == nil {
		return
	}

	logFile, err := iofiles.NewLogFile(m.AppFolderPath, taskGroup.SourceId, taskGroup.JobId, false)
	if err != nil {
		log.Println(errors.Wrap(err, "could not create log file"))
		return
	}
	defer logFile.Close()

	source, job, err := m.retrieveWorkDetails(taskGroup)
	if err != nil {
		err = multierror.Append(err, logFile.Write(err))
		log.Println(err)
		return
	}

	err = m.initWork(source, job)
	if err != nil {
		err = multierror.Append(err, logFile.Write(err))
		log.Println(err)
		return
	}

	// Execute tasks and save logs if any
	logs, err := taskGroup.Execute()
	if err != nil {
		err = multierror.Append(err, m.failWork(source, job))
		err = multierror.Append(err, logFile.Write(err))
		log.Println(err)
		return
	}

	if err := m.completeWork(source, job); err != nil {
		err = multierror.Append(err, logFile.Write(err))
		log.Println(err)
	}
	_ = logFile.Write(logs)
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
	if _, err := m.db.UpdateJob(job); err != nil {
		return errors.Wrap(err, "could not update job status")
	}

	if _, err := m.db.UpdateSource(source); err != nil {
		return errors.Wrap(err, "could not update source status")
	}

	return nil
}
