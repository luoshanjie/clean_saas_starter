package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type RollingFileHandler struct {
	mu          sync.Mutex
	category    string
	dir         string
	format      string // "text" or "json"
	level       slog.Level
	currentDate string
	file        *os.File
	handler     slog.Handler
}

func NewRollingFileHandler(dir, category, format string, level slog.Level) (*RollingFileHandler, error) {
	r := &RollingFileHandler{
		category: category,
		dir:      dir,
		format:   format,
		level:    level,
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir failed: %w", err)
	}
	if err := r.rotateIfNeeded(); err != nil {
		return nil, err
	}
	return r, nil
}

func (h *RollingFileHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *RollingFileHandler) Handle(ctx context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.rotateIfNeeded(); err != nil {
		return err
	}

	return h.handler.Handle(ctx, record)
}

func (h *RollingFileHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()
	return &RollingFileHandler{
		category:    h.category,
		dir:         h.dir,
		format:      h.format,
		level:       h.level,
		currentDate: h.currentDate,
		file:        h.file,
		handler:     h.handler.WithAttrs(attrs),
	}
}

func (h *RollingFileHandler) WithGroup(name string) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()
	return &RollingFileHandler{
		category:    h.category,
		dir:         h.dir,
		format:      h.format,
		level:       h.level,
		currentDate: h.currentDate,
		file:        h.file,
		handler:     h.handler.WithGroup(name),
	}
}

func (h *RollingFileHandler) rotateIfNeeded() error {
	today := time.Now().Format("2006-01-02")
	if today == h.currentDate && h.file != nil {
		return nil
	}
	if h.file != nil {
		_ = h.file.Close()
	}

	filename := filepath.Join(h.dir, fmt.Sprintf("%s-%s.log", h.category, today))
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file failed: %w", err)
	}

	h.file = f
	h.currentDate = today

	opts := &slog.HandlerOptions{Level: h.level}
	if h.format == "json" {
		h.handler = slog.NewJSONHandler(f, opts)
	} else {
		h.handler = slog.NewTextHandler(f, opts)
	}

	return nil
}
