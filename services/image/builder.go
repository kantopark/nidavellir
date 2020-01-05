package image

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

type Builder struct {
	WorkDir   string
	CommitTag string
}

func NewBuilder(dir, commitTag string) (*Builder, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, errors.Wrap(err, "directory does not exist")
	}

	return &Builder{
		WorkDir:   dir,
		CommitTag: commitTag,
	}, nil
}

func (b *Builder) Build(buildArgs map[string]string) (logs string, err error) {
	gitLog, err := checkout(b.commitTag())
	if err != nil {
		return gitLog, err
	}

	file, err := prepareDockerfile(b.WorkDir)
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
	tag, err := b.imageTag()
	if err != nil {
		return "", errors.Wrap(err, "could not generate image tag")
	}

	args := []string{"image", "build", "-f", file, "--tag", tag}

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
