package usecase

import (
	"io"
	"log/slog"

	"github.com/orurh/patchcourt/internal/platform/logx"
)

type FactoryOptions struct {
	Verbose bool
	Stderr  io.Writer
}

func NewWithOptions(opts FactoryOptions) *App {
	logger := logx.Nop()
	if opts.Verbose {
		logger = newVerboseLogger(opts.Stderr)
	}

	return New(logger)
}

func newVerboseLogger(stderr io.Writer) logx.Logger {
	if stderr == nil {
		return logx.Nop()
	}

	handler := slog.NewTextHandler(stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	return logx.NewSlogAdapter(slog.New(handler))
}
