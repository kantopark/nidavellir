package repo

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"nidavellir/libs"
)

type runtime struct {
	Setup rSetup            `yaml:"setup"`
	Env   map[string]string `yaml:"environment"`
	Steps []rStep           `yaml:"steps"`
}

type rSetup struct {
	Build  bool   `yaml:"build"`
	Commit string `yaml:"commit"`
	Image  string `yaml:"image"`
}

type rStep struct {
	Name   string            `yaml:"name"`
	Tasks  []rTask           `yaml:"tasks"`
	Env    map[string]string `yaml:"environment"`
	Branch []rBranch         `yaml:"branch"`
}

type rBranch struct {
	Code int    `yaml:"code"`
	Step string `yaml:"step"`
}

type rTask struct {
	Name string            `yaml:"name"`
	Cmd  string            `yaml:"cmd"`
	Env  map[string]string `yaml:"environment"`
}

func (r *Repo) formatRuntimeConfig(dir string) error {
	config, err := runtimeFromDir(dir)
	if err != nil {
		return err
	}

	r.Commit = config.Setup.Commit
	r.Image = config.Setup.Image
	r.NeedsBuild = config.Setup.Build

	if libs.LowerTrimReplaceSpace(r.WorkDir) == "" {
		return errors.Errorf("workdir needs to be initialized before initializing steps")
	}

	r.Steps, err = newSteps(config.Steps, r.Name, r.Image, r.WorkDir, config.Env)
	if err != nil {
		return err
	}
	return nil
}

func runtimeFromDir(dir string) (*runtime, error) {
	info, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrap(err, "could not read working directory files")
	}

	file := ""
	for _, f := range info {
		fn := strings.ToLower(f.Name())
		if !f.IsDir() && fn == "runtime.yaml" || fn == "runtime.yml" {
			file = filepath.Join(dir, f.Name())
		}
	}

	if file == "" {
		return nil, errors.New("no runtime.yaml found in working directory")
	}

	content, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrap(err, "could not read file content")
	}

	var config runtime
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, errors.Wrap(err, "could not decode yaml file")
	}

	if err := config.Setup.format(filepath.Dir(file)); err != nil {
		return nil, errors.Wrap(err, "could not format tag")
	}

	return &config, nil
}

func (s *rSetup) format(workDir string) error {
	s.Image = strings.TrimSpace(s.Image)
	if s.Image == "" {
		return errors.Errorf("image cannot be empty")
	}

	commit := libs.LowerTrim(s.Commit)

	if commit == "" || commit == "master" || commit == "latest" {
		// gets latest tag since tag not specified
		cmd := exec.Command("git", "rev-parse", "master")
		cmd.Dir = workDir

		if output, err := cmd.CombinedOutput(); err != nil {
			return errors.Wrap(err, "could not get latest git commit hash")
		} else {
			commit = strings.TrimSpace(string(output))
		}
	} else {
		// check specified tag exists
		cmd := exec.Command("git", "rev-parse", "--verify", commit)
		cmd.Dir = workDir

		if output, err := cmd.CombinedOutput(); err != nil {
			return errors.Wrap(err, "could not verify if hash or commit is valid")
		} else if strings.HasPrefix(strings.TrimSpace(string(output)), "fatal") {
			return errors.Errorf("%s is not a valid commit or tag", commit)
		}
	}

	s.Commit = commit

	return nil
}
