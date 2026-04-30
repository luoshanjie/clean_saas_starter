package bootstrap

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DBRuntime struct {
	Driver   string
	Postgres *pgxpool.Pool
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
