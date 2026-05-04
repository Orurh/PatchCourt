package usecase

import (
	"context"

	"github.com/orurh/patchcourt/internal/platform/logx"
	initusecase "github.com/orurh/patchcourt/internal/usecase/init"
)

type InitRequest = initusecase.Request
type InitResult = initusecase.Result
type InitService = initusecase.Service

func NewInitService(logger logx.Logger) InitService {
	return initusecase.NewService(logger)
}

func (a *App) RunInit(ctx context.Context, req InitRequest) (*InitResult, error) {
	return a.init.Run(ctx, req)
}
