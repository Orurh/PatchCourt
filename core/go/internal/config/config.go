package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Ignore IgnoreConfig           `yaml:"ignore"`
	CPP    CPPConfig              `yaml:"cpp"`
	Layers map[string]LayerConfig `yaml:"layers"`
}

type IgnoreConfig struct {
	Paths []string `yaml:"paths"`
}

type CPPConfig struct {
	CompileCommands    CompileCommandsConfig `yaml:"compile_commands"`
	IncludePaths       []string              `yaml:"include_paths"`
	SystemIncludePaths []string              `yaml:"system_include_paths"`
}

type CompileCommandsConfig struct {
	Path         string `yaml:"path"`
	AutoDiscover bool   `yaml:"auto_discover"`
}

type LayerConfig struct {
	Paths        []string `yaml:"paths"`
	ExcludePaths []string `yaml:"exclude_paths"`
	MayDependOn  []string `yaml:"may_depend_on"`
}

func Load(path string) (*Config, error) {
	if path == "" {
		return &Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	return &cfg, nil
}
