package pg

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"service/internal/domain/model"
	"service/internal/repo/pg/sqlcpg"
)

type FileRepoPG struct {
	DB *pgxpool.Pool
}

func (r *FileRepoPG) Create(ctx context.Context, f *model.File) error {
	return withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		id, err := parseUUIDText(f.ID)
		if err != nil {
			return err
		}
		tenantID, err := parseUUIDText(f.TenantID)
		if err != nil {
			return err
		}
		ownerID, err := parseUUIDText(f.OwnerID)
		if err != nil {
			return err
		}
		return q.FileCreate(ctx, sqlcpg.FileCreateParams{
			ID:        id,
			TenantID:  tenantID,
			Bucket:    f.Bucket,
			ObjectKey: f.ObjectKey,
			Size:      f.Size,
			Mime:      f.Mime,
			OwnerType: f.OwnerType,
			OwnerID:   ownerID,
			CreatedAt: pgTimestamptz(f.CreatedAt),
		})
	})
}
