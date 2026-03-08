package pg

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domainErr "service/internal/domain/errors"
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

func (r *FileRepoPG) GetByID(ctx context.Context, id string) (*model.File, error) {
	var out *model.File
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
SELECT id::text, tenant_id::text, bucket, object_key, size, mime, owner_type, owner_id::text, created_at
FROM files
WHERE id = $1::uuid
`, id)
		f := &model.File{}
		if err := row.Scan(&f.ID, &f.TenantID, &f.Bucket, &f.ObjectKey, &f.Size, &f.Mime, &f.OwnerType, &f.OwnerID, &f.CreatedAt); err != nil {
			return err
		}
		out = f
		return nil
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.ErrNotFound
		}
		return nil, err
	}
	return out, nil
}
