package repo

import (
	"os/exec"
	"path/filepath"
	"runtime"

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

func repoDir(name string) (string, error) {
	var dir string
	switch runtime.GOOS {
	case "windows":
		dir = "C:/temp/nidavellir"
	case "darwin", "linux":
		dir = "/var/nidavellir"
	default:
		return "", errors.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return filepath.Join(dir, name), nil
}
