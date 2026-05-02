package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate_AllowsEmptyConfig(t *testing.T) {
	require.NoError(t, Config{}.Validate())
}

func TestConfigValidate_RejectsCompileCommandsPathAndAutoDiscoverTogether(t *testing.T) {
	cfg := Config{
		CPP: CPPConfig{
			CompileCommands: CompileCommandsConfig{
				Path:         "build/compile_commands.json",
				AutoDiscover: true,
			},
		},
	}

	err := cfg.Validate()

	require.Error(t, err)
	require.Contains(t, err.Error(), "cpp.compile_commands.path")
}

func TestConfigValidate_RejectsLayerWithoutPaths(t *testing.T) {
	cfg := Config{
		Layers: map[string]LayerConfig{
			"api": {},
		},
	}

	err := cfg.Validate()

	require.Error(t, err)
	require.Contains(t, err.Error(), "layers.api.paths")
}

func TestConfigValidate_RejectsUnknownLayerDependency(t *testing.T) {
	cfg := Config{
		Layers: map[string]LayerConfig{
			"api": {
				Paths:       []string{"src/server/**"},
				MayDependOn: []string{"domain"},
			},
		},
	}

	err := cfg.Validate()

	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown layer "domain"`)
}

func TestConfigValidate_RejectsEmptyPatternsAndDependencies(t *testing.T) {
	cfg := Config{
		Ignore: IgnoreConfig{
			Paths: []string{""},
		},
		CPP: CPPConfig{
			IncludePaths: []string{"src", " "},
		},
		Layers: map[string]LayerConfig{
			"api": {
				Paths:        []string{"src/server/**"},
				ExcludePaths: []string{""},
				MayDependOn:  []string{""},
			},
		},
	}

	err := cfg.Validate()

	require.Error(t, err)
	require.Contains(t, err.Error(), "ignore.paths[0]")
	require.Contains(t, err.Error(), "cpp.include_paths[1]")
	require.Contains(t, err.Error(), "layers.api.exclude_paths[0]")
	require.Contains(t, err.Error(), "layers.api.may_depend_on[0]")
}

func TestLoad_ValidatesConfig(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, ".patchcourt.yaml")

	err := os.WriteFile(configPath, []byte(`
layers:
  api:
    paths:
      - src/server/**
    may_depend_on:
      - missing
`), 0o644)
	require.NoError(t, err)

	_, err = Load(configPath)

	require.Error(t, err)
	require.Contains(t, err.Error(), "validate config")
	require.Contains(t, err.Error(), `unknown layer "missing"`)
}

func TestLoad_ValidConfig(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, ".patchcourt.yaml")

	err := os.WriteFile(configPath, []byte(`
ignore:
  paths:
    - build/**

cpp:
  compile_commands:
    auto_discover: true
  include_paths:
    - src

layers:
  api:
    paths:
      - src/server/**
    may_depend_on:
      - domain

  domain:
    paths:
      - src/domain/**
    may_depend_on: []
`), 0o644)
	require.NoError(t, err)

	cfg, err := Load(configPath)

	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Contains(t, cfg.Layers, "api")
	require.Contains(t, cfg.Layers, "domain")
}
