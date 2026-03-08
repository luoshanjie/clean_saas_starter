package usecase

import (
	"context"
	"time"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
	"service/internal/domain/port"
)

const (
	loginOTPMaxAttempts = 5
)

type AuthLoginVerifyInput struct {
	ChallengeID string
	OTPCode     string
}

type AuthLoginVerifyOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	User         *model.User
}

type AuthLoginVerifyUsecase struct {
	Challenge port.AuthChallengeRepo
	TokenGen  func(ctx context.Context, user *model.User) (access, refresh string, expiresIn int, err error)
	Now       func() time.Time
}

func (u *AuthLoginVerifyUsecase) Execute(ctx context.Context, in AuthLoginVerifyInput) (*AuthLoginVerifyOutput, error) {
	if in.ChallengeID == "" || in.OTPCode == "" {
		return nil, domainErr.ErrValidation
	}
	ch, err := u.Challenge.GetLoginChallengeByID(ctx, in.ChallengeID)
	if err != nil || ch == nil {
		return nil, domainErr.ErrUnauthenticated
	}
	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}
	if ch.VerifiedAt != nil || now.After(ch.ExpiresAt) || ch.Attempts >= loginOTPMaxAttempts {
		return nil, domainErr.ErrUnauthenticated
	}

	if hashOTP(in.OTPCode) != ch.OTPHash {
		_ = u.Challenge.IncreaseLoginChallengeAttempts(ctx, ch.ID)
		return nil, domainErr.ErrUnauthenticated
	}
	if err := u.Challenge.MarkLoginChallengeVerified(ctx, ch.ID); err != nil {
		return nil, err
	}

	user, err := u.Challenge.GetUserByID(ctx, ch.UserID)
	if err != nil {
		return nil, domainErr.ErrUnauthenticated
	}
	access, refresh, expiresIn, err := u.TokenGen(ctx, user)
	if err != nil {
		return nil, err
	}
	return &AuthLoginVerifyOutput{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    expiresIn,
		User:         user,
	}, nil
}
