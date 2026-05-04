package source

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/engine"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/state"
)

type Analyzer interface {
	Analyze(ctx context.Context, req engine.AnalyzeRequest) (*engine.AnalyzeResult, error)
}

type ProjectModelSource interface {
	Label() string
	Load(ctx context.Context) (*model.ProjectModel, error)
}

type SourcePair struct {
	Before ProjectModelSource
	After  ProjectModelSource
}

type SnapshotSource struct {
	Path string
}

type StateSource struct {
	Root string
	Name string
}

type RootSource struct {
	Root       string
	ConfigPath string
	Operation  string
	Analyzer   Analyzer
}

func LoadPair(ctx context.Context, pair SourcePair) (*model.ProjectModel, *model.ProjectModel, error) {
	if pair.Before == nil {
		return nil, nil, fmt.Errorf("before project model source is required")
	}

	if pair.After == nil {
		return nil, nil, fmt.Errorf("after project model source is required")
	}

	beforeProject, err := pair.Before.Load(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("load before project model from %s: %w", pair.Before.Label(), err)
	}

	afterProject, err := pair.After.Load(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("load after project model from %s: %w", pair.After.Label(), err)
	}

	return beforeProject, afterProject, nil
}

func (s SnapshotSource) Label() string {
	if s.Path == "" {
		return "snapshot:<empty>"
	}

	return "snapshot:" + s.Path
}

func (s SnapshotSource) Load(ctx context.Context) (*model.ProjectModel, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if s.Path == "" {
		return nil, fmt.Errorf("snapshot path is required")
	}

	return state.ReadProjectModel(s.Path)
}

func (s StateSource) Label() string {
	root := s.Root
	if root == "" {
		root = "."
	}

	name := s.Name
	if name == "" {
		name = state.LatestStateName
	}

	return "state:" + state.StateDir(root, name)
}

func (s StateSource) Load(ctx context.Context) (*model.ProjectModel, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	state, err := state.LoadState(state.LoadStateOptions{
		Root: s.Root,
		Name: s.Name,
	})
	if err != nil {
		return nil, err
	}

	return state.Project, nil
}

func (s RootSource) Label() string {
	root := s.Root
	if root == "" {
		root = "."
	}

	return "root:" + root
}

func (s RootSource) Load(ctx context.Context) (*model.ProjectModel, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if s.Analyzer == nil {
		return nil, fmt.Errorf("analyzer is required")
	}

	root := s.Root
	if root == "" {
		root = "."
	}

	operation := s.Operation
	if operation == "" {
		operation = "review-root"
	}

	result, err := s.Analyzer.Analyze(ctx, engine.AnalyzeRequest{
		Operation:  operation,
		Root:       root,
		ConfigPath: s.ConfigPath,
	})
	if err != nil {
		return nil, err
	}

	if result == nil || result.Project == nil {
		return nil, fmt.Errorf("analyzer returned empty project model")
	}

	return result.Project, nil
}
