package usecase

import (
	"context"

	"github.com/orurh/patchcourt/internal/platform/logx"
	graphusecase "github.com/orurh/patchcourt/internal/usecase/graph"
)

type GraphFormat = graphusecase.Format

const (
	GraphFormatJSON    = graphusecase.FormatJSON
	GraphFormatMermaid = graphusecase.FormatMermaid
	GraphFormatDOT     = graphusecase.FormatDOT
)

type GraphRequest = graphusecase.Request
type GraphResult = graphusecase.Result
type GraphService = graphusecase.Service

func NewGraphService(projects ProjectBuilder, logger logx.Logger) GraphService {
	return graphusecase.NewService(projects, logger)
}

func (a *App) RunGraph(ctx context.Context, req GraphRequest) (*GraphResult, error) {
	return a.graph.Run(ctx, req)
}
