package scheduler

import (
	"regexp"

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
