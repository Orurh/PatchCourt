package cli

import (
	"context"

	"github.com/orurh/patchcourt/internal/usecase"
)

// Application описывает application/usecase-слой, который нужен CLI.
//
// CLI не должен зависеть от конкретной реализации usecase.App.
// Ему достаточно знать, какие use case методы доступны.
type Application interface {
	RunInit(ctx context.Context, req usecase.InitRequest) (*usecase.InitResult, error)
	RunScan(ctx context.Context, req usecase.ScanRequest) (*usecase.ScanResult, error)
	RunGraph(ctx context.Context, req usecase.GraphRequest) (*usecase.GraphResult, error)
	RunReview(ctx context.Context, req usecase.ReviewRequest) (*usecase.ReviewResult, error)
	RunExplain(ctx context.Context, req usecase.ExplainRequest) (*usecase.ExplainResult, error)
	RunCheck(ctx context.Context, req usecase.CheckRequest) (*usecase.CheckResult, error)
	RunEdge(ctx context.Context, req usecase.EdgeRequest) (*usecase.EdgeResult, error)
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
// Через factory CLI может получить обычный usecase.App, mock в тестах
// или другую реализацию application layer.
type AppFactory func(opts AppFactoryOptions) Application
