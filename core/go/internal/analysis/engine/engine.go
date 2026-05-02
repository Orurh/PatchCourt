package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/orurh/patchcourt/internal/analysis/compilecmds"
	"github.com/orurh/patchcourt/internal/analysis/project"
	"github.com/orurh/patchcourt/internal/analysis/resolver"
	"github.com/orurh/patchcourt/internal/analysis/rules"
	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/logx"
)

type Engine struct {
	logger logx.Logger
	rules  []rules.Rule
}

type Options struct {
	Logger logx.Logger
	Rules  []rules.Rule
}

type AnalyzeRequest struct {
	Operation  string
	Root       string
	ConfigPath string
}

type AnalyzeResult struct {
	Project *model.ProjectModel
	Config  *config.Config
}

func New(opts Options) *Engine {
	logger := opts.Logger
	if logger == nil {
		logger = logx.Nop()
	}

	ruleSet := opts.Rules
	if len(ruleSet) == 0 {
		ruleSet = rules.DefaultRules()
	}

	return &Engine{
		logger: logger,
		rules:  ruleSet,
	}
}

func (e *Engine) Analyze(ctx context.Context, req AnalyzeRequest) (*AnalyzeResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%s canceled before start: %w", req.Operation, err)
	}

	logger := e.logger.With(
		logx.String("operation", req.Operation),
		logx.String("root", req.Root),
		logx.String("config_path", req.ConfigPath),
	)

	logger.Debug("loading config")

	cfg, err := config.Load(req.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logger.Debug("building project model")

	includePaths, err := e.resolveCPPIncludePaths(req.Root, cfg)
	if err != nil {
		return nil, err
	}

	projectModel, err := project.Build(project.Options{
		Root:            req.Root,
		IgnorePaths:     cfg.Ignore.Paths,
		CPPIncludePaths: includePaths,
	})
	if err != nil {
		return nil, fmt.Errorf("build project model: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%s canceled after project indexing: %w", req.Operation, err)
	}

	logger.Debug(
		"applying rules",
		logx.Int("files", len(projectModel.Files)),
		logx.Int("dependencies", len(projectModel.Dependencies)),
	)

	rules.Apply(projectModel, cfg, e.rules)

	logger.Debug("analysis completed", logx.Int("findings", len(projectModel.Findings)))

	return &AnalyzeResult{
		Project: projectModel,
		Config:  cfg,
	}, nil
}

func (e *Engine) resolveCPPIncludePaths(root string, cfg *config.Config) ([]resolver.IncludePath, error) {
	includePaths := make([]resolver.IncludePath, 0)
	compileCommandsPath := resolveCompileCommandsPath(root, cfg)
	if compileCommandsPath != "" {
		db, err := compilecmds.Load(compileCommandsPath)
		if err != nil {
			return nil, err
		}

		includePaths = appendIncludePaths(
			includePaths,
			compilecmds.IncludePaths(db, root),
			model.ResolutionSourceCompileCommands,
			model.ResolutionConfidenceHigh,
		)

		e.logger.Debug(
			"loaded compile commands include paths",
			logx.String("compile_commands", compileCommandsPath),
			logx.Int("include_paths", len(includePaths)),
		)
	}

	includePaths = appendIncludePaths(
		includePaths,
		cfg.CPP.IncludePaths,
		model.ResolutionSourceConfig,
		model.ResolutionConfidenceHigh,
	)

	return uniqueIncludePaths(includePaths), nil
}

func resolveCompileCommandsPath(root string, cfg *config.Config) string {
	configuredPath := cfg.CPP.CompileCommands.Path
	if configuredPath != "" {
		if filepath.IsAbs(configuredPath) {
			return configuredPath
		}

		return filepath.Join(root, configuredPath)
	}

	if !cfg.CPP.CompileCommands.AutoDiscover {
		return ""
	}

	candidates := []string{
		filepath.Join(root, "compile_commands.json"),
		filepath.Join(root, "build", "compile_commands.json"),
	}

	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate
		}
	}

	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func appendIncludePaths(
	dst []resolver.IncludePath,
	paths []string,
	source model.ResolutionSource,
	confidence model.ResolutionConfidence,
) []resolver.IncludePath {
	for _, path := range paths {
		dst = append(dst, resolver.IncludePath{
			Path:       path,
			Source:     source,
			Confidence: confidence,
		})
	}

	return dst
}

func uniqueIncludePaths(values []resolver.IncludePath) []resolver.IncludePath {
	seen := make(map[string]struct{}, len(values))
	result := make([]resolver.IncludePath, 0, len(values))

	for _, value := range values {
		if value.Path == "" {
			continue
		}

		if _, ok := seen[value.Path]; ok {
			continue
		}

		seen[value.Path] = struct{}{}
		result = append(result, value)
	}

	return result
}
