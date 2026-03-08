package usecase

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
	"service/internal/domain/port"
)

const (
	loginOTPExpireSec = 300
)

type AuthLoginChallengeInput struct {
	Account  string
	Password string
}

type AuthLoginChallengeOutput struct {
	ChallengeID string
	MaskedPhone string
	ExpiresIn   int
}

type AuthLoginChallengeUsecase struct {
	Repo      port.AuthRepo
	Challenge port.AuthChallengeRepo
	Sender    port.LoginOTPSender
	IDGen     func() string
	Now       func() time.Time
	MockCode  string
}

func (u *AuthLoginChallengeUsecase) Execute(ctx context.Context, in AuthLoginChallengeInput) (*AuthLoginChallengeOutput, error) {
	if in.Account == "" || in.Password == "" {
		return nil, domainErr.ErrValidation
	}
	user, hash, err := u.Repo.GetUserByAccount(ctx, in.Account)
	if err != nil {
		return nil, domainErr.ErrUnauthenticated
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)); err != nil {
		return nil, domainErr.ErrUnauthenticated
	}
	phone, ok := normalizeMainlandPhone(user.Phone)
	if !ok {
		return nil, domainErr.ErrValidation
	}

	code := u.MockCode
	if code == "" {
		code = "123456"
	}
	if u.Sender != nil {
		if err := u.Sender.SendLoginOTP(ctx, phone, code); err != nil {
			return nil, err
		}
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}
	challengeID := ""
	if u.IDGen != nil {
		normalized, err := canonicalUUIDText(u.IDGen())
		if err != nil {
			return nil, domainErr.ErrValidation
		}
		challengeID = normalized
	}
	if challengeID == "" {
		return nil, domainErr.ErrValidation
	}
	challenge := &model.LoginChallenge{
		ID:        challengeID,
		UserID:    user.ID,
		OTPHash:   hashOTP(code),
		ExpiresAt: now.Add(loginOTPExpireSec * time.Second),
		Attempts:  0,
		CreatedAt: now,
	}
	if err := u.Challenge.CreateLoginChallenge(ctx, challenge); err != nil {
		return nil, err
	}

	return &AuthLoginChallengeOutput{
		ChallengeID: challengeID,
		MaskedPhone: maskPhone(phone),
		ExpiresIn:   loginOTPExpireSec,
	}, nil
}
