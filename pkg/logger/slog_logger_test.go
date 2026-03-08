package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSlogLogger_WritesFile(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Dir:           dir,
		Level:         "info",
		Format:        "json",
		EnableConsole: false,
	}
	l := NewSlogLogger(cfg, "app")
	l.Info("hello", "k", "v")

	name := filepath.Join(dir, "app-"+time.Now().Format("2006-01-02")+".log")
	info, err := os.Stat(name)
	if err != nil {
		t.Fatalf("stat log file: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("log file empty")
	}
}
