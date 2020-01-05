package repo

import (
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

func (r *Repo) Clone() error {
	if _, err := os.Stat(r.WorkDir); !os.IsNotExist(err) {
		return errors.Errorf("repo '%s' already exists. cannot clone and overwrite", r.Source)
	}

	if err := os.MkdirAll(r.WorkDir, 0777); err != nil {
		return errors.Wrap(err, "could not create repo directory")
	}

	cmd := exec.Command("git", "clone", r.gitUrl(), ".")
	cmd.Dir = r.WorkDir

	if _, err := cmd.CombinedOutput(); err != nil {
		return err
	}

	return nil
}
