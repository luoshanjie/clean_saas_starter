package port

import (
	"context"

	"service/internal/domain/model"
)

type AuthRepo interface {
	GetUserByAccount(ctx context.Context, account string) (*model.User, string, error)
	GetTokenVersionByUserID(ctx context.Context, userID string) (int, error)
	GetUserByID(ctx context.Context, userID string) (*model.User, error)
	GetPasswordHashByUserID(ctx context.Context, userID string) (string, error)
	UpdatePasswordByUserID(ctx context.Context, userID, passwordHash string, mustChange bool) error
}
