package docker

import (
	"fmt"
	"os/exec"

	"github.com/pkg/errors"
)

type Image struct {
	Tag string
}

type ImageBuildOptions struct {
	BuildArgs map[string]string
	Path      string
	WorkDir   string
	Tag       string
}

func (i *Image) Build(options *ImageBuildOptions) (logs string, err error) {
	args := []string{"image", "build", "--tag", options.Tag}

	for key, value := range options.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	args = append(args, ".")

	cmd := exec.Command("docker", args...)
	cmd.Dir = options.WorkDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, "error when building image")
	}

	return string(output), nil
}

func (i *Image) Exists(name, tag string) (bool, error) {
	cmd := exec.Command("docker", "image", "list", "--format", "{{.Repository}}:{{.Tag}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, errors.Wrap(err, "could not get list of images")
	}

	searchValue := fmt.Sprintf("%s:%s", name, tag)
	for _, imageTag := range splitOutput(output) {
		if searchValue == imageTag {
			return true, nil
		}
	}

	return false, nil
}
