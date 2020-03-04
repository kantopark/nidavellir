package scheduler

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"

	"nidavellir/libs"
)

type StepGroup struct {
	Name  string
	Tasks []*Task
	sep   string
}

func NewStepGroup(name string, tasks []*Task) (*StepGroup, error) {
	sg := &StepGroup{
		Name:  strings.TrimSpace(name),
		Tasks: tasks,
		sep:   fmt.Sprintf("\n\n%s\n\n", strings.Repeat("-", 100)),
	}

	if err := sg.Validate(); err != nil {
		return nil, err
	}
	return sg, nil
}

// Executes all tasks within step group in parallel subject to the semaphore weights
func (s *StepGroup) ExecuteTasks(ctx context.Context, sem *semaphore.Weighted) (string, error) {
	var errs error
	errCh := make(chan error, len(s.Tasks))
	done := make(chan bool, 1)
	ls := NewLogSlice()

	// set wait group to wait for number of tasks in the current step
	var wg sync.WaitGroup

	// for each task in step, acquire a semaphore and execute task. Once task is complete,
	// release the semaphore and reduce wait group count
	for _, task := range s.Tasks {
		if err := sem.Acquire(ctx, 1); err != nil {
			errCh <- errors.Wrap(err, "could not acquire semaphore lock to execute tasks")
			continue
		}
		wg.Add(1)
		go runTask(sem, &wg, ctx, task, errCh, ls)
	}

	// Put the wait group in a go routine. This ensures the done channel is only closed when
	// all jobs in the StepGroup are completed. Meanwhile, the for-loop will either be waiting
	// for the done channel to be closed and also listening to errors from the error channel
	go func() {
		wg.Wait()
		close(done)
	}()

	for {
		select {
		case err := <-errCh:
			errs = multierror.Append(errs, err)
		case <-done:
			// case when the done channel is closed cause all tasks executed successfully,
			// return errors if any
			return fmt.Sprintf("Step Group: %s\n%s\n\n", s.Name, ls.Join(s.sep)), errs
		}
	}
}

func runTask(sem *semaphore.Weighted, wg *sync.WaitGroup, ctx context.Context, task *Task, errCh chan<- error, ls *LogSlice) {
	defer sem.Release(1)
	defer wg.Done()
	done := make(chan bool, 1)

	go func() {
		if logs, err := task.Execute(); err != nil {
			errCh <- errors.Wrapf(err, "error executing task: %s", task.TaskName)
		} else {
			ls.Append(fmt.Sprintf(`
Task: %s

%s
`, task.TaskName, logs))
		}
		close(done)
	}()

	select {
	case <-ctx.Done():
		errCh <- ctx.Err()
	case <-done:
	}
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
