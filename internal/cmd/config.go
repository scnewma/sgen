package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

func ConfigFile() (string, error) {
	d, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.hcl"), nil
}

func ConfigDir() (string, error) {
	if dir := os.Getenv("SGEN_CONFIG_DIR"); dir != "" {
		return dir, nil
	}

	dir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find HOME: %w", err)
	}
	return filepath.Join(dir, ".config", "sgen"), nil
}

type ConfigSource struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	File *struct {
		Path string `yaml:"path"`
	} `yaml:"file"`
	Command *string `yaml:"command"`
}
