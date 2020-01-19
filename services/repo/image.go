package repo

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"nidavellir/config"
	"nidavellir/libs"
)

type Builder struct {
	WorkDir    string
	Image      string
	BuildArgs  map[string]string
	dockerfile string
}

func NewImageBuilder(image, workDir string) (*Builder, error) {
	if !libs.PathExists(workDir) {
		return nil, errors.New("directory does not exist")
	}

	conf, err := config.New()
	if err != nil {
		return nil, err
	}

	dockerfile := filepath.Join(workDir, "Dockerfile")
	if !libs.PathExists(dockerfile) {
		return nil, errors.New("dockerfile missing")
	}

	return &Builder{
		WorkDir:   workDir,
		Image:     image,
		BuildArgs: conf.Image.BuildArgs,
	}, nil
}

func (b *Builder) Build() (logs string, err error) {
	args := []string{"image", "build", "-f", b.dockerfile, "--tag", b.Image}

	for key, value := range b.BuildArgs {
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

// Checks if the image exists
func (b *Builder) ImageExists() (bool, error) {
	return ImageExists(b.Image)
}

// Checks if the given image name exists
func ImageExists(image string) (bool, error) {
	hasTag := strings.Contains(image, ":")

	args := []string{"image", "list", "--format"}
	if hasTag {
		args = append(args, "{{.Repository}}:{{.Tag}}")
	} else {
		args = append(args, "{{.Repository}}")
	}

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, errors.Wrap(err, "could not get list of docker images")
	}

	for _, name := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(name) == image {
			return true, nil
		}
	}
	return false, nil
}
