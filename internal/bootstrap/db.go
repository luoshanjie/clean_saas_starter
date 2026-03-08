package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/mattn/go-sqlite3"
)

type DBRuntime struct {
	Driver   string
	Postgres *pgxpool.Pool
	SQLite   *sql.DB
	cleanup  func()
}

func (r *DBRuntime) Close() {
	if r == nil || r.cleanup == nil {
		return
	}
	r.cleanup()
}

func InitDB(ctx context.Context, cfg Config) (*DBRuntime, error) {
	switch cfg.DBDriver {
	case DBDriverPostgres:
		return initPostgresDB(ctx, cfg.DBDSN)
	case DBDriverSQLite:
		return initSQLiteDB(ctx, cfg.SQLitePath)
	default:
		return nil, errors.New("unsupported DB_DRIVER: " + cfg.DBDriver)
	}
}

func initPostgresDB(ctx context.Context, dsn string) (*DBRuntime, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &DBRuntime{
		Driver:   DBDriverPostgres,
		Postgres: pool,
		cleanup:  pool.Close,
	}, nil
}

func initSQLiteDB(ctx context.Context, path string) (*DBRuntime, error) {
	if path == "" {
		return nil, errors.New("SQLITE_PATH is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &DBRuntime{
		Driver: DBDriverSQLite,
		SQLite: db,
		cleanup: func() {
			_ = db.Close()
		},
	}, nil
}
