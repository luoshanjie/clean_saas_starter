package pg

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
)

type FileRepoPG struct {
	DB *pgxpool.Pool
}

func (r *FileRepoPG) Create(ctx context.Context, f *model.File) error {
	return withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		id, err := parseUUIDText(f.ID)
		if err != nil {
			return err
		}
		ownerID, err := parseUUIDText(f.OwnerID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
INSERT INTO files (id, tenant_id, bucket, object_key, size, mime, owner_type, owner_id, created_at)
VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6, $7, $8, $9)
`,
			id,
			f.TenantID,
			f.Bucket,
			f.ObjectKey,
			f.Size,
			f.Mime,
			f.OwnerType,
			ownerID,
			pgTimestamptz(f.CreatedAt),
		)
		return err
	})
}

func (r *FileRepoPG) GetByID(ctx context.Context, id string) (*model.File, error) {
	var out *model.File
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
SELECT id::text, COALESCE(tenant_id::text, ''), bucket, object_key, size, mime, owner_type, owner_id::text, created_at
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

func (r *FileRepoPG) DeleteByID(ctx context.Context, id string) error {
	return withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM files WHERE id = $1::uuid`, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return domainErr.ErrNotFound
		}
		return nil
	})
}
