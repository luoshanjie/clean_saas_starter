package logger

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRollingFileHandler_WritesFile(t *testing.T) {
	dir := t.TempDir()
	h, err := NewRollingFileHandler(dir, "app", "json", slog.LevelInfo)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)
	if err := h.Handle(nil, rec); err != nil {
		t.Fatalf("handle: %v", err)
	}

	name := filepath.Join(dir, "app-"+time.Now().Format("2006-01-02")+".log")
	info, err := os.Stat(name)
	if err != nil {
		t.Fatalf("stat log file: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("log file empty")
	}
}
