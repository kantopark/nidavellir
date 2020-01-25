package scheduler

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"

	"nidavellir/libs"
	"nidavellir/services/repo"
)

type TaskGroup struct {
	JobId      int
	Name       string
	SourceId   int
	StepGroups []*StepGroup
	Image      string
	BuildLog   string
	RunLog     string
	TaskDate   string
	ctx        context.Context
	sem        *semaphore.Weighted
}

func NewTaskGroup(ctx context.Context, sourceId, jobId int, taskName, repoUrl, repoUniqueName, commitTag string, taskDate time.Time) (*TaskGroup, error) {
	tg := &TaskGroup{
		Name:       taskName,
		SourceId:   sourceId,
		TaskDate:   taskDate.Format("2006-01-02 15:04:05"),
		JobId:      jobId,
		ctx:        ctx,
		sem:        semaphore.NewWeighted(int64(runtime.NumCPU())),
		StepGroups: []*StepGroup{},
	}

	rp, err := repo.NewRepo(repoUrl, repoUniqueName)
	if err != nil {
		return nil, err
	}

	if err := tg.updateRepo(rp); err != nil {
		return nil, err
	}

	// Checks if image needs to be built
	if rp.NeedsBuild {
		// if so, check that image is updated. If image is updated, don't build, else build
		err := tg.updateImage(rp, repoUniqueName, commitTag)
		if err != nil {
			return nil, err
		}
	} else {
		// no need to build, but check if image exists, if not pull image
		hasImage, err := rp.HasImage()
		if err != nil {
			return nil, err
		} else if !hasImage {
			logs, err := rp.PullImage()
			if err != nil {
				return nil, err
			}
			tg.BuildLog = logs
		}
	}

	return tg, nil
}

// Adds tasks of the same level to the task group, the order in which these tasks are
// added will determine the order of execution
func (t *TaskGroup) AddTasks(tasks []*Task) {
	sg := NewStepGroup(tasks)
	t.StepGroups = append(t.StepGroups, sg)
}

func (t *TaskGroup) Execute() error {
	var logArray []string
	sep := fmt.Sprintf("\n%s\n", strings.Repeat("=", 100))

	for _, sg := range t.StepGroups {
		sg.SetImage(t.Image)

		logs, err := sg.ExecuteTasks(t.ctx, t.sem)
		if err != nil {
			t.RunLog = strings.Join(logArray, sep)
			return err
		}
		logArray = append(logArray, logs)
	}

	t.RunLog = strings.Join(logArray, sep)
	return nil
}

// Updates the repo to the latest version
func (t *TaskGroup) updateRepo(rp *repo.Repo) error {
	if err := rp.Clone(); err != nil {
		return err
	}
	return nil
}

// Check image is updated
func (t *TaskGroup) updateImage(rp *repo.Repo, uniqueName, commitTag string) error {
	if libs.IsEmptyOrWhitespace(commitTag) {
		return errors.New("commit tag cannot be empty")
	}

	image := fmt.Sprintf("%s:%s", uniqueName, commitTag)

	// check if the image exists, if not clone repo and build image
	if exists, err := repo.ImageExists(image); err != nil {
		return err
	} else if !exists {
		if logs, err := rp.BuildImage(); err != nil {
			return err
		} else {
			t.BuildLog = logs
		}
	}

	t.Image = image
	return nil
}
