package logger

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

type countHandler struct{ n int }

func (h *countHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *countHandler) Handle(context.Context, slog.Record) error {
	h.n++
	return nil
}
func (h *countHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *countHandler) WithGroup(string) slog.Handler      { return h }

func TestMultiHandler_Fanout(t *testing.T) {
	h1 := &countHandler{}
	h2 := &countHandler{}

	mh := NewMultiHandler(h1, h2)
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	if err := mh.Handle(context.Background(), rec); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if h1.n != 1 || h2.n != 1 {
		t.Fatalf("expected both handlers to be called")
	}
}
