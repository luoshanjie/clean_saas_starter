package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"service/pkg/logger"
)

func TestBuild_WithSQLiteRuntime(t *testing.T) {
	ctx := context.Background()
	rootDir := filepath.Join("..", "..")
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	dbPath := filepath.Join(t.TempDir(), "service.db")
	runtime, err := InitDB(ctx, Config{
		DBDriver:   DBDriverSQLite,
		SQLitePath: dbPath,
	})
	if err != nil {
		t.Fatalf("InitDB sqlite: %v", err)
	}
	defer runtime.Close()

	schema, err := os.ReadFile(filepath.Join("migrations", "sqlite", "demo_schema_init.sql"))
	if err != nil {
		t.Fatalf("read sqlite schema: %v", err)
	}
	if _, err := runtime.SQLite.ExecContext(ctx, string(schema)); err != nil {
		t.Fatalf("apply sqlite schema: %v", err)
	}

	app, err := Build(ctx, echo.New(), runtime, func() string {
		return "123e4567-e89b-12d3-a456-426614174000"
	}, time.Now, Config{
		DBDriver:   DBDriverSQLite,
		SQLitePath: dbPath,
		JWTSecret:  "test-secret",
	}, logger.NewNopLogger())
	if err != nil {
		t.Fatalf("Build sqlite: %v", err)
	}
	defer app.Close()
	if app == nil || app.Echo == nil {
		t.Fatalf("expected app to be initialized")
	}
}
