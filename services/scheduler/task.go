package scheduler

import (
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

func (t *Task) Execute() (string, error) {
	re := regexp.MustCompile(`\s`)

	return container.Run(&container.RunOptions{
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
}
