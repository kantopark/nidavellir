package config

import (
	"time"

	"github.com/pkg/errors"
)

type runConfig struct {
	MaxDuration time.Duration     `mapstructure:"max-duration"`
	BuildArgs   map[string]string `mapstructure:"build-args"`
}

func (r *runConfig) Validate() error {
	if r.MaxDuration.Nanoseconds() < 0 {
		return errors.Errorf("expected a non-negative duration but got %+v", r.MaxDuration)
	}

	return nil
}
