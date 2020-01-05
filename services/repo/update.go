package repo

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

func (r *Repo) Update() error {
	needsUpdate, err := r.needsToUpdate()
	if err != nil {
		return err
	}

	if !needsUpdate {
		return nil
	}

	if err := os.RemoveAll(r.WorkDir); err != nil {
		return errors.Wrap(err, "could not update repo. old repo cannot be removed")
	}

	if err := r.Clone(); err != nil {
		return err
	}

	return nil
}

func (r *Repo) needsToUpdate() (bool, error) {
	cmd := exec.Command("git", "ls-remote", "origin", "master", "|", "awk", "{ print $1 }")
	cmd.Dir = r.WorkDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, errors.Wrap(err, "could not fetch data from remote")
	}
	remoteHash := string(bytes.TrimSpace(output))

	cmd = exec.Command("git", "rev-parse", "master")
	cmd.Dir = r.WorkDir

	output, err = cmd.CombinedOutput()
	if err != nil {
		return false, errors.Wrap(err, "could not get local repo hash")
	}
	localHash := string(bytes.TrimSpace(output))

	return remoteHash != localHash, nil
}
