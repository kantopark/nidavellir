package dkimage

import (
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

func Pull(name string) (string, error) {
	name = strings.TrimSpace(name)
	args := []string{"dkimage", "pull", name}
	cmd := exec.Command("docker", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "error encountered when pulling dkimage: %s", name)
	}

	return string(output), nil
}
