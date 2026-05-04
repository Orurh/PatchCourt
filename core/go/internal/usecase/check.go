package usecase

import (
	"context"

	"github.com/orurh/patchcourt/internal/platform/logx"
	"github.com/orurh/patchcourt/internal/reportmodel"
	checkusecase "github.com/orurh/patchcourt/internal/usecase/check"
)

type CheckArtifact = reportmodel.CheckArtifact
type CheckResult = reportmodel.CheckResult

type CheckRequest = checkusecase.Request
type CheckService = checkusecase.Service

func NewCheckService(scan ScanService, logger logx.Logger) CheckService {
	return checkusecase.NewService(scan, logger)
}

func (a *App) RunCheck(ctx context.Context, req CheckRequest) (*CheckResult, error) {
	return a.check.Run(ctx, req)
}
