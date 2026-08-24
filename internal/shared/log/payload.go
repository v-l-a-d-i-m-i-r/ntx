package log

import "log/slog"

// Payload holds structured fields attached to a log record. Values
// implementing error are logged via their Error() string, since the JSON
// handler cannot otherwise marshal them.
type Payload map[string]any

// LogValue implements slog.LogValuer, converting error values to strings.
func (p Payload) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, len(p))

	for k, v := range p {
		if err, ok := v.(error); ok {
			v = err.Error()
		}

		attrs = append(attrs, slog.Any(k, v))
	}

	return slog.GroupValue(attrs...)
}
