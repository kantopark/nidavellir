package config

import (
	"os"
	"path/filepath"
	"runtime"
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
	return filepath.Join(w.Path, name)
}
