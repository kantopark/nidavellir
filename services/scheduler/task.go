package scheduler

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	container "nidavellir/services/docker/dkcontainer"
)

type Task struct {
	// Name of task
	TaskName string
	// Image to use for task
	Image string
	// Container run tag
	TaskTag   string
	Cmd       string
	Env       map[string]string
	OutputDir string
	WorkDir   string
}

func NewTask(taskName, image, tag, cmd, outputDir, workDir string, env map[string]string) (*Task, error) {
	taskName = strings.TrimSpace(taskName)
	if taskName == "" {
		return nil, errors.Errorf("task name cannot be empty")
	}

	for name, value := range map[string]string{
		"task name":        taskName,
		"image":            image,
		"task tag":         tag,
		"command":          cmd,
		"output directory": outputDir,
	} {
		if strings.TrimSpace(value) == "" {
			return nil, errors.Errorf("%s cannot be empty", name)
		}
	}

	return &Task{
		TaskName:  taskName,
		TaskTag:   tag,
		Image:     image,
		Cmd:       cmd,
		Env:       env,
		OutputDir: outputDir,
		WorkDir:   workDir,
	}, nil
}

type TaskOutput struct {
	Log      string
	ExitCode int
}

type TaskOutputs struct {
	outputs []*TaskOutput
}

func (t *Task) Execute() *TaskOutput {
	re := regexp.MustCompile(`\s`)

	result, err := container.Run(&container.RunOptions{
		Image:   t.Image,
		Name:    t.TaskTag,
		Restart: "no",
		Env:     t.Env,
		Cmd:     re.Split(t.Cmd, -1),
		Volumes: map[string]string{
			t.WorkDir:   "/repo",
			t.OutputDir: "/output",
		},
		Daemon:  false,
		WorkDir: t.WorkDir,
	})

	if result == nil {
		panic("result should never be empty")
	}

	logs := []string{"Task: " + t.TaskName, "\n", result.Logs}
	if err != nil {
		logs = append(logs, err.Error())
	}

	return &TaskOutput{
		Log:      strings.TrimSpace(strings.Join(logs, "\n")),
		ExitCode: result.ExitCode,
	}
}

// Add a task result to the list of outputs
func (t *TaskOutputs) Add(result *TaskOutput) {
	t.outputs = append(t.outputs, result)
}

// Returns the maximum of all the exit codes in the task outputs
func (t *TaskOutputs) ExitCode() int {
	exitCode := 0
	for _, r := range t.outputs {
		if r.ExitCode > exitCode {
			exitCode = r.ExitCode
		}
	}
	return exitCode
}

func (t *TaskOutputs) Logs() string {
	var logs []string
	sep := fmt.Sprintf("\n\n%s\n\n", strings.Repeat("-", 100))

	for _, r := range t.outputs {
		logs = append(logs, r.Log)
	}

	return strings.Join(logs, sep)
}

func (t *TaskOutputs) Combine() *TaskOutput {
	return &TaskOutput{
		Log:      t.Logs(),
		ExitCode: t.ExitCode(),
	}
}
