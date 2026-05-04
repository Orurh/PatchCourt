package init

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/orurh/patchcourt/internal/analyzer/discovery"
	"github.com/orurh/patchcourt/internal/platform/files"
	"github.com/orurh/patchcourt/internal/platform/logx"
)

type Request struct {
	Root       string
	Strict     bool
	Preset     string
	Write      bool
	Force      bool
	OutputPath string
}

type Result struct {
	ConfigYAML string
	ConfigPath string
	Written    bool
}

type Service struct {
	Logger logx.Logger
}

func NewService(logger logx.Logger) Service {
	if logger == nil {
		logger = logx.Nop()
	}

	return Service{
		Logger: logger,
	}
}

func (s Service) Run(ctx context.Context, req Request) (*Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("init canceled before start: %w", err)
	}

	root := req.Root
	if root == "" {
		root = "."
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	logger := s.Logger.With(
		logx.String("operation", "init"),
		logx.String("root", absRoot),
		logx.String("preset", req.Preset),
	)

	logger.Debug("discovering project architecture")

	result, err := discovery.GenerateInitConfig(discovery.InitOptions{
		Root:   absRoot,
		Strict: req.Strict,
		Preset: req.Preset,
	})
	if err != nil {
		return nil, fmt.Errorf("generate init config: %w", err)
	}

	initResult := &Result{
		ConfigYAML: result.ConfigYAML,
	}

	if !req.Write {
		logger.Debug("init config generated")
		return initResult, nil
	}

	outputPath := req.OutputPath
	if outputPath == "" {
		outputPath = filepath.Join(absRoot, ".patchcourt.yaml")
	}

	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return nil, fmt.Errorf("resolve output path: %w", err)
	}

	if !req.Force {
		if _, err := os.Stat(absOutputPath); err == nil {
			return nil, fmt.Errorf("config already exists: %s. Use --force to overwrite", absOutputPath)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat config path %s: %w", absOutputPath, err)
		}
	}

	if err := files.WriteFileAtomic(absOutputPath, []byte(result.ConfigYAML), 0o644); err != nil {
		return nil, fmt.Errorf("write config %s: %w", absOutputPath, err)
	}

	logger.Debug("init config written")

	initResult.ConfigPath = absOutputPath
	initResult.Written = true

	return initResult, nil
}
