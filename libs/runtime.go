package libs

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Runtime struct {
	Runtime string            `yaml:"runtime"`
	Env     map[string]string `yaml:"environment"`
	Steps   []Step            `yaml:"steps"`
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

func ReadRuntime(path string) (*Runtime, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not read file content")
	}

	var runtime Runtime
	if err := yaml.Unmarshal(content, &runtime); err != nil {
		return nil, errors.Wrap(err, "could not decode yaml file")
	}

	return &runtime, nil
}
