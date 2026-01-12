package logger

import (
	"context"
	"log/slog"
)

type fieldsKeyType string

const fieldsKey fieldsKeyType = "fields"

// Append adds a slog attribute to the provided context so that it will be
// included in any Record created with such context.
func Append(parent context.Context, attr slog.Attr) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	if v, ok := parent.Value(fieldsKey).([]slog.Attr); ok {
		v = append(v, attr)
		return context.WithValue(parent, fieldsKey, v)
	}

	v := []slog.Attr{}
	v = append(v, attr)
	return context.WithValue(parent, fieldsKey, v)
}
