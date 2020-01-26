package scheduler

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"
)

type StepGroup struct {
	Name  string
	tasks []*Task
	sep   string
}

func NewStepGroup(name string, tasks []*Task) *StepGroup {
	return &StepGroup{
		Name:  name,
		tasks: tasks,
		sep:   fmt.Sprintf("\n\n%s\n\n", strings.Repeat("-", 100)),
	}
}

func (s *StepGroup) SetImage(image string) {
	for _, task := range s.tasks {
		task.Image = image
	}
}

// Executes all tasks within step group in parallel subject to the semaphore weights
func (s *StepGroup) ExecuteTasks(ctx context.Context, sem *semaphore.Weighted) (string, error) {
	var errs error
	errCh := make(chan error, 1)
	done := make(chan bool, 1)
	ls := NewLogSlice()

	// set wait group to wait for number of tasks in the current step
	var wg sync.WaitGroup

	// for each task in step, acquire a semaphore and execute task. Once task is complete,
	// release the semaphore and reduce wait group count
	for _, task := range s.tasks {
		if err := sem.Acquire(ctx, 1); err != nil {
			errCh <- errors.Wrap(err, "could not acquire semaphore lock to execute tasks")
			continue
		}
		wg.Add(1)
		go runTask(sem, &wg, task, errCh, ls)
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

func runTask(sem *semaphore.Weighted, wg *sync.WaitGroup, task *Task, errCh chan<- error, ls *LogSlice) {
	defer sem.Release(1)
	defer wg.Done()
	if logs, err := task.Execute(); err != nil {
		errCh <- errors.Wrapf(err, "error executing task: %s", task.TaskName)
	} else {
		ls.Append(fmt.Sprintf(`
Task: %s

%s
`, task.TaskName, logs))
	}
}
