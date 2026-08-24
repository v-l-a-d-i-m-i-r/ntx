// Package log gives structured loggers for ntx binaries.
package log

import (
	"context"
	"log/slog"
)

// Logger writes structured log records. It wraps slog.Logger so callers do
// not need to depend on log/slog directly.
type Logger struct {
	slog *slog.Logger
}

const payloadFieldName = "payload"

// DebugContext logs at debug level.
func (l *Logger) DebugContext(ctx context.Context, msg string, payload any) {
	l.slog.DebugContext(ctx, msg, payloadFieldName, payload)
}

// InfoContext logs at info level.
func (l *Logger) InfoContext(ctx context.Context, msg string, payload any) {
	l.slog.InfoContext(ctx, msg, payloadFieldName, payload)
}

// WarnContext logs at warn level.
func (l *Logger) WarnContext(ctx context.Context, msg string, payload any) {
	l.slog.WarnContext(ctx, msg, payloadFieldName, payload)
}

// ErrorContext logs at error level.
func (l *Logger) ErrorContext(ctx context.Context, msg string, payload any) {
	l.slog.ErrorContext(ctx, msg, payloadFieldName, payload)
}

// Debug logs at debug level.
func (l *Logger) Debug(msg string, payload any) {
	l.slog.Debug(msg, payloadFieldName, payload)
}

// Info logs at info level.
func (l *Logger) Info(msg string, payload any) {
	l.slog.Info(msg, payloadFieldName, payload)
}

// Warn logs at warn level.
func (l *Logger) Warn(msg string, payload any) {
	l.slog.Warn(msg, payloadFieldName, payload)
}

// Error logs at error level.
func (l *Logger) Error(msg string, payload any) {
	l.slog.Error(msg, payloadFieldName, payload)
}
