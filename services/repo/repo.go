package repo

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"nidavellir/config"
	"nidavellir/libs"
)

type Repo struct {
	// Repo source url
	Source string
	// Repo name
	Name string
	// Set this to true if using gitlab
	Gitlab bool
	// Gitlab or Github token
	Token   string
	WorkDir string
}

func NewRepo(source, name string) (*Repo, error) {
	conf, err := config.New()
	if err != nil {
		return nil, err
	}

	return &Repo{
		Source:  source,
		Name:    libs.LowerTrimReplaceSpace(name),
		Gitlab:  os.Getenv("NIDA_GITLAB") == "1",
		Token:   os.Getenv("NIDA_TOKEN"),
		WorkDir: conf.WorkDir.RepoPath(name),
	}, nil
}

func (r *Repo) Clone() error {
	if r.Exists() {
		if update, err := r.needsToUpdate(); err != nil {
			return err
		} else if !update {
			return nil
		}

		if err := os.RemoveAll(r.WorkDir); err != nil {
			return errors.Wrap(err, "could not update repo. old repo cannot be removed")
		}
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

// Checks if the repository exists
func (r *Repo) Exists() bool {
	return libs.PathExists(r.WorkDir)
}

func (r *Repo) Runtime() (*Runtime, error) {
	if !r.Exists() {
		return nil, errors.New("directory does not exist. could not get runtime configuration")
	}

	return RuntimeFromDir(r.WorkDir)
}

func (r *Repo) BuildImage() error {
	conf, err := r.Runtime()
	if err != nil {
		return err
	}
	b, err := NewImageBuilder(r.Name, r.WorkDir, conf)
	if err != nil {
		return err
	}

	if exists, err := b.ImageExists(); err != nil {
		return err
	} else if exists {
		return nil
	}

	logs, err := b.Build()
	if err != nil {
		return err
	}
	log.Print(logs)
	return nil
}

func (r *Repo) gitUrl() string {
	token := strings.TrimSpace(r.Token)
	if token == "" {
		return r.Source
	}

	parts := strings.Split(r.Source, "://")
	schema := parts[0]
	path := parts[1]

	if r.Gitlab {
		return fmt.Sprintf("%s://gitlab-ci-token:%s@%s", schema, token, path)
	} else {
		return fmt.Sprintf("%s://%s@%s", schema, token, path)
	}
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
