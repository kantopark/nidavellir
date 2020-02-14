package scheduler

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"

	"nidavellir/services/iofiles"
	"nidavellir/services/repo"
)

type TaskGroup struct {
	Name       string
	StepGroups []*StepGroup
	ctx        context.Context
	sem        *semaphore.Weighted
	rp         *repo.Repo
	SourceId   int
	JobId      int
	TaskDate   string
	Completed  bool
	Duration   time.Duration
	AppFolder  string
	OutputDir  string
}

func NewTaskGroup(rp *repo.Repo, ctx context.Context, sourceId, jobId int, taskDate time.Time, appFolder string) (*TaskGroup, error) {
	outputDir, err := iofiles.GetOutputDir(appFolder, sourceId, jobId)
	if err != nil {
		return nil, err
	}

	tg := &TaskGroup{
		Name:       rp.Name,
		ctx:        ctx,
		rp:         rp,
		sem:        semaphore.NewWeighted(int64(runtime.NumCPU())),
		StepGroups: []*StepGroup{},
		SourceId:   sourceId,
		JobId:      jobId,
		TaskDate:   taskDate.Format("2006-01-02 15:04:05"),
		Completed:  false,
		Duration:   1 * time.Hour, // default duration is 1 hour
		AppFolder:  appFolder,
		OutputDir:  outputDir,
	}

	if err := tg.updateRepo(); err != nil {
		return nil, err
	}

	// Checks if image needs to be built
	if rp.NeedsBuild {
		// if so, check that image is updated. If image is updated, don't build, else build
		err := tg.updateImage()
		if err != nil {
			return nil, err
		}
	} else if err := tg.pullImage(); err != nil {
		// no need to build, but check if image exists, if not pull image
		return nil, err
	}

	if err := tg.addStepGroups(); err != nil {
		return nil, errors.Wrap(err, "could not create TaskGroup due to errors in StepGroup configuration")
	}

	return tg, nil
}

// Adds any environment variable to all tasks in the TaskGroup. These variables will have higher priority
func (t *TaskGroup) AddEnvVar(env map[string]string) *TaskGroup {
	for _, sg := range t.StepGroups {
		for _, task := range sg.Tasks {
			for k, v := range env {
				task.Env[k] = v
			}
		}
	}
	return t
}

// Sets the maximum job duration.
func (t *TaskGroup) SetMaxDuration(duration time.Duration) *TaskGroup {
	t.Duration = duration
	return t
}

func (t *TaskGroup) Execute() (string, error) {
	var logArray []string

	formatLogs := func() string {
		sep := fmt.Sprintf("\n%s\n", strings.Repeat("=", 100))
		return fmt.Sprintf("Task Group: %s\n\n%s", t.Name, strings.Join(logArray, sep))
	}

	ctx, cancel := context.WithTimeout(t.ctx, t.Duration)
	defer cancel()

	for _, sg := range t.StepGroups {
		logs, err := sg.ExecuteTasks(ctx, t.sem)
		if err != nil {
			return formatLogs(), err
		}
		logArray = append(logArray, logs)
	}
	t.Completed = true

	return formatLogs(), nil
}

// Adds StepGroups from the repo.Steps information. Order of execution for the StepGroup
// is determined by their relative position in the repo's runtime.yaml config file.
//Tasks in each StepGroup will be executed in parallel.
func (t *TaskGroup) addStepGroups() error {
	for _, step := range t.rp.Steps {
		var groups []*Task

		for _, task := range step.TaskInfoList {
			t, err := NewTask(
				task.Name,
				task.Image,
				fmt.Sprintf("%s_%d", task.Tag, t.JobId),
				task.Cmd,
				t.OutputDir,
				task.WorkDir,
				task.Env,
			)
			if err != nil {
				return errors.Wrap(err, "invalid task specifications")
			}

			groups = append(groups, t)
		}

		sg, err := NewStepGroup(step.Name, groups)
		if err != nil {
			return err
		}

		t.StepGroups = append(t.StepGroups, sg)
	}
	return nil
}

// Updates the repo to the latest version
func (t *TaskGroup) updateRepo() error {
	if err := t.rp.Clone(); err != nil {
		return err
	}
	return nil
}

// Check image is updated
func (t *TaskGroup) updateImage() error {
	rp := t.rp

	// check if the image exists, if not clone repo and build image
	if exists, err := repo.ImageExists(rp.Image); err != nil {
		return err
	} else if exists {
		return nil
	}

	// build image since it does not exist
	logs, err := rp.BuildImage()
	if err != nil {
		return err
	}
	logs = fmt.Sprintf("Building image for task group: %s\n\n%s", t.Name, logs)

	t.logImageOutput(logs)
	return nil
}

func (t *TaskGroup) pullImage() error {
	rp := t.rp

	hasImage, err := rp.HasImage()
	if err != nil {
		return err
	} else if !hasImage {
		logs, err := rp.PullImage()
		if err != nil {
			return err
		}
		logs = fmt.Sprintf("Pulling image for task group: %s\n\n%s", t.Name, logs)
		t.logImageOutput(logs)
	}

	return nil
}

// saves the image build logs into a file
func (t *TaskGroup) logImageOutput(logs string) {
	logFile, err := iofiles.NewImageLogFile(t.AppFolder, t.SourceId, t.JobId, false)
	if err != nil {
		log.Println(errors.Wrap(err, "could not create log file"))
		return
	}
	defer logFile.Close()

	_ = logFile.Write(logs)
}
