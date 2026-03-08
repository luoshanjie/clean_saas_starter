package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

type ConsoleTextHandler struct {
	w     io.Writer
	level slog.Level
}

func NewConsoleTextHandler(w io.Writer, level slog.Level) slog.Handler {
	return &ConsoleTextHandler{w: w, level: level}
}

func (h *ConsoleTextHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *ConsoleTextHandler) Handle(_ context.Context, record slog.Record) error {
	buf := &strings.Builder{}
	ts := record.Time.Format("2006-01-02T15:04:05.000")
	lvl := strings.ToLower(record.Level.String())
	fmt.Fprintf(buf, "%s | %s | %s", ts, lvl, record.Message)

	record.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(buf, " | %s=%v", a.Key, a.Value)
		return true
	})
	buf.WriteString("\n")
	_, err := io.WriteString(h.w, buf.String())
	return err
}

func (h *ConsoleTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ConsoleTextHandlerWithAttrs{base: h, attrs: attrs}
}

func (h *ConsoleTextHandler) WithGroup(name string) slog.Handler {
	return &ConsoleTextHandlerWithAttrs{base: h, group: name}
}

type ConsoleTextHandlerWithAttrs struct {
	base  *ConsoleTextHandler
	attrs []slog.Attr
	group string
}

func (h *ConsoleTextHandlerWithAttrs) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

func (h *ConsoleTextHandlerWithAttrs) Handle(ctx context.Context, record slog.Record) error {
	for _, a := range h.attrs {
		record.AddAttrs(a)
	}
	if h.group != "" {
		record.AddAttrs(slog.String("group", h.group))
	}
	return h.base.Handle(ctx, record)
}

func (h *ConsoleTextHandlerWithAttrs) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ConsoleTextHandlerWithAttrs{base: h.base, attrs: append(h.attrs, attrs...), group: h.group}
}

func (h *ConsoleTextHandlerWithAttrs) WithGroup(name string) slog.Handler {
	return &ConsoleTextHandlerWithAttrs{base: h.base, attrs: h.attrs, group: name}
}
