package scheduler

import (
	"context"

	"github.com/pkg/errors"

	"nidavellir/config"
)

type Scheduler struct {
	cancelFunc func()
	db         IStore
	manager    *JobManager
}

// Scheduler pings the database at fixed interval to look for new jobs
// If there are, it will push the job into the manager
func NewScheduler(db IStore, conf config.AppConfig) (*Scheduler, error) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	manager, err := NewJobManager(db, ctx, conf)
	if err != nil {
		return nil, err
	}

	s := &Scheduler{
		cancelFunc: cancelFunc,
		db:         db,
		manager:    manager,
	}

	return s, nil
}

func (s *Scheduler) Close() {
	s.cancelFunc()
	s.manager.Close()
}

// List all the errors
func (s *Scheduler) Errors() []error {
	return s.manager.Errors()
}

// fetches eligible jobs and puts them in the job queue
func (s *Scheduler) Start() {
	s.manager.Start()
}

// Adds a job to the JobManager
func (s *Scheduler) AddJob(sourceId int, trigger string) error {
	source, err := s.db.GetSource(sourceId)
	if err != nil {
		return errors.Wrapf(err, "Could not get sou")
	}
	return s.manager.AddJob(source, trigger)
}
