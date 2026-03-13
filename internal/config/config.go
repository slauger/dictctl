package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultBackend string `yaml:"default_backend"`
	Language       string `yaml:"language"`
	Device         string `yaml:"device"`
	Backends       struct {
		Local struct {
			Model  string `yaml:"model"`
			Binary string `yaml:"binary"`
		} `yaml:"local"`
		OpenAI struct {
			APIKey string `yaml:"api_key"`
			Model  string `yaml:"model"`
		} `yaml:"openai"`
	} `yaml:"backends"`
}

func Load() (*Config, error) {
	cfg := &Config{
		DefaultBackend: "local",
		Language:       "en",
	}
	cfg.Backends.Local.Model = "large-v3-turbo"
	cfg.Backends.OpenAI.Model = "whisper-1"

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, nil
	}

	path := filepath.Join(home, ".config", "dictctl", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.DefaultBackend == "" {
		cfg.DefaultBackend = "local"
	}
	if cfg.Language == "" {
		cfg.Language = "en"
	}
	if cfg.Backends.Local.Model == "" {
		cfg.Backends.Local.Model = "large-v3-turbo"
	}
	if cfg.Backends.OpenAI.Model == "" {
		cfg.Backends.OpenAI.Model = "whisper-1"
	}

	return cfg, nil
}
