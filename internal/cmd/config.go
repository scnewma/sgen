package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var ConfigNotFound = errors.New("config not found")

func LoadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not find HOME: %w", err)
	}
	path := filepath.Join(homeDir, ".config", "sgen", "config.yaml")

	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(contents, &cfg); err != nil {
		return nil, fmt.Errorf("decoding file %q: %w", path, err)
	}
	return &cfg, nil
}

type Config struct {
	Sources []ConfigSource `yaml:"sources"`
}

type ConfigSource struct {
	Name            string `yaml:"name"`
	DefaultTemplate string `yaml:"default_template"`
	Type            string `yaml:"type"`
	File            *struct {
		Path string `yaml:"path"`
	} `yaml:"file"`
	Command *string `yaml:"command"`
}

func (c *Config) GetSource(name string) *ConfigSource {
	for _, cs := range c.Sources {
		if cs.Name == name {
			return &cs
		}
	}
	return nil
}
