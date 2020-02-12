package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"nidavellir/libs"
)

type appConfig struct {
	WorkDir string `mapstructure:"workdir"`
	TLS     struct {
		KeyFile  string `mapstructure:"keyfile"`
		CertFile string `mapstructure:"certfile"`
	} `mapstructure:"tls"`
	Port int `mapstructure:"port"`
}

func (a *appConfig) Validate() error {
	a.WorkDir = strings.TrimSpace(a.WorkDir)
	if a.WorkDir == "" {
		switch runtime.GOOS {
		case "windows":
			a.WorkDir = "C:/temp/nidavellir"
		case "darwin", "linux":
			a.WorkDir = "/var/nidavellir"
		default:
			return errors.Errorf("unsupported platform: %s", runtime.GOOS)
		}
	}

	if !libs.PathExists(a.WorkDir) {
		if err := os.MkdirAll(a.WorkDir, 0777); err != nil {
			return errors.Errorf("could not create working directory at: %s", a.WorkDir)
		}
	}

	return nil
}

// Returns the path of the repository
func (a *appConfig) RepoPath(name string) string {
	name = libs.LowerTrimReplaceSpace(name)
	return filepath.Join(a.WorkDir, "repo", name)
}

func (a *appConfig) LogFilePath(name, runDate string) string {
	dir := a.createFolder("logs", name)
	return filepath.Join(dir, runDate+".log")
}

// Gets the image log file path. The path is uniquely defined by the image
// name and the image tag
func (a *appConfig) ImageBuildLogPath(name, tag string) string {
	dir := a.createFolder("image-logs", name)

	return filepath.Join(dir, tag+".log")
}

func (a *appConfig) OutputDir(jobId int) string {
	return a.createFolder("output", strconv.Itoa(jobId))
}

func (a *appConfig) createFolder(group, name string) string {
	name = libs.LowerTrimReplaceSpace(name)
	dir := filepath.Join(a.WorkDir, group, name)

	if !libs.PathExists(dir) {
		if err := os.MkdirAll(dir, 0777); err != nil {
			panic(err)
		}
	}

	return dir
}

// Checks if the application has TLS certificates
func (a *appConfig) HasCerts() bool {
	exists := func(filepath string) bool {
		filepath = strings.TrimSpace(filepath)
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			return false
		}

		return true
	}

	return exists(a.TLS.CertFile) && exists(a.TLS.KeyFile)
}
