package log

import (
	"log/slog"
	"os"
)

// StdoutParams configures NewStdout.
type StdoutParams struct {
	Level   Level
	Service string
}

// NewStdout makes a Logger that writes JSON to stdout at the given level.
// Every record includes a "service" attribute set to params.Service.
func NewStdout(params StdoutParams) *Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(string(params.Level)),
	})

	return &Logger{slog: slog.New(handler).With("service", params.Service)}
}
