package repo

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"

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
	Token string
	// Repo's file path. The root of that file path will be the working directory
	WorkDir string

	// git commit to check out
	Commit string
	// Image name used by the repo. Should ideally contain the tags as well
	Image string
	// checks if the repo needs to build the image
	NeedsBuild bool

	Steps []*Step
}

// Creates a new repository given the source (remote gitlab or github url) and
// name (the unique identifier for the repo which will be used as the image name
// and file path). It is assumed that the Repo is already that latest (so please
// re-clone or pull the actual repo before calling NewRepo). If that is the case,
// the repo will then checkout any previous versions as specified in the
// runtime.yaml config file
func NewRepo(source, name string) (*Repo, error) {
	conf, err := config.New()
	if err != nil {
		return nil, err
	}

	workDir := conf.App.RepoPath(name)
	r := &Repo{
		Source:  source,
		Name:    libs.LowerTrimReplaceSpace(name),
		Gitlab:  os.Getenv("NIDA_GITLAB") == "1",
		Token:   os.Getenv("NIDA_TOKEN"),
		WorkDir: workDir,
	}

	if !r.Exists() {
		err := r.Clone()
		if err != nil {
			return nil, err
		}
	}

	err = r.formatRuntimeConfig(workDir)
	if err != nil {
		return nil, err
	}

	// Checkout repo
	if err := r.Checkout(); err != nil {
		return nil, errors.Wrapf(err, "could not checkout '%s' for repo", r.Commit)
	}

	return r, nil
}

// Clones the repo if it does not exists. If repo exists, checks if it is outdated. If repo is outdated,
// remove original repo and clone it again (thus force updating it)
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

// Checks if the repository exists in the filepath. If it doesn't, it should lead to
// a repo.Clone
func (r *Repo) Exists() bool {
	return libs.PathExists(r.WorkDir)
}

// Checks if the image required by the repository exists
func (r *Repo) HasImage() (bool, error) {
	return ImageExists(r.Image)
}

// Attempts to pull the image
func (r *Repo) PullImage() (string, error) {
	cmd := exec.Command("docker", "image", "pull", r.Image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, string(output))
	}
	return string(output), nil
}

// Builds the image for the repository given the rSetup instructions from
// the runtime config
func (r *Repo) BuildImage() (string, error) {
	b, err := NewImageBuilder(r.Image, r.WorkDir)
	if err != nil {
		return "", err
	}

	if exists, err := ImageExists(r.Image); err != nil {
		return "", err
	} else if exists {
		return "", err
	}

	logs, err := b.Build()
	if err != nil {
		return "", err
	}
	return logs, nil
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

func (r *Repo) Checkout() error {
	masterHash, err := r.getCommitHash("master")
	if err != nil {
		return err
	}

	if masterHash == r.Commit {
		// no changes to the commit. master is same as tag
		return nil
	}

	// master hash not equal to commit hash. Checkout commit
	cmd := exec.Command("git", "checkout", r.Commit)
	cmd.Dir = r.WorkDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "could not checkout '%s'", r.Commit)
	}

	logs := strings.TrimSpace(string(output))
	if strings.HasPrefix(logs, "error") {
		return errors.Wrapf(err, "could not checkout '%s'", r.Commit)
	}

	return nil
}

func (r *Repo) getCommitHash(commit string) (string, error) {
	cmd := exec.Command("git", "rev-parse", commit)
	cmd.Dir = r.WorkDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, "could not rev-parse to get commit hash")
	}

	return strings.TrimSpace(string(output)), nil
}
