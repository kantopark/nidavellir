package repo

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"nidavellir/libs"
)

type Builder struct {
	WorkDir   string
	Image     string
	CommitTag string
	Runtime   *Runtime
}

func NewImageBuilder(name, workDir string, runtime *Runtime) (*Builder, error) {
	if !libs.PathExists(workDir) {
		return nil, errors.New("directory does not exist")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("image name cannot be empty")
	} else if strings.Contains(name, ":") {
		return nil, errors.New("image name should not contain ':'")
	}

	commitTag, err := runtime.CommitTag(workDir)
	if err != nil {
		return nil, err
	}

	return &Builder{
		WorkDir:   workDir,
		Image:     fmt.Sprintf("%s:%s", name, commitTag),
		CommitTag: commitTag,
	}, nil
}

func (b *Builder) Build(buildArgs map[string]string) (logs string, err error) {
	gitLog, err := checkout(b.CommitTag)
	if err != nil {
		return gitLog, err
	}

	file, err := b.prepareDockerfile()
	if err != nil {
		return "", err
	}

	if file == "" {
		return "image is updated and thus not built", nil
	}

	buildLog, err := b.buildImage(file, buildArgs)
	logs = gitLog + "\n\n\n" + buildLog
	if err != nil {
		return logs, err
	}

	return
}

func (b *Builder) buildImage(file string, buildArgs map[string]string) (string, error) {
	args := []string{"image", "build", "-f", file, "--tag", b.Image}

	for key, value := range buildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	args = append(args, ".")

	cmd := exec.Command("docker", args...)
	cmd.Dir = b.WorkDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, "error when building image")
	}

	return string(output), nil
}

func checkout(commit string) (string, error) {
	cmd := exec.Command("git", "checkout", commit)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "could not checkout '%s'", commit)
	}

	logs := strings.TrimSpace(string(output))
	if strings.HasPrefix(logs, "error") {
		return "", errors.Wrapf(err, "could not checkout '%s'", commit)
	}

	return logs, nil
}

func (b *Builder) ImageExists() (bool, error) {
	args := []string{"image", "list", "--format", "{{.Repository}}"}
	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, errors.Wrap(err, "could not get list of docker images")
	}

	for _, name := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(name) == b.Image {
			return true, nil
		}
	}
	return false, nil
}

func (b *Builder) prepareDockerfile() (string, error) {
	switch t := b.Runtime.Setup.Type; t {
	case "dockerfile":
		fp := filepath.Join(b.WorkDir, "Dockerfile")
		if !libs.PathExists(fp) {
			return "", errors.New("dockerfile missing")
		}

		return filepath.Join(b.WorkDir, "Dockerfile"), nil
	case "python", "r":
		file, err := newDockerfile(t, b.WorkDir)
		if err != nil {
			return "", err
		}
		if err := file.fetchFile(); err != nil {
			return "", err
		}

		if err := file.writeRequirements(); err != nil {
			return "", err
		}

		if file.HasChanges {
			if err := file.createDockerfile(); err != nil {
				return "", err
			}
			return file.FilePath, nil
		}

		return "", nil
	default:
		return "", errors.Errorf("unsupported runtime '%s'", t)
	}
}
