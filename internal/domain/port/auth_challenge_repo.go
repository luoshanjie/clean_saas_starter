package port

import (
	"context"

	"service/internal/domain/model"
)

type AuthChallengeRepo interface {
	CreateLoginChallenge(ctx context.Context, challenge *model.LoginChallenge) error
	GetLoginChallengeByID(ctx context.Context, challengeID string) (*model.LoginChallenge, error)
	IncreaseLoginChallengeAttempts(ctx context.Context, challengeID string) error
	MarkLoginChallengeVerified(ctx context.Context, challengeID string) error
	GetUserByID(ctx context.Context, userID string) (*model.User, error)
}
