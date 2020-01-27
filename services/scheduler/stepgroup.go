package scheduler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"

	"nidavellir/config"
	"nidavellir/libs"
)

type StepGroup struct {
	Name  string
	tasks []*Task
	sep   string
	dur   time.Duration
}

func NewStepGroup(name string, tasks []*Task) (*StepGroup, error) {
	conf, err := config.New()
	if err != nil {
		return nil, errors.Wrap(err, "could not create StepGroup")
	}

	return &StepGroup{
		Name:  strings.TrimSpace(name),
		tasks: tasks,
		sep:   fmt.Sprintf("\n\n%s\n\n", strings.Repeat("-", 100)),
		dur:   conf.Run.MaxDuration,
	}, nil
}

// Executes all tasks within step group in parallel subject to the semaphore weights
func (s *StepGroup) ExecuteTasks(ctx context.Context, sem *semaphore.Weighted) (string, error) {
	var errs error
	errCh := make(chan error, len(s.tasks))
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
	logCh := make(chan string)

	conf, err := config.New()
	if err != nil {
		log.Fatal(errors.Wrap(err, "could not execute tasks as config cannot be read"))
	}

	ctx, cancel := context.WithTimeout(ctx, conf.Run.MaxDuration)
	defer cancel() // this function is called when task executes finishes early and will release resources

	go func() {
		if logs, err := task.Execute(); err != nil {
			errCh <- errors.Wrapf(err, "error executing task: %s", task.TaskName)
		} else {
			logCh <- fmt.Sprintf(`
Task: %s

%s
`, task.TaskName, logs)
		}
	}()

	select {
	case logs := <-logCh:
		ls.Append(logs)
	case <-ctx.Done():
		errCh <- ctx.Err()
	}
}

func (s *StepGroup) Validate() error {
	var errs error
	if libs.IsEmptyOrWhitespace(s.Name) {
		errs = multierror.Append(errs, errors.New("name cannot be empty"))
	}

	if len(s.tasks) == 0 {
		errs = multierror.Append(errs, errors.New("StepGroup has no tasks"))
	}

	taskNameMap := make(map[string]int, len(s.tasks))
	for i, task := range s.tasks {
		if j, exists := taskNameMap[task.TaskTag]; exists {
			errs = multierror.Append(errs, errors.Errorf("task %d and task %d has repeated tag name '%s'", i, j, task.TaskTag))
		} else {
			taskNameMap[task.TaskTag] = i
		}
	}

	return nil
}
