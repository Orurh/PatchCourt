package cli

import (
	"context"

	"github.com/orurh/patchcourt/internal/app"
	"github.com/orurh/patchcourt/internal/platform/logx"
)

// Application описывает application/usecase-слой, который нужен CLI.
//
// CLI не должен зависеть от конкретной реализации app.App.
// Ему достаточно знать, какие use case методы доступны.
type Application interface {
	RunInit(ctx context.Context, req app.InitRequest) (*app.InitResult, error)
	RunScan(ctx context.Context, req app.ScanRequest) (*app.ScanResult, error)
	RunGraph(ctx context.Context, req app.GraphRequest) (*app.GraphResult, error)
	RunReview(ctx context.Context, req app.ReviewRequest) (*app.ReviewResult, error)
}

// AppFactory создает Application для CLI-команды.
//
// Через factory CLI может получить обычный app.App, mock в тестах
// или другую реализацию application layer.
type AppFactory func(logger logx.Logger) Application

func defaultAppFactory(logger logx.Logger) Application {
	return app.New(logger)
}
