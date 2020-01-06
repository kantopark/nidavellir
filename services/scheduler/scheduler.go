package scheduler

import (
	"context"
	"time"

	"nidavellir/services/store"
)

type Scheduler struct {
	db         Db
	queue      chan *store.Job
	ctx        context.Context
	cancelFunc func()
	err        chan error
}

type Db interface {
	GetJobs(options *store.ListJobOption) ([]*store.Job, error)
}

func New(db Db) *Scheduler {
	ctx, cancelFunc := context.WithCancel(context.Background())
	s := &Scheduler{
		db:         db,
		queue:      make(chan *store.Job, 100),
		ctx:        ctx,
		cancelFunc: cancelFunc,
		err:        make(chan error, 1),
	}

	go s.fetchJobs()

	return s
}

// Returns the error channel. Use this as a channel to implement an error
// stop in the main function
func (s *Scheduler) Error() <-chan error {
	return s.err
}

func (s *Scheduler) Close() {
	s.cancelFunc()
}

// fetches eligible jobs and puts them in the job queue
func (s *Scheduler) fetchJobs() {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			jobs, err := s.db.GetJobs(&store.ListJobOption{
				Trigger: store.TriggerSchedule,
				State:   store.ScheduleNoop,
			})
			if err != nil {
				s.err <- err
				return
			}
			for _, j := range jobs {
				s.queue <- j
			}
		case <-s.ctx.Done():
			return
		}
	}
}
