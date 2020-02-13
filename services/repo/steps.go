package repo

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"nidavellir/libs"
)

type Step struct {
	Name         string
	TaskInfoList []*TaskInfo
	Env          map[string]string
}

type TaskInfo struct {
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
		Name:         s.Name,
		TaskInfoList: nil,
		Env:          make(map[string]string),
	}

	// global env has less priority
	for k, v := range globalEnv {
		sg.Env[k] = v
	}

	for k, v := range s.Env {
		sg.Env[k] = v
	}

	if len(s.Tasks) == 0 {
		return nil, errors.Errorf("step '%s' has no tasks", s.Name)
	}

	for _, t := range s.Tasks {
		task := t.newTask(repoName, s.Name, image, repoDir, sg.Env)
		sg.TaskInfoList = append(sg.TaskInfoList, task)
	}

	return sg, nil
}

func (t *rTask) newTask(repoName, stepName, image, repoDir string, stepEnv map[string]string) *TaskInfo {
	f := libs.LowerTrimReplaceSpace
	task := &TaskInfo{
		Name:    t.Name,
		Tag:     fmt.Sprintf("%s__%s__%s", f(repoName), f(stepName), f(t.Name)),
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
