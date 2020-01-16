package dkcontainer

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/errors"

	"nidavellir/libs"
)

type RunOptions struct {
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
	Network string

	// These are not docker container run specs specifically

	// Working directory
	WorkDir string
}

func (o *RunOptions) imageTag() (string, error) {
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

func Run(options *RunOptions) (logs string, err error) {
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

	if !libs.IsEmptyOrWhitespace(options.Name) {
		args = append(args, "--name", options.Name)
	}

	if !libs.IsEmptyOrWhitespace(options.Network) {
		args = append(args, "--network", options.Network)
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
