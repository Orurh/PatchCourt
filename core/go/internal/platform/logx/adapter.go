package logx

import "log/slog"

// SlogAdapter adapts slog.Logger to Logger.
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter returns a Logger backed by slog.
func NewSlogAdapter(logger *slog.Logger) Logger {
	if logger == nil {
		return Nop()
	}

	return &SlogAdapter{logger: logger}
}

func (s *SlogAdapter) Debug(msg string, fields ...Field) {
	s.logger.Debug(msg, toSlogArgs(fields)...)
}

func (s *SlogAdapter) Info(msg string, fields ...Field) {
	s.logger.Info(msg, toSlogArgs(fields)...)
}

func (s *SlogAdapter) Warn(msg string, fields ...Field) {
	s.logger.Warn(msg, toSlogArgs(fields)...)
}

func (s *SlogAdapter) Error(msg string, fields ...Field) {
	s.logger.Error(msg, toSlogArgs(fields)...)
}

func (s *SlogAdapter) With(fields ...Field) Logger {
	return &SlogAdapter{
		logger: s.logger.With(toSlogArgs(fields)...),
	}
}

// Sync flushes buffered logs if supported. slog does not require flushing.
func (s *SlogAdapter) Sync() error {
	return nil
}

func toSlogArgs(fields []Field) []any {
	args := make([]any, 0, len(fields))
	for _, field := range fields {
		args = append(args, slog.Any(field.Key, field.Value))
	}
	return args
}

var _ Logger = (*SlogAdapter)(nil)
