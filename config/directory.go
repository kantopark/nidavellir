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

type workDirConfig struct {
	Path string `mapstructure:"path"`
}

func (w *workDirConfig) Validate() error {
	w.Path = strings.TrimSpace(w.Path)
	if w.Path == "" {
		switch runtime.GOOS {
		case "windows":
			w.Path = "C:/temp/nidavellir"
		case "darwin", "linux":
			w.Path = "/var/nidavellir"
		default:
			return errors.Errorf("unsupported platform: %s", runtime.GOOS)
		}
	}

	if !libs.PathExists(w.Path) {
		if err := os.MkdirAll(w.Path, 0777); err != nil {
			return errors.Errorf("could not create working directory at: %s", w.Path)
		}
	}
	return nil
}

// Returns the path of the repository
func (w *workDirConfig) RepoPath(name string) string {
	name = libs.LowerTrimReplaceSpace(name)
	return filepath.Join(w.Path, "repo", name)
}

func (w *workDirConfig) LogFilePath(name, runDate string) string {
	dir := w.createFolder("logs", name)
	return filepath.Join(dir, runDate+".log")
}

// Gets the image log file path. The path is uniquely defined by the image
// name and the image tag
func (w *workDirConfig) ImageBuildLogPath(name, tag string) string {
	dir := w.createFolder("image-logs", name)

	return filepath.Join(dir, tag+".log")
}

func (w *workDirConfig) OutputDir(jobId int) string {
	return w.createFolder("output", strconv.Itoa(jobId))
}

func (w *workDirConfig) createFolder(group, name string) string {
	name = libs.LowerTrimReplaceSpace(name)
	dir := filepath.Join(w.Path, group, name)

	if !libs.PathExists(dir) {
		if err := os.MkdirAll(dir, 0777); err != nil {
			panic(err)
		}
	}

	return dir
}
