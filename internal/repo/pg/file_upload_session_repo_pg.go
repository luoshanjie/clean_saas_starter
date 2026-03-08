package pg

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"service/internal/domain/model"
	domainPort "service/internal/domain/port"
)

type FileUploadSessionRepoPG struct {
	DB *pgxpool.Pool
}

var _ domainPort.FileUploadSessionRepo = (*FileUploadSessionRepoPG)(nil)

func (r *FileUploadSessionRepoPG) Create(ctx context.Context, s *model.FileUploadSession) error {
	if s == nil {
		return errors.New("nil file upload session")
	}
	_, err := r.DB.Exec(ctx, `
		INSERT INTO file_upload_sessions (
			id, tenant_id, uploaded_by, scope_type, biz_type, file_name, content_type, size_bytes,
			file_url, status, expires_at, confirmed_at, last_error, created_at, updated_at
		) VALUES (
			$1, NULLIF($2, '')::uuid, $3::uuid, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15
		)
	`, s.ID, s.TenantID, s.UploadedBy, s.ScopeType, s.BizType, s.FileName, s.ContentType, s.SizeBytes, s.FileURL,
		s.Status, s.ExpiresAt, s.ConfirmedAt, s.LastError, s.CreatedAt, s.UpdatedAt)
	return err
}

func (r *FileUploadSessionRepoPG) GetByID(ctx context.Context, id string) (*model.FileUploadSession, error) {
	row := r.DB.QueryRow(ctx, `
		SELECT id::text,
		       COALESCE(tenant_id::text, '') AS tenant_id,
		       uploaded_by::text,
		       scope_type,
		       biz_type,
		       file_name,
		       content_type,
		       size_bytes,
		       file_url,
		       status,
		       expires_at,
		       confirmed_at,
		       COALESCE(mime_type, '') AS mime_type,
		       COALESCE(duration_sec, 0) AS duration_sec,
		       deleted_at,
		       COALESCE(last_error, '') AS last_error,
		       created_at,
		       updated_at
		  FROM file_upload_sessions
		 WHERE id = $1::uuid
	`, id)
	return scanFileUploadSession(row)
}

func (r *FileUploadSessionRepoPG) Confirm(ctx context.Context, id string, confirmedAt time.Time) (*model.FileUploadSession, error) {
	row := r.DB.QueryRow(ctx, `
		UPDATE file_upload_sessions
		   SET status = 'confirmed',
		       confirmed_at = $2,
		       updated_at = $2
		 WHERE id = $1::uuid
		 RETURNING id::text,
		           COALESCE(tenant_id::text, '') AS tenant_id,
		           uploaded_by::text,
		           scope_type,
		           biz_type,
		           file_name,
		           content_type,
		           size_bytes,
		           file_url,
		           status,
		           expires_at,
		           confirmed_at,
		           COALESCE(mime_type, '') AS mime_type,
		           COALESCE(duration_sec, 0) AS duration_sec,
		           deleted_at,
		           COALESCE(last_error, '') AS last_error,
		           created_at,
		           updated_at
	`, id, confirmedAt)
	return scanFileUploadSession(row)
}

func (r *FileUploadSessionRepoPG) ListExpiredPending(ctx context.Context, now time.Time, limit int) ([]*model.FileUploadSession, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.DB.Query(ctx, `
		SELECT id::text,
		       COALESCE(tenant_id::text, '') AS tenant_id,
		       uploaded_by::text,
		       scope_type,
		       biz_type,
		       file_name,
		       content_type,
		       size_bytes,
		       file_url,
		       status,
		       expires_at,
		       confirmed_at,
		       COALESCE(mime_type, '') AS mime_type,
		       COALESCE(duration_sec, 0) AS duration_sec,
		       deleted_at,
		       COALESCE(last_error, '') AS last_error,
		       created_at,
		       updated_at
		  FROM file_upload_sessions
		 WHERE status = 'pending_confirm'
		   AND expires_at < $1
		 ORDER BY expires_at ASC
		 LIMIT $2
	`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*model.FileUploadSession, 0, limit)
	for rows.Next() {
		s, err := scanFileUploadSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *FileUploadSessionRepoPG) MarkCleaned(ctx context.Context, id string, cleanedAt time.Time) error {
	_, err := r.DB.Exec(ctx, `
		UPDATE file_upload_sessions
		   SET status = 'cleaned',
		       updated_at = $2
		 WHERE id = $1::uuid
	`, id, cleanedAt)
	return err
}

func (r *FileUploadSessionRepoPG) SetLastError(ctx context.Context, id, lastError string, updatedAt time.Time) error {
	_, err := r.DB.Exec(ctx, `
		UPDATE file_upload_sessions
		   SET last_error = $2,
		       updated_at = $3
		 WHERE id = $1::uuid
	`, id, strings.TrimSpace(lastError), updatedAt)
	return err
}

type fileUploadRowScanner interface {
	Scan(dest ...any) error
}

func scanFileUploadSession(row fileUploadRowScanner) (*model.FileUploadSession, error) {
	var (
		s           model.FileUploadSession
		confirmedAt *time.Time
		deletedAt   *time.Time
	)
	err := row.Scan(
		&s.ID,
		&s.TenantID,
		&s.UploadedBy,
		&s.ScopeType,
		&s.BizType,
		&s.FileName,
		&s.ContentType,
		&s.SizeBytes,
		&s.FileURL,
		&s.Status,
		&s.ExpiresAt,
		&confirmedAt,
		&s.MimeType,
		&s.DurationSec,
		&deletedAt,
		&s.LastError,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if confirmedAt != nil {
		s.ConfirmedAt = confirmedAt
	}
	if deletedAt != nil {
		s.DeletedAt = deletedAt
	}
	return &s, nil
}
