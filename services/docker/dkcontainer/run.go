package dkcontainer

import (
	"fmt"
	"os/exec"
	"regexp"
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

type RunResult struct {
	ExitCode int
	Logs     string
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

	if len(regexp.MustCompile(`\s`).Split(o.Name, -1)) > 1 {
		return "", errors.Errorf("Invalid container tag name '%s'", o.Name)
	}

	return fmt.Sprintf("%s:%s", o.Image, o.Tag), nil
}

func Run(options *RunOptions) (*RunResult, error) {
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

	if imageLog, err := options.imageTag(); err != nil {
		code, err := errorWithExitCode(err)

		return &RunResult{
			ExitCode: code,
			Logs:     imageLog,
		}, err
	} else {
		args = append(args, imageLog)
	}

	args = append(args, options.Cmd...)

	cmd := exec.Command("docker", args...)
	if !libs.IsEmptyOrWhitespace(options.WorkDir) && libs.PathExists(options.WorkDir) {
		cmd.Dir = options.WorkDir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		code, err := errorWithExitCode(err)

		return &RunResult{
			ExitCode: code,
			Logs:     string(output),
		}, err
	}

	return &RunResult{
		ExitCode: 0,
		Logs:     string(output),
	}, nil
}

func errorWithExitCode(err error) (int, error) {
	if err == nil {
		return 0, nil
	}

	if err, ok := err.(*exec.ExitError); ok {
		return err.ExitCode(), err
	} else {
		return 999, err // default exit code
	}
}
