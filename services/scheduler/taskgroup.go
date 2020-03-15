package scheduler

import (
	"context"
	"fmt"
	"log"
	"regexp"
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
	Duration   time.Duration
	AppFolder  string
	OutputDir  string
}

type ExecutionResult struct {
	Logs      string
	Completed bool
	Steps     []int
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

// Executes the TaskGroup and returns the ExecutionResult. Note that even if the TaskGroup
// returns an error, the ExecutionResult will not be empty. This is because the
// ExecutionResult will store successful intermediate results
func (t *TaskGroup) Execute() (*ExecutionResult, error) {
	output := &ExecutionResult{}

	if len(t.StepGroups) == 0 {
		return output, nil
	}

	var logs []string
	ctx, cancel := context.WithTimeout(t.ctx, t.Duration)
	defer cancel()

	index := 0
	sg := t.StepGroups[index]
	for {
		output.Steps = append(output.Steps, index)
		result, err := sg.ExecuteTasks(ctx, t.sem)
		if err != nil {
			return output, err
		}
		logs = append(logs, result.Log)

		sg, index, err = t.nextStep(index, result.ExitCode)
		if err != nil || sg == nil {
			output.Completed = sg == nil
			output.Logs = formatLogs(t.Name, logs)

			return output, err
		}
	}
}

// determines the next step based on the branching rules conditioned on the current step index and exit code
func (t *TaskGroup) nextStep(index, exitCode int) (*StepGroup, int, error) {
	if exitCode == 0 {
		if index+1 == len(t.StepGroups) {
			return nil, index, nil
		} else {
			return t.StepGroups[index+1], index + 1, nil
		}
	}

	sg := t.StepGroups[index]
	name, exist := sg.Branch[exitCode]
	if !exist {
		return nil, index, errors.Errorf("StepGroup '%s' returned exit code %d which could not be handled", sg.Name, exitCode)
	}

	for i, next := range t.StepGroups[index+1:] {
		// ideal, the next step exists in one of the next step
		if next.Name == name {
			return next, index + i + 1, nil
		}
	}

	return nil, index, errors.Errorf("no valid steps detected after StepGroup '%s' received exit code %d for next StepGroup '%s'", sg.Name, exitCode, name)
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

		sg, err := NewStepGroup(step.Name, groups, step.Branch)
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

func formatLogs(name string, logs []string) string {
	sep := fmt.Sprintf("\n%s\n", strings.Repeat("=", 100))
	body := strings.Join(logs, sep)

	re := regexp.MustCompile(`(?m)^:\s*`)
	content := fmt.Sprintf("Task Group: %s\n%s", name, body)
	return re.ReplaceAllString(content, "")
}
