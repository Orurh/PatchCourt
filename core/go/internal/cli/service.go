package cli

import (
	"context"

	"github.com/orurh/patchcourt/internal/app"
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
	RunExplain(ctx context.Context, req app.ExplainRequest) (*app.ExplainResult, error)
	RunCheck(ctx context.Context, req app.CheckRequest) (*app.CheckResult, error)
	RunEdge(ctx context.Context, req app.EdgeRequest) (*app.EdgeResult, error)
}

// AppFactoryOptions описывает нейтральные CLI-настройки,
// которые нужны factory для сборки application layer.
//
// CLI не передает сюда инфраструктурные зависимости напрямую:
// logger, adapters и прочая wiring-логика остаются за factory.
type AppFactoryOptions struct {
	Verbose bool
}

// AppFactory создает Application для CLI-команды.
//
// Через factory CLI может получить обычный app.App, mock в тестах
// или другую реализацию application layer.
type AppFactory func(opts AppFactoryOptions) Application
