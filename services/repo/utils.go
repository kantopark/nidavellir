package repo

import (
	"os/exec"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

// Checks if the runtime shell (environment) has the necessary shell executables to
// run the repo service
func SystemCheck() error {
	var errs error
	if _, err := exec.LookPath("docker"); err != nil {
		errs = multierror.Append(errs, errors.New("docker is required"))
	}
	if _, err := exec.LookPath("git"); err != nil {
		errs = multierror.Append(errs, errors.New("git is required"))
	}

	return errs
}
