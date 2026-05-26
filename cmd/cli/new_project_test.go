package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

func TestParseProjectSpec_DefaultOutput(t *testing.T) {
	spec, err := parseProjectSpec("my-saas", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.OutputDir != filepath.Clean("../my-saas") {
		t.Fatalf("unexpected output dir: %s", spec.OutputDir)
	}
}

func TestScaffoldProject(t *testing.T) {
	srcDir := t.TempDir()
	outDir := filepath.Join(t.TempDir(), "my-saas")

	mustWriteFile(t, filepath.Join(srcDir, ".env.example"), "DB_DRIVER=postgres\nDB_DSN=postgres://postgres:postgres@127.0.0.1:5432/service?sslmode=disable\n")
	mustWriteFile(t, filepath.Join(srcDir, "go.mod"), "module service\n")
	mustWriteFile(t, filepath.Join(srcDir, "Makefile"), "APP_NAME := service\nPROJECT := service\nSRC := cmd/service/main.go\n")
	mustWriteFile(t, filepath.Join(srcDir, "README.md"), "# Clean SaaS Starter\n\nAn open-source multi-tenant SaaS starter for building new systems on top of a reusable kernel instead of starting from a vertical business project.\n\nThis repository is a generic SaaS scaffold, not a concrete business application.\n")
	mustWriteFile(t, filepath.Join(srcDir, "README.zh-CN.md"), "# Clean SaaS Starter\n\n一个开源的多租户 SaaS 脚手架，用于在可复用的内核之上构建新系统，而不是从某个具体业务项目开始演化。\n\n本仓库是一个通用 SaaS 脚手架，不是某个垂直业务应用。\n")
	mustWriteFile(t, filepath.Join(srcDir, "cmd", "service", "main.go"), "package main\n\nimport _ \"service/internal/bootstrap\"\n")
	mustWriteFile(t, filepath.Join(srcDir, "cmd", "cli", "new_project_test.go"), "package main\n")
	mustWriteFile(t, filepath.Join(srcDir, "app.yaml.example"), "database:\n  driver: \"postgres\"\n  dsn: \"postgres://user:password@127.0.0.1:5432/service?sslmode=disable\"\n")
	mustWriteFile(t, filepath.Join(srcDir, "deploy", "systemd", "service.service"), "WorkingDirectory=/opt/service\nExecStart=/opt/service/service\nEnvironmentFile=-/etc/service/service.env\n")
	mustWriteFile(t, filepath.Join(srcDir, "deploy", "systemd", "service.env.example"), "APP_CONFIG_FILE=/etc/service/app.yaml\nLOG_DIR=/var/log/service\n")
	mustWriteFile(t, filepath.Join(srcDir, "migrations", "pgsql", "0000_create_db.sql"), "CREATE DATABASE service_dev\n")
	mustWriteFile(t, filepath.Join(srcDir, "migrations", "sqlite", "0001_kernel_core.sql"), "-- sqlite baseline\n")
	mustWriteFile(t, filepath.Join(srcDir, "internal", "repo", "sqlite", "auth_repo_sqlite.go"), "package sqlite\n")
	mustWriteFile(t, filepath.Join(srcDir, "docs", "superpowers", "plans", "plan.md"), "# internal plan\n")
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
	assertFileContains(t, filepath.Join(outDir, "README.md"), "# My Saas")
	assertFileContains(t, filepath.Join(outDir, "README.md"), "My Saas backend service built on a reusable SaaS kernel.")
	assertFileContains(t, filepath.Join(outDir, "README.md"), "My Saas backend service generated from Clean SaaS Starter.")
	assertFileContains(t, filepath.Join(outDir, "README.zh-CN.md"), "# My Saas")
	assertFileContains(t, filepath.Join(outDir, "README.zh-CN.md"), "My Saas 后端服务，基于可复用 SaaS 内核构建。")
	assertFileContains(t, filepath.Join(outDir, "README.zh-CN.md"), "My Saas 后端服务，由 Clean SaaS Starter 生成。")
	assertFileContains(t, filepath.Join(outDir, "cmd", "my-saas", "main.go"), "\"github.com/acme/my-saas/internal/bootstrap\"")
	assertFileContains(t, filepath.Join(outDir, "app.yaml.example"), ":5432/my_saas?")
	assertFileContains(t, filepath.Join(outDir, ".env.example"), "DB_DRIVER=postgres")
	assertFileContains(t, filepath.Join(outDir, "deploy", "systemd", "service.service"), "WorkingDirectory=/opt/my-saas")
	assertFileContains(t, filepath.Join(outDir, "deploy", "systemd", "service.service"), "ExecStart=/opt/my-saas/my-saas")
	assertFileContains(t, filepath.Join(outDir, "deploy", "systemd", "service.service"), "EnvironmentFile=-/etc/my-saas/my-saas.env")
	assertFileContains(t, filepath.Join(outDir, "deploy", "systemd", "service.env.example"), "APP_CONFIG_FILE=/etc/my-saas/app.yaml")
	assertFileContains(t, filepath.Join(outDir, "deploy", "systemd", "service.env.example"), "LOG_DIR=/var/log/my-saas")
	assertFileContains(t, filepath.Join(outDir, "migrations", "pgsql", "0000_create_db.sql"), "CREATE DATABASE my_saas_dev")

	assertPathMissing(t, filepath.Join(outDir, "migrations", "sqlite"))
	assertPathMissing(t, filepath.Join(outDir, "internal", "repo", "sqlite"))
	assertPathMissing(t, filepath.Join(outDir, "docs", "superpowers"))
	assertPathMissing(t, filepath.Join(outDir, "cmd", "cli", "new_project_test.go"))
	if _, err := os.Stat(filepath.Join(outDir, ".env")); !os.IsNotExist(err) {
		t.Fatalf("expected .env to be skipped")
	}
	if _, err := os.Stat(filepath.Join(outDir, ".git")); !os.IsNotExist(err) {
		t.Fatalf("expected .git to be skipped")
	}
}

func TestInitProjectGit(t *testing.T) {
	outDir := t.TempDir()
	if err := initProjectGit(outDir); err != nil {
		t.Fatalf("init git: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, ".git")); err != nil {
		t.Fatalf("expected .git directory: %v", err)
	}
}

func TestPrintProjectNextSteps(t *testing.T) {
	spec, err := parseProjectSpec("my-saas", "../my-saas", "")
	if err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	var out bytes.Buffer
	printProjectNextSteps(&out, spec, true)
	text := out.String()
	for _, want := range []string{
		"project scaffold ready: ../my-saas",
		"cd ../my-saas",
		"cp .env.example .env",
		"edit .env and app.yaml.example",
		"make dev",
		"make package-linux-amd64",
		"git repository initialized",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected next steps to contain %q, got:\n%s", want, text)
		}
	}
}

func TestDetectSourceProjectMeta(t *testing.T) {
	srcDir := t.TempDir()
	mustWriteFile(t, filepath.Join(srcDir, ".env.example"), "DB_DRIVER=postgres\nDB_DSN=postgres://postgres:postgres@127.0.0.1:5432/service?sslmode=disable\n")
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

func assertPathMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected path to be skipped: %s", path)
	}
}
