package port

import (
	"context"
	"time"

	"service/internal/domain/model"
)

type FileUploadSessionRepo interface {
	Create(ctx context.Context, s *model.FileUploadSession) error
	GetByID(ctx context.Context, id string) (*model.FileUploadSession, error)
	Confirm(ctx context.Context, id string, confirmedAt time.Time) (*model.FileUploadSession, error)
	ListExpiredPending(ctx context.Context, now time.Time, limit int) ([]*model.FileUploadSession, error)
	MarkCleaned(ctx context.Context, id string, cleanedAt time.Time) error
	SetLastError(ctx context.Context, id, lastError string, updatedAt time.Time) error
}
