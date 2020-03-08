package config

type AuthConfig struct {
	Type string            `mapstructure:"type"`
	Info map[string]string `mapstructure:"info"`
}

var BasicAuth = AuthConfig{Type: "BASIC"}
