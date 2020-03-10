package config

import (
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type Config struct {
	Acct accountConfig `mapstructure:"account"`
	App  AppConfig     `mapstructure:"app"`
	Run  runConfig     `mapstructure:"run"`
	Auth []AuthConfig  `mapstructure:"auth"`
}

type IValidate interface {
	Validate() error
}

func New() (*Config, error) {
	if err := setConfigDirectory(); err != nil {
		return nil, err
	}

	viper.SetConfigName("nida")
	viper.SetConfigType("yml")
	viper.SetEnvPrefix("nida")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, errors.Wrap(err, "could not read in configuration")
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal config")
	}

	for _, t := range []IValidate{
		&config.Acct,
		&config.App,
		&config.Run,
	} {
		if err := t.Validate(); err != nil {
			return nil, err
		}
	}

	return &config, nil
}

func setConfigDirectory() error {
	// Set os specific directory to store config file. This usually is reserved for
	// instances where we full write permissions on the machine
	switch runtime.GOOS {
	case "windows":
		viper.AddConfigPath("C:/kantopark/")
	case "linux":
		viper.AddConfigPath("/var/kantopark")

	default:
		return errors.Errorf("Unsupported platform: %s", runtime.GOOS)
	}

	// alternative path to store the config file. This is used in cases where the
	// user does not have full write permissions
	usr, err := user.Current()
	if err != nil {
		return errors.Wrap(err, "could not get user directory")
	}
	viper.AddConfigPath(filepath.Join(usr.HomeDir, "kantopark"))

	// set root directory where we're writing the program
	_, file, _, _ := runtime.Caller(0)
	workDir := filepath.Dir(filepath.Dir(file))
	viper.AddConfigPath(workDir)

	return nil
}
