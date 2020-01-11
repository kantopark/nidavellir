package scheduler

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"nidavellir/config"
	"nidavellir/services/repo"
	"nidavellir/services/store"
)

type JobManager struct {
	conf  *config.Config
	ctx   context.Context
	db    IStore
	queue *JobQueue
}

// The manager holds a queue of job. Whenever there are new jobs, it will dispatch
// the job. At any one time, it can only run one job. Thus the jobs are queued.
func NewJobManager(ctx context.Context) (*JobManager, error) {
	conf, err := config.New()
	if err != nil {
		return nil, err
	}

	m := &JobManager{
		conf:  conf,
		ctx:   ctx,
		queue: NewTaskQueue(),
	}

	go m.dispatchWork()

	return m, nil
}

// Adds a job into the manager queue. Jobs are saved as TaskGroups in the
// manager queue
func (m *JobManager) AddJob(source *store.Source, trigger string) error {
	job, err := m.db.AddJob(source.Id, trigger)
	if err != nil {
		return err
	}

	conf, err := repo.RuntimeFromDir(m.conf.WorkDir.RepoPath(source.UniqueName))
	if err != nil {
		return err
	}

	tg, err := NewTaskGroup(m.ctx, source, job.Id, source.Name)
	if err != nil {
		return err
	}
	if tg.BuildLog != "" {
		file, err := os.OpenFile(imageLogPath(tg.Image), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		logOutput(file, tg.BuildLog)
	}

	for i, step := range conf.Steps {
		var groups []*Task
		for j, task := range step.Tasks {
			envVars := taskEnvVar(source.SecretMap(), conf.Env, task.Env, source.NextTime)

			t, err := NewTask(&Task{
				TaskTag:    fmt.Sprintf("%s_%d-%d", source.UniqueName, i, j),
				SourceId:   source.Id,
				SourceName: source.Name,
				JobId:      job.Id,
				Step:       step.Step,
				Name:       task.Name,
				Cmd:        task.Cmd,
				Env:        envVars,
			})
			if err != nil {
				return errors.Wrap(err, "invalid task specifications")
			}

			groups = append(groups, t)
		}

		tg.AddTasks(groups)
	}

	m.queue.Enqueue(tg)
	return nil
}

func (m *JobManager) dispatchWork() {
	var count int64
	count = 0
	ticker := time.NewTicker(5 * time.Second)

	dispatch := func() {
		taskGroup := m.queue.Dequeue()
		if taskGroup == nil {
			return
		}
		file, err := os.OpenFile(logFilePath(taskGroup.Name, taskGroup.TaskDate), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
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

		if err := m.initWork(source, job); err != nil {
			logOutput(file, err)
			return
		}

		// Execute tasks and save logs if any
		if err := taskGroup.Execute(); err != nil {
			if taskGroup.BuildLog != "" {
				logOutput(file, taskGroup.BuildLog)
			}
			logOutput(file, err)

			if _ = m.failWork(source, job); err != nil {
				log.Println(err)
			}
		} else {
			if taskGroup.BuildLog != "" {
				logOutput(file, taskGroup.BuildLog)
			}

			if err := m.completeWork(source, job); err != nil {
				log.Println(err)
			}
		}
	}

	for {
		select {
		case <-ticker.C:
			if count == 0 && m.queue.HasJob() {
				atomic.AddInt64(&count, 1)
				dispatch()
				atomic.AddInt64(&count, -1)
			}
		case <-m.ctx.Done():
			return
		}
	}

}

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

func (m *JobManager) initWork(source *store.Source, job *store.Job) error {
	source.ToRunning()
	if err := job.ToStartState(); err != nil {
		return err
	}

	return m.updateJobAndSourceStatus(source, job)
}

func (m *JobManager) failWork(source *store.Source, job *store.Job) error {
	source.ToCompleted()
	if err := job.ToFailureState(); err != nil {
		return err
	}

	return m.updateJobAndSourceStatus(source, job)
}

func (m *JobManager) completeWork(source *store.Source, job *store.Job) error {
	source.ToCompleted()
	if err := job.ToSuccessState(); err != nil {
		return err
	}

	return m.updateJobAndSourceStatus(source, job)

}

func (m *JobManager) updateJobAndSourceStatus(source *store.Source, job *store.Job) error {
	if _, err := m.db.UpdateJob(*job); err != nil {
		return errors.Wrap(err, "could not update job status")
	}

	if _, err := m.db.UpdateSource(*source); err != nil {
		return errors.Wrap(err, "could not update source status")
	}

	return nil
}

func taskEnvVar(secrets, runtimeEnvs, taskEnvs map[string]string, taskDate time.Time) map[string]string {
	envs := make(map[string]string)

	for _, envMap := range []map[string]string{secrets, runtimeEnvs, taskEnvs} {
		for key, value := range envMap {
			envs[key] = value
		}
	}

	envs["task_date"] = taskDate.Format("2006-01-02 15:04:05")

	return envs
}
