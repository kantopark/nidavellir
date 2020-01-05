package docker

import (
	"os/exec"

	"github.com/pkg/errors"
)

type Volume struct{}

func (v *Volume) Create(name string) (logs string, err error) {
	cmd := exec.Command("docker", "volume", "create", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "could not create volume '%s'", name)
	}

	return string(output), nil
}

func (v *Volume) Exists(name string) (bool, error) {
	cmd := exec.Command("docker", "volume", "list", "--format", "{{.Name}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, errors.Wrap(err, "could not get list of volume names")
	}

	for _, volume := range splitOutput(output) {
		if volume == name {
			return true, nil
		}
	}

	return false, nil
}
