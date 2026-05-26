package pg

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"service/internal/domain/authctx"
)

func withRLS(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	info, ok := authctx.From(ctx)
	if !ok {
		return errors.New("missing auth context")
	}
	if info.ScopeType == "" {
		return errors.New("missing scope_type")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// PostgreSQL-only defense layer: set transaction-local context variables
	// consumed by RLS policies. Portable tenant isolation must still be modeled
	// in repository contracts so non-PostgreSQL adapters can enforce it without
	// relying on RLS.
	if _, err := tx.Exec(ctx, `
		SELECT
			set_config('app.scope_type', $1, true),
			set_config('app.tenant_id', $2, true),
			set_config('app.user_id', $3, true)
	`, info.ScopeType, info.TenantID, info.UserID); err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
