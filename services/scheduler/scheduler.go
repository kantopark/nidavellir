package scheduler

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"nidavellir/services/store"
)

type Scheduler struct {
	cancelFunc func()
	ctx        context.Context
	err        chan error
	store      IStore
	manager    *JobManager
}

// Scheduler pings the database at fixed interval to look for new jobs
// If there are, it will push the job into the manager
func NewScheduler(db IStore, appFolderPath string) (*Scheduler, error) {
	manager, err := NewJobManager(db, appFolderPath)
	if err != nil {
		return nil, err
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	s := &Scheduler{
		cancelFunc: cancelFunc,
		ctx:        ctx,
		err:        make(chan error),
		manager:    manager,
		store:      db,
	}

	return s, nil
}

func (s *Scheduler) Errors() <-chan error {
	return s.err
}

func (s *Scheduler) Close() {
	s.cancelFunc()
	s.manager.Close()
}

// fetches eligible jobs and puts them in the job queue
func (s *Scheduler) Start() {
	s.manager.Start()
	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ticker.C:
			todos, err := s.store.GetSources(&store.GetSourceOption{
				ScheduledToRun: true,
				MaskSecrets:    false,
			})
			if err != nil {
				s.err <- errors.Wrap(err, "could not fetch sources in scheduler")
				continue
			}

			for _, t := range todos {
				if err := s.manager.AddJob(t, store.TriggerSchedule); err != nil {
					s.err <- errors.Wrap(err, "could not add new job")
				}
			}

		case <-s.ctx.Done():
			return
		}
	}
}

// Adds a job to the JobManager
func (s *Scheduler) AddJob(sourceId int, trigger string) error {
	source, err := s.store.GetSource(sourceId)
	if err != nil {
		return errors.Wrapf(err, "Could not get sou")
	}
	return s.manager.AddJob(source, trigger)
}
