package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"service/internal/domain/model"
	domainPort "service/internal/domain/port"
)

type FileRepoSQLite struct {
	DB *sql.DB
}

var _ domainPort.FileRepo = (*FileRepoSQLite)(nil)

func (r *FileRepoSQLite) Create(ctx context.Context, f *model.File) error {
	if f == nil {
		return errors.New("nil file")
	}
	_, err := r.DB.ExecContext(ctx, `
INSERT INTO files (id, tenant_id, bucket, object_key, size, mime, owner_type, owner_id, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID,
		f.TenantID,
		f.Bucket,
		f.ObjectKey,
		f.Size,
		f.Mime,
		f.OwnerType,
		f.OwnerID,
		formatSQLiteTime(f.CreatedAt),
	)
	return err
}

func (r *FileRepoSQLite) GetByID(ctx context.Context, id string) (*model.File, error) {
	row := r.DB.QueryRowContext(ctx, `
SELECT id, tenant_id, bucket, object_key, size, mime, owner_type, owner_id, created_at
FROM files
WHERE id = ?`, id)
	var (
		f         model.File
		createdAt string
	)
	if err := row.Scan(&f.ID, &f.TenantID, &f.Bucket, &f.ObjectKey, &f.Size, &f.Mime, &f.OwnerType, &f.OwnerID, &createdAt); err != nil {
		return nil, mapNotFound(err)
	}
	var err error
	f.CreatedAt, err = parseSQLiteTime(createdAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}
