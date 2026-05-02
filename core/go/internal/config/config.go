package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

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

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config %s: %w", path, err)
	}

	return &cfg, nil
}

func (c Config) Validate() error {
	var errs []error

	if c.CPP.CompileCommands.Path != "" && c.CPP.CompileCommands.AutoDiscover {
		errs = append(errs, errors.New("cpp.compile_commands.path and cpp.compile_commands.auto_discover cannot both be set"))
	}

	errs = append(errs, validatePathList("ignore.paths", c.Ignore.Paths, true)...)
	errs = append(errs, validatePathList("cpp.include_paths", c.CPP.IncludePaths, true)...)
	errs = append(errs, validatePathList("cpp.system_include_paths", c.CPP.SystemIncludePaths, true)...)

	layerNames := make(map[string]struct{}, len(c.Layers))
	for layerName := range c.Layers {
		if strings.TrimSpace(layerName) == "" {
			errs = append(errs, errors.New("layers contains an empty layer name"))
			continue
		}

		layerNames[layerName] = struct{}{}
	}

	for layerName, layer := range c.Layers {
		if strings.TrimSpace(layerName) == "" {
			continue
		}

		if len(layer.Paths) == 0 {
			errs = append(errs, fmt.Errorf("layers.%s.paths must contain at least one path pattern", layerName))
		}

		errs = append(errs, validatePathList(fmt.Sprintf("layers.%s.paths", layerName), layer.Paths, false)...)
		errs = append(errs, validatePathList(fmt.Sprintf("layers.%s.exclude_paths", layerName), layer.ExcludePaths, true)...)

		for i, dependency := range layer.MayDependOn {
			if strings.TrimSpace(dependency) == "" {
				errs = append(errs, fmt.Errorf("layers.%s.may_depend_on[%d] must not be empty", layerName, i))
				continue
			}

			if _, ok := layerNames[dependency]; !ok {
				errs = append(errs, fmt.Errorf("layers.%s.may_depend_on references unknown layer %q", layerName, dependency))
			}
		}
	}

	return errors.Join(errs...)
}

func validatePathList(field string, values []string, allowEmptyList bool) []error {
	errs := make([]error, 0)

	if !allowEmptyList && len(values) == 0 {
		errs = append(errs, fmt.Errorf("%s must contain at least one path pattern", field))
		return errs
	}

	for i, value := range values {
		if strings.TrimSpace(value) == "" {
			errs = append(errs, fmt.Errorf("%s[%d] must not be empty", field, i))
		}
	}

	return errs
}
