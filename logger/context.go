package logger

import (
	"context"
	"log/slog"
)

type loggerKeyType string

const loggerKey loggerKeyType = "logger"

// WithLogger creates a new context with the provided logger value.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger from the context.
// If no logger is found, it returns the default slog logger.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}

	return slog.Default()
}
