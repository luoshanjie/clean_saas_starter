package port

import (
	"context"

	"service/internal/domain/model"
)

type PhoneChangeChallengeRepo interface {
	SavePhoneChangeChallenge(ctx context.Context, challenge *model.PhoneChangeChallenge) error
	GetPhoneChangeChallengeByID(ctx context.Context, challengeID string) (*model.PhoneChangeChallenge, error)
	IncreasePhoneChangeChallengeAttempts(ctx context.Context, challengeID string) error
	MarkPhoneChangeChallengeVerified(ctx context.Context, challengeID string) error
}
