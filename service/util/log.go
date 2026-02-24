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
	w        io.Writer
	level    slog.Level
	preAttrs []slog.Attr
}

func NewColorHandler(w io.Writer, opts *slog.HandlerOptions) *ColorHandler {
	level := slog.LevelInfo
	if opts != nil && opts.Level != nil {
		level = opts.Level.Level()
	}
	return &ColorHandler{
		w:     w,
		level: level,
	}
}

func (h *ColorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newH := *h
	newH.preAttrs = append(append([]slog.Attr{}, h.preAttrs...), attrs...)
	return &newH
}

func (h *ColorHandler) WithGroup(name string) slog.Handler {
	return h
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

	for _, a := range h.preAttrs {
		_, _ = fmt.Fprintf(h.w, " %s=%v", a.Key, a.Value) //nolint:errcheck
	}

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

func LogError(logger *slog.Logger, msg string, err error, attrs ...any) error {
	allAttrs := append([]any{"error", err}, attrs...)
	logger.Error(msg, allAttrs...)
	return fmt.Errorf("%s: %w", msg, err)
}
