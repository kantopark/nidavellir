package scheduler

import (
	"errors"
	"sync"
)

type JobQueue struct {
	list []*TaskGroup
	lock sync.RWMutex
}

func NewTaskQueue() *JobQueue {
	return &JobQueue{
		list: []*TaskGroup{},
		lock: sync.RWMutex{},
	}
}

func (q *JobQueue) Enqueue(tg *TaskGroup) {
	q.lock.Lock()
	q.list = append(q.list, tg)
	q.lock.Unlock()
}

func (q *JobQueue) Dequeue() *TaskGroup {
	if !q.HasJob() {
		return nil
	}

	q.lock.Lock()
	tg := q.list[0]
	q.list = q.list[1:]
	q.lock.Unlock()
	return tg
}

func (q *JobQueue) Len() int {
	return len(q.list)
}

func (q *JobQueue) HasJob() bool {
	return q.Len() > 0
}

func (q *JobQueue) First() (*TaskGroup, error) {
	if q.Len() == 0 {
		return nil, errors.New("queue is empty")
	}

	return q.list[0], nil
}
