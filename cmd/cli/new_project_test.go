package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseProjectSpec(t *testing.T) {
	spec, err := parseProjectSpec("my-saas", "../my-saas", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.Slug != "my-saas" {
		t.Fatalf("unexpected slug: %s", spec.Slug)
	}
	if spec.ModulePath != "my-saas" {
		t.Fatalf("unexpected module path: %s", spec.ModulePath)
	}
	if spec.DBName != "my_saas" {
		t.Fatalf("unexpected db name: %s", spec.DBName)
	}
}

func TestScaffoldProject(t *testing.T) {
	srcDir := t.TempDir()
	outDir := filepath.Join(t.TempDir(), "my-saas")

	mustWriteFile(t, filepath.Join(srcDir, ".env.example"), "DB_DRIVER=postgres\nDB_DSN=postgres://postgres:postgres@127.0.0.1:5432/service?sslmode=disable\n# SQLITE_PATH=./var/service.db\n")
	mustWriteFile(t, filepath.Join(srcDir, "go.mod"), "module service\n")
	mustWriteFile(t, filepath.Join(srcDir, "Makefile"), "APP_NAME := service\nPROJECT := service\nSRC := cmd/service/main.go\n")
	mustWriteFile(t, filepath.Join(srcDir, "cmd", "service", "main.go"), "package main\n\nimport _ \"service/internal/bootstrap\"\n")
	mustWriteFile(t, filepath.Join(srcDir, "app.yaml.example"), "database:\n  driver: \"postgres\"\n  dsn: \"postgres://user:password@127.0.0.1:5432/service?sslmode=disable\"\n  # sqlite_path: \"./var/service.db\"\n")
	mustWriteFile(t, filepath.Join(srcDir, "migrations", "pgsql", "0000_create_db.sql"), "CREATE DATABASE service_dev\n")
	mustWriteFile(t, filepath.Join(srcDir, ".env"), "JWT_SECRET=secret\n")
	mustWriteFile(t, filepath.Join(srcDir, ".git", "config"), "[core]\n")

	spec, err := parseProjectSpec("my-saas", outDir, "github.com/acme/my-saas")
	if err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	if err := scaffoldProject(srcDir, spec); err != nil {
		t.Fatalf("scaffold project: %v", err)
	}

	assertFileContains(t, filepath.Join(outDir, "go.mod"), "module github.com/acme/my-saas")
	assertFileContains(t, filepath.Join(outDir, "Makefile"), "APP_NAME := my-saas")
	assertFileContains(t, filepath.Join(outDir, "Makefile"), "SRC := cmd/my-saas/main.go")
	assertFileContains(t, filepath.Join(outDir, "cmd", "my-saas", "main.go"), "\"github.com/acme/my-saas/internal/bootstrap\"")
	assertFileContains(t, filepath.Join(outDir, "app.yaml.example"), ":5432/my_saas?")
	assertFileContains(t, filepath.Join(outDir, "app.yaml.example"), "./var/my_saas.db")
	assertFileContains(t, filepath.Join(outDir, ".env.example"), "DB_DRIVER=postgres")
	assertFileContains(t, filepath.Join(outDir, ".env.example"), "./var/my_saas.db")
	assertFileContains(t, filepath.Join(outDir, "migrations", "pgsql", "0000_create_db.sql"), "CREATE DATABASE my_saas_dev")

	if _, err := os.Stat(filepath.Join(outDir, ".env")); !os.IsNotExist(err) {
		t.Fatalf("expected .env to be skipped")
	}
	if _, err := os.Stat(filepath.Join(outDir, ".git")); !os.IsNotExist(err) {
		t.Fatalf("expected .git to be skipped")
	}
}

func TestDetectSourceProjectMeta(t *testing.T) {
	srcDir := t.TempDir()
	mustWriteFile(t, filepath.Join(srcDir, ".env.example"), "DB_DRIVER=postgres\nDB_DSN=postgres://postgres:postgres@127.0.0.1:5432/service?sslmode=disable\n# SQLITE_PATH=./var/service.db\n")
	mustWriteFile(t, filepath.Join(srcDir, "go.mod"), "module github.com/acme/source\n")
	mustWriteFile(t, filepath.Join(srcDir, "Makefile"), "APP_NAME := source\nPROJECT := source\n")
	mustWriteFile(t, filepath.Join(srcDir, "cmd", "source", "main.go"), "package main\n")
	mustWriteFile(t, filepath.Join(srcDir, "cmd", "cli", "main.go"), "package main\n")

	meta, err := detectSourceProjectMeta(srcDir)
	if err != nil {
		t.Fatalf("detect source project meta: %v", err)
	}
	if meta.ModulePath != "github.com/acme/source" {
		t.Fatalf("unexpected module path: %s", meta.ModulePath)
	}
	if meta.CmdName != "source" {
		t.Fatalf("unexpected cmd name: %s", meta.CmdName)
	}
	if meta.DBName == "" {
		t.Fatalf("expected non-empty db name")
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	if !containsText(data, want) {
		t.Fatalf("expected %s to contain %q, got:\n%s", path, want, string(data))
	}
}
