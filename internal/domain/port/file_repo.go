package port

import (
	"context"

	"service/internal/domain/model"
)

type FileRepo interface {
	Create(ctx context.Context, f *model.File) error
}
