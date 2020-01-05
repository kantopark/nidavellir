package repo

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Repo struct {
	// Repo source url
	Source string
	// Set this to true if using gitlab
	Gitlab bool
	// Gitlab or Github token
	Token   string
	WorkDir string
}

func New(name string) (*Repo, error) {
	return &Repo{
		Source: name,
		Gitlab: os.Getenv("NIDA_GITLAB") == "1",
		Token:  os.Getenv("NIDA_TOKEN"),
	}, nil
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

func SystemCheck() error {
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("git is required")
	}
	return nil
}
