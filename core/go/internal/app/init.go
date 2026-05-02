package app

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/analysis/discovery"
	"github.com/orurh/patchcourt/internal/platform/logx"
)

type InitRequest struct {
	Root   string
	Strict bool
}

type InitResult struct {
	ConfigYAML string
}

func (a *App) RunInit(ctx context.Context, req InitRequest) (*InitResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("init canceled before start: %w", err)
	}

	logger := a.logger.With(
		logx.String("operation", "init"),
		logx.String("root", req.Root),
	)

	logger.Debug("discovering project architecture")

	result, err := discovery.GenerateInitConfig(discovery.InitOptions{
		Root:   req.Root,
		Strict: req.Strict,
	})
	if err != nil {
		return nil, fmt.Errorf("generate init config: %w", err)
	}

	logger.Debug("init config generated")

	return &InitResult{
		ConfigYAML: result.ConfigYAML,
	}, nil
}
