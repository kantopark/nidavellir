package config

type imageConfig struct {
	BuildArgs map[string]string `mapstructure:"build-args"`
}
