package repo

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/errors"

	"nidavellir/config"
	"nidavellir/libs"
)

type Builder struct {
	WorkDir     string
	Image       string
	CommitTag   string
	BuildArgs   map[string]string
	RuntimeType string
}

func NewImageBuilder(imageName, commitTag, workDir, runtimeType string) (*Builder, error) {
	if !libs.PathExists(workDir) {
		return nil, errors.New("directory does not exist")
	}

	imageName = strings.TrimSpace(imageName)
	if imageName == "" {
		return nil, errors.New("image name cannot be empty")
	} else if strings.Contains(imageName, ":") {
		return nil, errors.New("image name should not contain ':'")
	}

	conf, err := config.New()
	if err != nil {
		return nil, err
	}

	return &Builder{
		WorkDir:     workDir,
		Image:       fmt.Sprintf("%s:%s", imageName, commitTag),
		CommitTag:   commitTag,
		BuildArgs:   conf.Image.BuildArgs,
		RuntimeType: runtimeType,
	}, nil
}

func (b *Builder) Build() (logs string, err error) {
	file, err := b.prepareDockerfile()
	if err != nil {
		return "", err
	}

	if file == "" {
		return "image is updated and thus not built", nil
	}

	logs, err = b.buildImage(file)
	if err != nil {
		return logs, err
	}

	return
}

func (b *Builder) buildImage(file string) (string, error) {
	args := []string{"image", "build", "-f", file, "--tag", b.Image}

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

// Based on the runtime setup type, generates a Dockerfile. If there are any changes to the
// Dockerfile, a build.Dockerfile will be created in the working repository. The presence of
// this file will cause the function to return "build.Dockerfile" which will then trigger an
// image build. If no changes, an empty string is returned which will not trigger any builds
func (b *Builder) prepareDockerfile() (string, error) {
	file, err := NewDockerfile(b.RuntimeType, b.WorkDir)
	if err != nil {
		return "", err
	}

	switch b.RuntimeType {
	case "dockerfile":
		if err := file.loadContent(); err != nil {
			return "", err
		}

		file.writeBuildArgs(b.BuildArgs)

	case "python", "r":
		if err := file.fetchFile(); err != nil {
			return "", err
		}

		file.writeBuildArgs(b.BuildArgs)
		if err := file.writeRequirements(); err != nil {
			return "", err
		}

	default:
		return "", errors.Errorf("unsupported runtime '%s'", b.RuntimeType)
	}

	if file.HasChanges {
		if err := file.createDockerfile(); err != nil {
			return "", err
		}
		return file.FilePath, nil
	}

	return "", nil
}

// Checks if the given image name exists
func ImageExists(imageName string) (bool, error) {
	args := []string{"image", "list", "--format", "{{.Repository}}"}
	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, errors.Wrap(err, "could not get list of docker images")
	}

	for _, name := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(name) == imageName {
			return true, nil
		}
	}
	return false, nil
}
