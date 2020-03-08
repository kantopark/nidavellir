package scheduler

import (
	"errors"
	"sync"
)

type JobQueue struct {
	list []*TaskGroup
	lock sync.RWMutex
}

func NewJobQueue() *JobQueue {
	return &JobQueue{
		list: []*TaskGroup{},
		lock: sync.RWMutex{},
	}
}

// Moves a TaskGroup to the back of the queue
func (q *JobQueue) Enqueue(tg *TaskGroup) {
	q.lock.Lock()
	q.list = append(q.list, tg)
	q.lock.Unlock()
}

// Removes the first item in the JobQueue
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

// Moves a TaskGroup to the top of the queue where it'll be executed next (immediately)
func (q *JobQueue) EnqueueTop(tg *TaskGroup) {
	q.lock.Lock()
	q.list = append([]*TaskGroup{tg}, q.list...)
	q.lock.Unlock()
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
