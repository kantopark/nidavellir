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
	manager    IManager
}

func NewScheduler(db IStore, manager IManager) *Scheduler {
	ctx, cancelFunc := context.WithCancel(context.Background())
	s := &Scheduler{
		cancelFunc: cancelFunc,
		ctx:        ctx,
		err:        make(chan error),
		manager:    manager,
		store:      db,
	}

	go s.fetchAndQueueJobs()

	return s
}

func (s *Scheduler) Errors() <-chan error {
	return s.err
}

func (s *Scheduler) Close() {
	s.cancelFunc()
}

// fetches eligible jobs and puts them in the job queue
func (s *Scheduler) fetchAndQueueJobs() {
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
					continue
				}
			}

		case <-s.ctx.Done():
			return
		}
	}
}
