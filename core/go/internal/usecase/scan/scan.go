package scan

import (
	"context"

	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/usecase/projectbuild"
)

type Format string

const (
	FormatText     Format = "text"
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
)

type Request struct {
	Root       string
	ConfigPath string
	Format     Format
}

type Result struct {
	Project *model.ProjectModel
	Config  *config.Config
}

type Service struct {
	Projects projectbuild.Builder
}

func NewService(projects projectbuild.Builder) Service {
	return Service{
		Projects: projects,
	}
}

func (s Service) Run(ctx context.Context, req Request) (*Result, error) {
	result, err := s.Projects.Build(ctx, projectbuild.Request{
		Operation:  "scan",
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
