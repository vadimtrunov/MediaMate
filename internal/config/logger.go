package config

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// SetupLogger configures the global logger based on the configuration
func SetupLogger(level string) *slog.Logger {
	var logLevel slog.Level

	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
		AddSource: logLevel == slog.LevelDebug, // Add source file/line in debug mode
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	slog.SetDefault(logger)
	return logger
}

// WithContext adds context fields to a logger
func WithContext(ctx context.Context, logger *slog.Logger, fields ...any) *slog.Logger {
	return logger.With(fields...)
}

// LoggerFromContext retrieves a logger from context or returns the default logger
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// ContextWithLogger adds a logger to the context
func ContextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

type contextKey string

const loggerKey contextKey = "logger"
