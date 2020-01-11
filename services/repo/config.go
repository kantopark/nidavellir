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
	Type string `yaml:"type"`
	Tag  string `yaml:"tag"`
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

	config.Setup.format()

	return &config, nil
}

func (s *Setup) format() {
	s.Type = libs.LowerTrim(s.Type)
	s.Tag = libs.LowerTrim(s.Tag)
}

func (r *Runtime) CommitTag(workDir string) (string, error) {
	r.Setup.format()
	tag := r.Setup.Tag

	if tag == "" || tag == "master" || tag == "latest" {
		cmd := exec.Command("git", "rev-parse", "master")
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", errors.Wrap(err, "could not get latest git commit hash")
		} else {
			return strings.TrimSpace(string(output)), nil
		}
	}

	// verify that commitTag given is valid
	cmd := exec.Command("git", "rev-parse", "--verify", tag)
	cmd.Dir = workDir

	if output, err := cmd.CombinedOutput(); err != nil {
		return "", errors.Wrap(err, "could not verify if hash or commit is valid")
	} else if strings.HasPrefix(strings.TrimSpace(string(output)), "fatal") {
		return "", errors.Errorf("%s is not a valid commit or tag", tag)
	}

	return tag, nil
}
