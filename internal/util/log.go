package util

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
)

type ColorHandler struct {
	handler slog.Handler
	w       io.Writer
}

func NewColorHandler(w io.Writer, opts *slog.HandlerOptions) *ColorHandler {
	return &ColorHandler{
		handler: slog.NewTextHandler(w, opts),
		w:       w,
	}
}

func (h *ColorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ColorHandler{
		handler: h.handler.WithAttrs(attrs),
		w:       h.w,
	}
}

func (h *ColorHandler) WithGroup(name string) slog.Handler {
	return &ColorHandler{
		handler: h.handler.WithGroup(name),
		w:       h.w,
	}
}

func (h *ColorHandler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String()
	var color string

	switch r.Level {
	case slog.LevelDebug:
		color = colorGray
	case slog.LevelInfo:
		color = colorBlue
	case slog.LevelWarn:
		color = colorYellow
	case slog.LevelError:
		color = colorRed
	default:
		color = colorReset
	}

	timestamp := r.Time.Format("15:04:05")
	_, _ = fmt.Fprintf(h.w, "%s%s%s [%s%s%s] %s", //nolint:errcheck
		colorGray, timestamp, colorReset,
		color, level, colorReset,
		r.Message)

	r.Attrs(func(a slog.Attr) bool {
		_, _ = fmt.Fprintf(h.w, " %s=%v", a.Key, a.Value) //nolint:errcheck
		return true
	})

	_, _ = fmt.Fprintln(h.w) //nolint:errcheck
	return nil
}

func NewLogger(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := NewColorHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler)
}
