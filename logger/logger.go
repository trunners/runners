package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/muesli/termenv"
)

const (
	timeFormat = "[15:04:05]"
)

// New creates a new slog.Logger that supports contextual fields.
func New() *slog.Logger {
	output := termenv.NewOutput(os.Stdout)
	colors := colors(output)
	handler := Handler{
		Handler: slog.NewTextHandler(os.Stdout, nil),
		output:  output,
		colors:  colors,
	}

	return slog.New(handler)
}

type Handler struct {
	slog.Handler

	output *termenv.Output
	colors Colors
}

func (h Handler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(fieldsKey).([]slog.Attr); ok {
		for _, v := range attrs {
			r.AddAttrs(v)
		}
	}

	level := h.output.String(fmt.Sprintf("%s:", r.Level.String()))
	switch r.Level {
	case slog.LevelDebug:
		level = level.Foreground(h.colors.Subtext)
	case slog.LevelInfo:
		level = level.Foreground(h.colors.Sky)
	case slog.LevelWarn:
		level = level.Foreground(h.colors.Peach)
	case slog.LevelError:
		level = level.Foreground(h.colors.Red)
	}

	time := h.output.String(r.Time.Format(timeFormat)).Foreground(h.colors.Subtext)
	message := h.output.String(r.Message).Foreground(h.colors.Text)
	fmt.Println(time, level, message) //nolint:forbidigo // Println is used here intentionally

	r.Attrs(func(a slog.Attr) bool {
		fmt.Println( //nolint:forbidigo // Println is used here intentionally
			"\t",
			h.output.String(fmt.Sprintf("%s:", a.Key)).Foreground(h.colors.Subtext),
			h.output.String(a.Value.String()).Foreground(h.colors.Subsubtext),
		)
		return true
	})

	return nil
}
