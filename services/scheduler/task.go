package scheduler

import (
	"regexp"

	"nidavellir/services/docker"
)

type Task struct {
	TaskTag    string
	SourceId   int
	SourceName string
	JobId      int
	Step       string
	Name       string
	Cmd        string
	Env        map[string]string
	Image      string
}

func NewTask(task *Task) (*Task, error) {
	return &Task{
		TaskTag:    task.TaskTag,
		SourceId:   task.SourceId,
		SourceName: task.SourceName,
		JobId:      task.JobId,
		Step:       task.Step,
		Name:       task.Name,
		Cmd:        task.Cmd,
		Env:        task.Env,
	}, nil
}

func (t *Task) Execute() (string, error) {
	c := docker.NewContainer()
	re := regexp.MustCompile(`\s`)

	return c.Run(&docker.ContainerRunOptions{
		Image:   t.Image,
		Name:    t.TaskTag,
		Restart: "no",
		Env:     t.Env,
		Cmd:     re.Split(t.Cmd, -1),
		Volumes: map[string]string{
			outputDir(t.JobId): "/output",
		},
		Daemon: false,
	})
}
