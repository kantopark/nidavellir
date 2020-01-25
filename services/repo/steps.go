package repo

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"nidavellir/libs"
)

type Step struct {
	Name  string
	Tasks []*Task
	Env   map[string]string
}

type Task struct {
	Name    string
	Tag     string
	Image   string
	Cmd     string
	WorkDir string
	Env     map[string]string
}

func newSteps(steps []rStep, repoName, image, repoDir string, globalEnv map[string]string) ([]*Step, error) {
	var errs error
	var res []*Step
	if len(steps) == 0 {
		return nil, errors.New("no steps in runtime configuration")
	}

	for _, s := range steps {
		step, err := s.newStepGroup(repoName, image, repoDir, globalEnv)
		if err != nil {
			errs = multierror.Append(errs, err)
		} else {
			res = append(res, step)
		}
	}

	if errs != nil {
		return nil, errs
	}
	return res, nil
}

func (s *rStep) newStepGroup(repoName, image, repoDir string, globalEnv map[string]string) (*Step, error) {
	sg := &Step{
		Name:  s.Name,
		Tasks: nil,
		Env:   make(map[string]string),
	}

	// global env has less priority
	for k, v := range globalEnv {
		sg.Env[k] = v
	}

	for k, v := range s.Env {
		sg.Env[k] = v
	}

	if len(s.Tasks) == 0 {
		return nil, errors.Errorf("step '%s' has no tasks")
	}

	for _, t := range s.Tasks {
		task := t.newTask(repoName, s.Name, image, repoDir, sg.Env)
		sg.Tasks = append(sg.Tasks, task)
	}

	return sg, nil
}

func (t *rTask) newTask(repoName, stepName, image, repoDir string, stepEnv map[string]string) *Task {
	f := libs.LowerTrimReplaceSpace
	task := &Task{
		Name:    t.Name,
		Tag:     fmt.Sprintf("%s:%s:%s", f(repoName), f(stepName), f(t.Name)),
		Image:   image,
		Cmd:     t.Cmd,
		WorkDir: repoDir,
		Env:     make(map[string]string),
	}

	// step env has less priority
	for k, v := range stepEnv {
		task.Env[k] = v
	}

	for k, v := range t.Env {
		task.Env[k] = v
	}

	return task
}

// Adds any environment variable to all tasks in repo steps. This variables will take priority
func (r *Repo) AddEnvVars(env map[string]string) {
	for _, sg := range r.Steps {
		for _, task := range sg.Tasks {
			for k, v := range env {
				task.Env[k] = v
			}
		}
	}
}
