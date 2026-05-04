package usecase

import (
	"context"

	"github.com/orurh/patchcourt/internal/reportmodel"
	explainusecase "github.com/orurh/patchcourt/internal/usecase/explain"
)

type ExplainResult = reportmodel.ExplainResult

type ExplainFormat = explainusecase.Format

const (
	ExplainFormatText = explainusecase.FormatText
	ExplainFormatJSON = explainusecase.FormatJSON
)

type ExplainRequest = explainusecase.Request
type ExplainService = explainusecase.Service

func NewExplainService(projects ProjectBuilder) ExplainService {
	return explainusecase.NewService(projects)
}

func (a *App) RunExplain(ctx context.Context, req ExplainRequest) (*ExplainResult, error) {
	return a.explain.Run(ctx, req)
}
