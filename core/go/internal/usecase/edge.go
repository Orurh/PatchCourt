package usecase

import (
	"context"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	edgeusecase "github.com/orurh/patchcourt/internal/usecase/edge"
)

type EdgeResult = reportmodel.EdgeResult
type EdgeUsageSummary = reportmodel.EdgeUsageSummary
type EdgeFileCount = reportmodel.EdgeFileCount

type EdgeFormat = edgeusecase.Format

const (
	EdgeFormatText = edgeusecase.FormatText
	EdgeFormatJSON = edgeusecase.FormatJSON
)

type EdgeRequest = edgeusecase.Request
type EdgeService = edgeusecase.Service

func NewEdgeService(projects ProjectBuilder) EdgeService {
	return edgeusecase.NewService(projects, func(project *model.ProjectModel, opts edgeusecase.ReportOptions) *edgeusecase.Result {
		return BuildEdgeReport(project, EdgeReportOptions{
			Root:      opts.Root,
			Source:    opts.Source,
			FromLayer: opts.FromLayer,
			ToLayer:   opts.ToLayer,
			Limit:     opts.Limit,
		})
	})
}

func (a *App) RunEdge(ctx context.Context, req EdgeRequest) (*EdgeResult, error) {
	return a.edge.Run(ctx, req)
}
