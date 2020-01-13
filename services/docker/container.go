package docker

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"nidavellir/libs"
)

type Container struct {
}

func NewContainer() *Container {
	return &Container{}
}

type ContainerRunOptions struct {
	Image string
	Tag   string
	Name  string
	// see https://docs.docker.com/engine/reference/commandline/run/#restart-policies---restart
	Restart string
	Env     map[string]string
	Cmd     []string
	Ports   map[int]int
	Volumes map[string]string
	Daemon  bool

	// These are not docker container run specs specifically

	// Working directory
	WorkDir string
}

func (o *ContainerRunOptions) imageTag() (string, error) {
	o.Image = strings.TrimSpace(o.Image)
	o.Tag = strings.TrimSpace(o.Tag)
	if o.Image == "" {
		return "", errors.New("Image not specified")
	}

	if o.Tag == "" {
		return o.Image, nil
	}
	return fmt.Sprintf("%s:%s", o.Image, o.Tag), nil
}

func (c *Container) Run(options *ContainerRunOptions) (logs string, err error) {
	args := []string{"container", "run"}

	if options.Daemon {
		args = append(args, "-d")
	}

	for key, value := range options.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	for src, dest := range options.Volumes {
		args = append(args, "-v", fmt.Sprintf("%s:%s", src, dest))
	}

	for host, target := range options.Ports {
		args = append(args, "-p", fmt.Sprintf("%d:%d", host, target))
	}

	if name := strings.TrimSpace(options.Name); name != "" {
		args = append(args, "--name", name)
	}

	if restart := strings.TrimSpace(options.Restart); restart == "" {
		args = append(args, "--restart", "unless-stopped")
	} else {
		args = append(args, "--restart", restart)
		if restart == "no" {
			args = append(args, "--rm")
		}
	}

	if image, err := options.imageTag(); err != nil {
		return "", err
	} else {
		args = append(args, image)
	}

	args = append(args, options.Cmd...)

	cmd := exec.Command("docker", args...)
	if !libs.IsEmptyOrWhitespace(options.WorkDir) && libs.PathExists(options.WorkDir) {
		cmd.Dir = options.WorkDir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, "error when running container")
	}

	return string(output), nil
}

type ContainerStopOptions struct {
	Name string
	Port int
}

func (c *Container) Stop(options *ContainerStopOptions) (logs string, err error) {
	containers, err := c.Search(&ContainerSearchOptions{
		Name: options.Name,
		Port: options.Port,
	})
	if err != nil {
		return "", errors.Wrap(err, "could not find any containers to stop")
	}

	for _, id := range containers {
		cmd := exec.Command("docker", "container", "rm", "-f", id)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", errors.Wrapf(err, "could not stop container '%s'", id)
		}
		logs += string(output) + "\n"
	}
	return
}

type ContainerSearchOptions struct {
	Name string
	Port int
}

func (c *Container) Search(options *ContainerSearchOptions) ([]string, error) {
	sep := "::"
	cmd := exec.Command("docker", "container", "list", "-a", "--format", "{{.Names}}"+sep+"{{.Ports}}"+sep+"{{.ID}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "could not get list of containers")
	}

	idMap := make(map[string]int)
	for _, namesPorts := range splitOutput(output) {
		parts := strings.Split(namesPorts, "::")
		id := parts[2]

		for _, name := range strings.Split(parts[0], ",") {
			if options.Name != "" && name == options.Name {
				idMap[id] = 0
			}
		}

		for _, fullAddress := range strings.Split(parts[1], ",") {
			if addresses := strings.Split(fullAddress, "->"); len(addresses) == 2 {
				if _parts := strings.Split(addresses[0], ":"); len(_parts) == 2 {
					if port, err := strconv.Atoi(_parts[1]); err == nil && options.Port == port {
						idMap[id] = 0
					}
				}
			}
		}
	}

	var ids []string
	for k := range idMap {
		ids = append(ids, k)
	}

	return ids, nil
}
