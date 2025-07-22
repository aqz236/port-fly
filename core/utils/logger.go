package utils

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// LogLevel represents log levels
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Logger interface defines the logging contract
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
	WithGroup(name string) Logger
}

// PortFlyLogger implements the Logger interface using slog
type PortFlyLogger struct {
	logger *slog.Logger
}

// LoggerConfig contains logger configuration
type LoggerConfig struct {
	Level      string `json:"level" yaml:"level"`
	Format     string `json:"format" yaml:"format"`     // "json" or "text"
	Output     string `json:"output" yaml:"output"`     // "stdout", "stderr", or file path
	MaxSize    int    `json:"max_size" yaml:"max_size"` // megabytes
	MaxBackups int    `json:"max_backups" yaml:"max_backups"`
	MaxAge     int    `json:"max_age" yaml:"max_age"` // days
	Compress   bool   `json:"compress" yaml:"compress"`
}

// NewLogger creates a new logger instance
func NewLogger(config LoggerConfig) (Logger, error) {
	var writer io.Writer

	// Determine output destination
	switch strings.ToLower(config.Output) {
	case "stdout", "":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		// File output with rotation
		if err := os.MkdirAll(filepath.Dir(config.Output), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		writer = &lumberjack.Logger{
			Filename:   config.Output,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		}
	}

	// Parse log level
	var level slog.Level
	switch strings.ToLower(config.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info", "":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return nil, fmt.Errorf("invalid log level: %s", config.Level)
	}

	// Create handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Add timestamp formatting
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	}

	switch strings.ToLower(config.Format) {
	case "json", "":
		handler = slog.NewJSONHandler(writer, opts)
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	default:
		return nil, fmt.Errorf("invalid log format: %s", config.Format)
	}

	logger := slog.New(handler)

	return &PortFlyLogger{logger: logger}, nil
}

// Debug logs a debug message
func (l *PortFlyLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Info logs an info message
func (l *PortFlyLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Warn logs a warning message
func (l *PortFlyLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error logs an error message
func (l *PortFlyLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// With returns a new logger with the given attributes
func (l *PortFlyLogger) With(args ...any) Logger {
	return &PortFlyLogger{logger: l.logger.With(args...)}
}

// WithGroup returns a new logger with the given group name
func (l *PortFlyLogger) WithGroup(name string) Logger {
	return &PortFlyLogger{logger: l.logger.WithGroup(name)}
}

// DefaultLogger returns a default logger for development
func DefaultLogger() Logger {
	logger, _ := NewLogger(LoggerConfig{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	})
	return logger
}

// LoggerContext is a context key for logger
type loggerContextKey struct{}

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// FromContext extracts a logger from the context
func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(loggerContextKey{}).(Logger); ok {
		return logger
	}
	return DefaultLogger()
}
