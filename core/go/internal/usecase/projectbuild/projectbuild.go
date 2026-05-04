package projectbuild

import (
	"context"

	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/engine"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/usecase/ports"
)

type Request struct {
	Operation  string
	Root       string
	ConfigPath string
}

type Result struct {
	Project *model.ProjectModel
	Config  *config.Config
}

type Builder struct {
	Analysis ports.AnalysisService
}

func New(analysis ports.AnalysisService) Builder {
	return Builder{
		Analysis: analysis,
	}
}

func (b Builder) Build(ctx context.Context, req Request) (*Result, error) {
	result, err := b.Analysis.Analyze(ctx, engine.AnalyzeRequest{
		Operation:  req.Operation,
		Root:       req.Root,
		ConfigPath: req.ConfigPath,
	})
	if err != nil {
		return nil, err
	}

	return &Result{
		Project: result.Project,
		Config:  result.Config,
	}, nil
}
