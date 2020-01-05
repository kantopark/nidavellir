package image

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

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

func (b *Builder) commitTag() string {
	commit := strings.TrimSpace(b.CommitTag)
	if commit == "" {
		return "master"
	}

	return commit
}

func (b *Builder) latestHash() (string, error) {
	cmd := exec.Command("git", "rev-parse", "master")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, "could not get latest hash from repo")
	}
	return strings.TrimSpace(string(output)), nil
}

func (b *Builder) imageTag() (string, error) {
	if strings.TrimSpace(b.CommitTag) == "" {
		commit, err := b.latestHash()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s:%s", filepath.Base(b.WorkDir), commit), nil
	}

	return fmt.Sprintf("%s:%s", filepath.Base(b.WorkDir), strings.TrimSpace(b.CommitTag)), nil
}
