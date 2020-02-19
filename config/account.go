package config

import (
	"github.com/pkg/errors"

	"nidavellir/libs"
)

type accountConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

func (a *accountConfig) Validate() error {
	a.Username = libs.LowerTrim(a.Username)
	if a.Username == "" {
		return errors.New("admin username cannot be empty")
	}

	a.Password = libs.LowerTrim(a.Password)
	if a.Password == "" {
		return errors.New("admin password cannot be empty")
	}

	return nil
}
