package scheduler

import (
	"context"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"nidavellir/config"
)

type Scheduler struct {
	ctx        context.Context
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
		ctx:        ctx,
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
	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// tick every 1 minute to check health of the manager
			s.manager.Start()
			log.Print("Scheduler healthy")
		}
	}
}

// Adds a job to the JobManager
func (s *Scheduler) AddJob(sourceId int, trigger string) error {
	source, err := s.db.GetSource(sourceId)
	if err != nil {
		return errors.Wrapf(err, "Could not get sou")
	}
	return s.manager.AddJob(source, trigger)
}
