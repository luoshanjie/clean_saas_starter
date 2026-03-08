package usecase

import (
	"context"
	"errors"
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
	Repo     port.FileUploadSessionRepo
	FileRepo port.FileRepo
	Storage  port.ObjectStorage
	Now      func() time.Time
}

func (u *FileUploadConfirmUsecase) Execute(ctx context.Context, in FileUploadConfirmInput) (*FileUploadConfirmOutput, error) {
	if u.Repo == nil || u.FileRepo == nil || u.Storage == nil || u.Now == nil {
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
	if s.Status != "pending_confirm" && s.Status != "confirmed" {
		return nil, domainErr.ErrValidation
	}
	if s.Status == "pending_confirm" && u.Now().After(s.ExpiresAt) {
		return nil, domainErr.ErrValidation
	}
	updated := s
	if s.Status == "pending_confirm" {
		updated, err = u.Repo.Confirm(ctx, s.ID, u.Now())
		if err != nil {
			return nil, fmt.Errorf("upload_confirm_persist_failed: %w", err)
		}
	}
	if _, err := u.FileRepo.GetByID(ctx, updated.ID); err == nil {
		return &FileUploadConfirmOutput{Status: "ok", FileID: updated.ID, FileURL: updated.FileURL}, nil
	} else if !errors.Is(err, domainErr.ErrNotFound) {
		return nil, fmt.Errorf("file_asset_load_failed: %w", err)
	}
	resolved, err := u.Storage.ResolveFile(ctx, port.ResolveFileInput{FileURL: updated.FileURL})
	if err != nil {
		return nil, fmt.Errorf("upload_confirm_resolve_failed: %w", err)
	}
	if err := u.FileRepo.Create(ctx, &model.File{
		ID:        updated.ID,
		TenantID:  strings.TrimSpace(updated.TenantID),
		Bucket:    resolved.Bucket,
		ObjectKey: resolved.ObjectKey,
		Size:      updated.SizeBytes,
		Mime:      strings.TrimSpace(updated.ContentType),
		OwnerType: "user",
		OwnerID:   strings.TrimSpace(updated.UploadedBy),
		CreatedAt: updated.CreatedAt,
	}); err != nil {
		return nil, fmt.Errorf("file_asset_persist_failed: %w", err)
	}
	return &FileUploadConfirmOutput{Status: "ok", FileID: updated.ID, FileURL: updated.FileURL}, nil
}

type FileDownloadPresignInput struct {
	FileID  string
	FileURL string
}

type FileDownloadPresignOutput struct {
	DownloadURL string
}

type FileDownloadPresignUsecase struct {
	Storage  port.ObjectStorage
	FileRepo port.FileRepo
}

func (u *FileDownloadPresignUsecase) Execute(ctx context.Context, in FileDownloadPresignInput) (*FileDownloadPresignOutput, error) {
	if u.Storage == nil {
		return nil, domainErr.ErrValidation
	}
	fileID := strings.TrimSpace(in.FileID)
	fileURL := strings.TrimSpace(in.FileURL)
	if fileID == "" && fileURL == "" {
		return nil, domainErr.ErrValidation
	}
	input := port.PresignDownloadInput{FileURL: fileURL}
	if fileID != "" {
		if u.FileRepo == nil {
			return nil, domainErr.ErrValidation
		}
		f, err := u.FileRepo.GetByID(ctx, fileID)
		if err != nil {
			return nil, err
		}
		input.ObjectKey = strings.TrimSpace(f.ObjectKey)
	}
	out, err := u.Storage.PresignDownload(ctx, input)
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
