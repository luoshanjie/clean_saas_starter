package port

import (
	"context"

	"service/internal/domain/model"
)

type FileRepo interface {
	Create(ctx context.Context, f *model.File) error
	GetByID(ctx context.Context, id string) (*model.File, error)
	DeleteByID(ctx context.Context, id string) error
}
