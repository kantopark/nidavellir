package scheduler

import (
	"context"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"

	"nidavellir/libs"
)

type StepGroup struct {
	Name   string
	Tasks  []*Task
	Branch map[int]string
}

func NewStepGroup(name string, tasks []*Task, branch map[int]string) (*StepGroup, error) {
	sg := &StepGroup{
		Name:   strings.TrimSpace(name),
		Tasks:  tasks,
		Branch: branch,
	}

	if err := sg.Validate(); err != nil {
		return nil, err
	}
	return sg, nil
}

// Executes all tasks within step group in parallel subject to the semaphore weights
func (s *StepGroup) ExecuteTasks(ctx context.Context, sem *semaphore.Weighted) (*TaskOutput, error) {
	ch := make(chan *TaskOutput, len(s.Tasks))

	// set wait group to wait for number of tasks in the current step
	var wg sync.WaitGroup

	// for each task in step, acquire a semaphore and execute task. Once task is complete,
	// release the semaphore and reduce wait group count
	for _, task := range s.Tasks {
		if err := sem.Acquire(ctx, 1); err != nil {
			ch <- &TaskOutput{
				Log:      errors.Wrap(err, "could not acquire semaphore lock to execute tasks").Error(),
				ExitCode: 999,
			}
			continue
		}
		wg.Add(1)
		go runTask(sem, &wg, task, ch)
	}

	// Put the wait group in a go routine. This ensures the done channel is only closed when
	// all jobs in the StepGroup are completed. Meanwhile, the for-loop will either be waiting
	// for the done channel to be closed and also listening to errors from the error channel
	go func() {
		wg.Wait()
		close(ch)
	}()

	outputs := &TaskOutputs{}
	for {
		select {
		case result, ok := <-ch:
			if ok {
				outputs.Add(result)
			} else {
				// channel closed. Time to join all the output together
				return outputs.Combine(), nil
			}

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func runTask(sem *semaphore.Weighted, wg *sync.WaitGroup, task *Task, ch chan<- *TaskOutput) {
	defer sem.Release(1)
	defer wg.Done()
	ch <- task.Execute()
}

func (s *StepGroup) Validate() error {
	var errs error
	if libs.IsEmptyOrWhitespace(s.Name) {
		errs = multierror.Append(errs, errors.New("name cannot be empty"))
	}

	if len(s.Tasks) == 0 {
		errs = multierror.Append(errs, errors.New("StepGroup has no tasks"))
	}

	taskNameMap := make(map[string]int, len(s.Tasks))
	for i, task := range s.Tasks {
		if j, exists := taskNameMap[task.TaskTag]; exists {
			errs = multierror.Append(errs, errors.Errorf("task %d and task %d has repeated tag name '%s'", i, j, task.TaskTag))
		} else {
			taskNameMap[task.TaskTag] = i
		}
	}

	return nil
}
