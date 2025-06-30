// Package logging provides structured logging functionality.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Logger provides structured logging capabilities.
type Logger struct {
	*slog.Logger
}

// NewLogger creates a new logger with the specified level.
func NewLogger(level string) *Logger {
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

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})

	return &Logger{
		Logger: slog.New(handler),
	}
}

// WithTool returns a logger with tool information.
func (l *Logger) WithTool(toolName string) *Logger {
	return &Logger{
		Logger: l.With(slog.String("tool", toolName)),
	}
}

// WithSession returns a logger with session information.
func (l *Logger) WithSession(sessionID string) *Logger {
	return &Logger{
		Logger: l.With(slog.String("session", sessionID)),
	}
}

// Debug logs a debug message with optional arguments.
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Info logs an info message with optional arguments.
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Warn logs a warning message with optional arguments.
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Error logs an error message with optional arguments.
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}
