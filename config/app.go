package config

import (
	"os"
	"runtime"
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
