package dkcontainer

import (
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

type StopOptions struct {
	Name                string
	Port                int
	IgnoreNotFoundError bool
}

func Stop(options *StopOptions) (logs string, err error) {
	containers, err := Search(&SearchOptions{
		Name: options.Name,
		Port: options.Port,
	})
	if err != nil {
		if options.IgnoreNotFoundError {
			return
		}
		return "", errors.Wrap(err, "could not find any containers to stop")
	}

	var stopped []string
	for _, id := range containers {
		cmd := exec.Command("docker", "container", "rm", "-f", id)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", errors.Wrapf(err, "could not stop container '%s'", id)
		}
		stopped = append(stopped, string(output))
		logs += string(output)
	}

	logs = strings.Join(stopped, ", ")
	return
}
