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

type Runtime struct {
	Setup Setup             `yaml:"setup"`
	Env   map[string]string `yaml:"environment"`
	Steps []Step            `yaml:"steps"`
}

type Setup struct {
	Build  bool   `yaml:"build"`
	Commit string `yaml:"commit"`
	Image  string `yaml:"image"`
}

type Step struct {
	Step  string `yaml:"step"`
	Tasks []Task `yaml:"tasks"`
}

type Task struct {
	Name string            `yaml:"name"`
	Cmd  string            `yaml:"cmd"`
	Env  map[string]string `yaml:"env"`
}

func RuntimeFromDir(dir string) (*Runtime, error) {
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

	return NewRuntime(file)
}

func NewRuntime(path string) (*Runtime, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not read file content")
	}

	var config Runtime
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, errors.Wrap(err, "could not decode yaml file")
	}

	if err := config.Setup.format(filepath.Dir(path)); err != nil {
		return nil, errors.Wrap(err, "could not format tag")
	}

	return &config, nil
}

func (s *Setup) format(workDir string) error {
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
