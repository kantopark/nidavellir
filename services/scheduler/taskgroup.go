package scheduler

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"

	"nidavellir/services/repo"
	"nidavellir/services/store"
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

func NewTaskGroup(ctx context.Context, source *store.Source, jobId int, name string) (*TaskGroup, error) {
	tg := &TaskGroup{
		Name:       name,
		SourceId:   source.Id,
		TaskDate:   source.NextTime.Format("2006-01-02 15:04:05"),
		JobId:      jobId,
		ctx:        ctx,
		sem:        semaphore.NewWeighted(int64(runtime.NumCPU())),
		StepGroups: []*StepGroup{},
	}

	if err := tg.updateImage(source); err != nil {
		return nil, err
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

func (t *TaskGroup) updateImage(source *store.Source) error {
	// Update repo and check image is updated
	rp, err := repo.NewRepo(source.RepoUrl, source.UniqueName)
	if err != nil {
		return err
	}

	// clone or update repo
	if err := rp.Clone(); err != nil {
		return err
	}

	// check if image is updated, if not rebuild it
	commitTag := strings.TrimSpace(source.CommitTag)
	if commitTag == "" {
		if hash, err := latestHash(repoDir(source.UniqueName)); err != nil {
			return err
		} else {
			commitTag = hash
		}
	}

	image := fmt.Sprintf("%s:%s", source.UniqueName, commitTag)

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

func latestHash(dirname string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "master")
	cmd.Dir = dirname
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, "could not get latest hash from repo")
	}
	return strings.TrimSpace(string(output)), nil
}
