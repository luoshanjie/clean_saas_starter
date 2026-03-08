package bootstrap

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestInitDB_SQLite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "service.db")
	runtime, err := InitDB(context.Background(), Config{
		DBDriver:   DBDriverSQLite,
		SQLitePath: dbPath,
	})
	if err == nil {
		runtime.Close()
	}
	if err != nil {
		t.Fatalf("unexpected sqlite init error: %v", err)
	}
	if runtime == nil || runtime.SQLite == nil {
		t.Fatalf("expected sqlite runtime to be initialized")
	}
}

func TestNewBootstrapRepos_DriverAware(t *testing.T) {
	now := func() time.Time { return time.Unix(0, 0) }

	t.Run("nil_database_runtime", func(t *testing.T) {
		_, err := newBootstrapRepos(nil, now, OSSConfig{}, "")
		if err == nil {
			t.Fatalf("expected error for nil database runtime")
		}
	})

	t.Run("sqlite_runtime", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}
		defer db.Close()
		repos, err := newBootstrapRepos(&DBRuntime{Driver: DBDriverSQLite, SQLite: db}, now, OSSConfig{}, "")
		if err != nil {
			t.Fatalf("unexpected sqlite repo wiring error: %v", err)
		}
		if repos.authRepo == nil || repos.tenantRepo == nil || repos.auditRepo == nil || repos.fileUploadSessionRepo == nil {
			t.Fatalf("expected sqlite repos to be initialized")
		}
	})
}
