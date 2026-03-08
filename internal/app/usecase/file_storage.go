package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
	"service/internal/domain/port"
)

const (
	defaultUploadSessionTTL = 24 * time.Hour
)

type FileUploadSessionCreateInput struct {
	TenantID    string
	ActorUserID string
	ScopeType   string
	BizType     string
	FileName    string
	ContentType string
	SizeBytes   int64
}

type FileUploadSessionCreateOutput struct {
	UploadID  string
	UploadURL string
	FileURL   string
	Status    string
	ExpiresAt time.Time
}

type FileUploadSessionCreateUsecase struct {
	Storage port.ObjectStorage
	Repo    port.FileUploadSessionRepo
	IDGen   func() string
	Now     func() time.Time
	TTL     time.Duration
}

func (u *FileUploadSessionCreateUsecase) Execute(ctx context.Context, in FileUploadSessionCreateInput) (*FileUploadSessionCreateOutput, error) {
	if u.Storage == nil || u.Repo == nil || u.IDGen == nil || u.Now == nil {
		return nil, domainErr.ErrValidation
	}
	if strings.TrimSpace(in.ActorUserID) == "" || strings.TrimSpace(in.BizType) == "" || strings.TrimSpace(in.FileName) == "" {
		return nil, domainErr.ErrValidation
	}
	uploadID, err := canonicalUUIDText(u.IDGen())
	if err != nil {
		return nil, domainErr.ErrValidation
	}
	now := u.Now()
	ttl := u.TTL
	if ttl <= 0 {
		ttl = defaultUploadSessionTTL
	}
	expiresAt := now.Add(ttl)
	ps, err := u.Storage.PresignUpload(ctx, port.PresignUploadInput{
		BizType:     strings.TrimSpace(in.BizType),
		FileName:    strings.TrimSpace(in.FileName),
		ContentType: strings.TrimSpace(in.ContentType),
	})
	if err != nil {
		return nil, fmt.Errorf("upload_presign_failed: %w", err)
	}
	s := &model.FileUploadSession{
		ID:          uploadID,
		TenantID:    strings.TrimSpace(in.TenantID),
		UploadedBy:  strings.TrimSpace(in.ActorUserID),
		ScopeType:   strings.TrimSpace(in.ScopeType),
		BizType:     strings.TrimSpace(in.BizType),
		FileName:    strings.TrimSpace(in.FileName),
		ContentType: strings.TrimSpace(in.ContentType),
		SizeBytes:   in.SizeBytes,
		FileURL:     ps.FileURL,
		Status:      "pending_confirm",
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := u.Repo.Create(ctx, s); err != nil {
		return nil, fmt.Errorf("upload_session_persist_failed: %w", err)
	}
	return &FileUploadSessionCreateOutput{
		UploadID:  uploadID,
		UploadURL: ps.UploadURL,
		FileURL:   ps.FileURL,
		Status:    s.Status,
		ExpiresAt: expiresAt,
	}, nil
}

type FileUploadConfirmInput struct {
	UploadID     string
	TenantID     string
	ActorUserID  string
	SizeBytes    int64
	Checksum     string
	ClientSource string
}

type FileUploadConfirmOutput struct {
	Status  string
	FileID  string
	FileURL string
}

type FileUploadConfirmUsecase struct {
	Repo port.FileUploadSessionRepo
	Now  func() time.Time
}

func (u *FileUploadConfirmUsecase) Execute(ctx context.Context, in FileUploadConfirmInput) (*FileUploadConfirmOutput, error) {
	if u.Repo == nil || u.Now == nil {
		return nil, domainErr.ErrValidation
	}
	if strings.TrimSpace(in.UploadID) == "" || strings.TrimSpace(in.ActorUserID) == "" {
		return nil, domainErr.ErrValidation
	}
	s, err := u.Repo.GetByID(ctx, strings.TrimSpace(in.UploadID))
	if err != nil {
		return nil, fmt.Errorf("upload_session_load_failed: %w", err)
	}
	if s == nil {
		return nil, domainErr.ErrNotFound
	}
	if s.UploadedBy != strings.TrimSpace(in.ActorUserID) {
		return nil, domainErr.ErrForbidden
	}
	if s.Status == "confirmed" {
		return &FileUploadConfirmOutput{Status: "ok", FileID: s.ID, FileURL: s.FileURL}, nil
	}
	if s.Status != "pending_confirm" {
		return nil, domainErr.ErrValidation
	}
	if u.Now().After(s.ExpiresAt) {
		return nil, domainErr.ErrValidation
	}
	updated, err := u.Repo.Confirm(ctx, s.ID, u.Now())
	if err != nil {
		return nil, fmt.Errorf("upload_confirm_persist_failed: %w", err)
	}
	return &FileUploadConfirmOutput{Status: "ok", FileID: updated.ID, FileURL: updated.FileURL}, nil
}

type FileDownloadPresignInput struct {
	FileURL string
}

type FileDownloadPresignOutput struct {
	DownloadURL string
}

type FileDownloadPresignUsecase struct {
	Storage port.ObjectStorage
}

func (u *FileDownloadPresignUsecase) Execute(ctx context.Context, in FileDownloadPresignInput) (*FileDownloadPresignOutput, error) {
	if u.Storage == nil || strings.TrimSpace(in.FileURL) == "" {
		return nil, domainErr.ErrValidation
	}
	out, err := u.Storage.PresignDownload(ctx, port.PresignDownloadInput{
		FileURL: strings.TrimSpace(in.FileURL),
	})
	if err != nil {
		return nil, err
	}
	return &FileDownloadPresignOutput{DownloadURL: out.DownloadURL}, nil
}

type CleanupExpiredUploadSessionsInput struct {
	Limit int
}

type CleanupExpiredUploadSessionsOutput struct {
	Scanned int
	Cleaned int
	Failed  int
}

type CleanupExpiredUploadSessionsUsecase struct {
	Storage port.ObjectStorage
	Repo    port.FileUploadSessionRepo
	Now     func() time.Time
}

func (u *CleanupExpiredUploadSessionsUsecase) Execute(ctx context.Context, in CleanupExpiredUploadSessionsInput) (*CleanupExpiredUploadSessionsOutput, error) {
	if u.Storage == nil || u.Repo == nil || u.Now == nil {
		return nil, domainErr.ErrValidation
	}
	limit := in.Limit
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	now := u.Now()
	sessions, err := u.Repo.ListExpiredPending(ctx, now, limit)
	if err != nil {
		return nil, err
	}
	out := &CleanupExpiredUploadSessionsOutput{Scanned: len(sessions)}
	for _, s := range sessions {
		if s == nil || strings.TrimSpace(s.FileURL) == "" {
			out.Failed++
			continue
		}
		if err := u.Storage.DeleteByFileURL(ctx, s.FileURL); err != nil {
			out.Failed++
			_ = u.Repo.SetLastError(ctx, s.ID, err.Error(), now)
			continue
		}
		if err := u.Repo.MarkCleaned(ctx, s.ID, now); err != nil {
			out.Failed++
			_ = u.Repo.SetLastError(ctx, s.ID, err.Error(), now)
			continue
		}
		out.Cleaned++
	}
	return out, nil
}
