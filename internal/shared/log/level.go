package log

import (
	"log/slog"
	"strings"
)

// Level sets the minimum log level. Valid values: "debug", "info", "warn", "error".
type Level string

// Valid Level values.
const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

func parseLevel(raw string) slog.Level {
	switch Level(strings.ToLower(raw)) {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelError
	}
}
