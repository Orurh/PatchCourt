package app

import "github.com/orurh/patchcourt/internal/platform/logx"

type App struct {
	logger logx.Logger
}

func New(logger logx.Logger) *App {
	if logger == nil {
		logger = logx.Nop()
	}

	return &App{
		logger: logger,
	}
}
