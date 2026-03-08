package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"service/internal/domain/model"
	domainPort "service/internal/domain/port"
)

type FileUploadSessionRepoSQLite struct {
	DB *sql.DB
}

var _ domainPort.FileUploadSessionRepo = (*FileUploadSessionRepoSQLite)(nil)

func (r *FileUploadSessionRepoSQLite) Create(ctx context.Context, s *model.FileUploadSession) error {
	if s == nil {
		return errors.New("nil file upload session")
	}
	_, err := r.DB.ExecContext(ctx, `
INSERT INTO file_upload_sessions (
	id, tenant_id, uploaded_by, scope_type, biz_type, file_name, content_type, size_bytes,
	file_url, status, expires_at, confirmed_at, last_error, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID,
		nullableString(s.TenantID),
		s.UploadedBy,
		s.ScopeType,
		s.BizType,
		s.FileName,
		s.ContentType,
		s.SizeBytes,
		s.FileURL,
		s.Status,
		formatSQLiteTime(s.ExpiresAt),
		nullableSQLiteTime(s.ConfirmedAt),
		s.LastError,
		formatSQLiteTime(s.CreatedAt),
		formatSQLiteTime(s.UpdatedAt),
	)
	return err
}

func (r *FileUploadSessionRepoSQLite) GetByID(ctx context.Context, id string) (*model.FileUploadSession, error) {
	row := r.DB.QueryRowContext(ctx, `
SELECT id, COALESCE(tenant_id, ''), uploaded_by, scope_type, biz_type, file_name, content_type, size_bytes,
       file_url, status, expires_at, COALESCE(confirmed_at, ''), COALESCE(mime_type, ''), COALESCE(duration_sec, 0),
       COALESCE(deleted_at, ''), COALESCE(last_error, ''), created_at, updated_at
FROM file_upload_sessions
WHERE id = ?`, id)
	return scanFileUploadSession(row)
}

func (r *FileUploadSessionRepoSQLite) Confirm(ctx context.Context, id string, confirmedAt time.Time) (*model.FileUploadSession, error) {
	if _, err := r.DB.ExecContext(ctx, `
UPDATE file_upload_sessions
SET status = 'confirmed', confirmed_at = ?, updated_at = ?
WHERE id = ?`, formatSQLiteTime(confirmedAt), formatSQLiteTime(confirmedAt), id); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *FileUploadSessionRepoSQLite) ListExpiredPending(ctx context.Context, now time.Time, limit int) ([]*model.FileUploadSession, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.DB.QueryContext(ctx, `
SELECT id, COALESCE(tenant_id, ''), uploaded_by, scope_type, biz_type, file_name, content_type, size_bytes,
       file_url, status, expires_at, COALESCE(confirmed_at, ''), COALESCE(mime_type, ''), COALESCE(duration_sec, 0),
       COALESCE(deleted_at, ''), COALESCE(last_error, ''), created_at, updated_at
FROM file_upload_sessions
WHERE status = 'pending_confirm' AND expires_at < ?
ORDER BY expires_at ASC
LIMIT ?`, formatSQLiteTime(now), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.FileUploadSession{}
	for rows.Next() {
		s, err := scanFileUploadSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *FileUploadSessionRepoSQLite) MarkCleaned(ctx context.Context, id string, cleanedAt time.Time) error {
	_, err := r.DB.ExecContext(ctx, `
UPDATE file_upload_sessions
SET status = 'cleaned', updated_at = ?
WHERE id = ?`, formatSQLiteTime(cleanedAt), id)
	return err
}

func (r *FileUploadSessionRepoSQLite) SetLastError(ctx context.Context, id, lastError string, updatedAt time.Time) error {
	_, err := r.DB.ExecContext(ctx, `
UPDATE file_upload_sessions
SET last_error = ?, updated_at = ?
WHERE id = ?`, strings.TrimSpace(lastError), formatSQLiteTime(updatedAt), id)
	return err
}

type fileUploadSessionScanner interface {
	Scan(dest ...any) error
}

func scanFileUploadSession(row fileUploadSessionScanner) (*model.FileUploadSession, error) {
	var (
		s                               model.FileUploadSession
		expiresAt, createdAt, updatedAt string
		confirmedAt                     sql.NullString
		deletedAt                       sql.NullString
	)
	if err := row.Scan(
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
		&expiresAt,
		&confirmedAt,
		&s.MimeType,
		&s.DurationSec,
		&deletedAt,
		&s.LastError,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, mapNotFound(err)
	}
	var err error
	s.ExpiresAt, err = parseSQLiteTime(expiresAt)
	if err != nil {
		return nil, err
	}
	s.ConfirmedAt, err = parseNullableSQLiteTime(confirmedAt)
	if err != nil {
		return nil, err
	}
	s.DeletedAt, err = parseNullableSQLiteTime(deletedAt)
	if err != nil {
		return nil, err
	}
	s.CreatedAt, err = parseSQLiteTime(createdAt)
	if err != nil {
		return nil, err
	}
	s.UpdatedAt, err = parseSQLiteTime(updatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
