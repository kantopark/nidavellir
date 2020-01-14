package dkcontainer

import (
	"os/exec"

	"github.com/pkg/errors"
)

type StopOptions struct {
	Name string
	Port int
}

func Stop(options *StopOptions) (logs string, err error) {
	containers, err := Search(&SearchOptions{
		Name: options.Name,
		Port: options.Port,
	})
	if err != nil {
		return "", errors.Wrap(err, "could not find any containers to stop")
	}

	for _, id := range containers {
		cmd := exec.Command("docker", "container", "rm", "-f", id)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", errors.Wrapf(err, "could not stop container '%s'", id)
		}
		logs += string(output) + "\n"
	}
	return
}
