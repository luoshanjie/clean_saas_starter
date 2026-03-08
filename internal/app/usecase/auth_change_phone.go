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
	changePhoneOTPExpireSec   = 300
	changePhoneResendGapSec   = 60
	changePhoneOTPMaxAttempts = 5
)

type AuthChangePhoneChallengeInput struct {
	UserID      string
	Role        string
	NewPhone    string
	OldPassword string
}

type AuthChangePhoneChallengeOutput struct {
	ChallengeID    string
	MaskedNewPhone string
	ExpiresIn      int
	ResendIn       int
}

type AuthChangePhoneChallengeUsecase struct {
	AuthRepo      port.AuthRepo
	PhoneRepo     port.AuthPhoneRepo
	ChallengeRepo port.PhoneChangeChallengeRepo
	Sender        port.ChangePhoneOTPSender
	IDGen         func() string
	Now           func() time.Time
	MockCode      string
}

func (u *AuthChangePhoneChallengeUsecase) Execute(ctx context.Context, in AuthChangePhoneChallengeInput) (*AuthChangePhoneChallengeOutput, error) {
	if in.UserID == "" || in.NewPhone == "" || in.OldPassword == "" {
		return nil, domainErr.ErrValidation
	}
	if !allowChangePhoneRole(in.Role) {
		return nil, domainErr.ErrForbidden
	}
	newPhone, ok := normalizeMainlandPhone(in.NewPhone)
	if !ok {
		return nil, domainErr.ErrInvalidPhone
	}
	user, err := u.AuthRepo.GetUserByID(ctx, in.UserID)
	if err != nil || user == nil {
		return nil, domainErr.ErrUnauthenticated
	}
	oldHash, err := u.AuthRepo.GetPasswordHashByUserID(ctx, in.UserID)
	if err != nil {
		return nil, domainErr.ErrUnauthenticated
	}
	if err := bcrypt.CompareHashAndPassword([]byte(oldHash), []byte(in.OldPassword)); err != nil {
		return nil, domainErr.ErrInvalidOldPassword
	}
	if oldPhone, ok := normalizeMainlandPhone(user.Phone); ok && oldPhone == newPhone {
		return nil, domainErr.ErrValidation
	}
	exists, err := u.PhoneRepo.PhoneExists(ctx, newPhone)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domainErr.ErrPhoneExists
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}
	chID, err := canonicalUUIDText(u.IDGen())
	if err != nil {
		return nil, domainErr.ErrValidation
	}
	code := u.MockCode
	if code == "" {
		code = "123456"
	}
	if u.Sender != nil {
		if err := u.Sender.SendChangePhoneOTP(ctx, newPhone, code); err != nil {
			return nil, err
		}
	}
	challenge := &model.PhoneChangeChallenge{
		ID:          chID,
		UserID:      in.UserID,
		NewPhone:    newPhone,
		OTPHash:     hashOTP(code),
		ExpiresAt:   now.Add(changePhoneOTPExpireSec * time.Second),
		Attempts:    0,
		ResendCount: 0,
		LastSentAt:  now,
		CreatedAt:   now,
	}
	if err := u.ChallengeRepo.SavePhoneChangeChallenge(ctx, challenge); err != nil {
		return nil, err
	}
	return &AuthChangePhoneChallengeOutput{
		ChallengeID:    chID,
		MaskedNewPhone: maskPhone(newPhone),
		ExpiresIn:      changePhoneOTPExpireSec,
		ResendIn:       changePhoneResendGapSec,
	}, nil
}

type AuthChangePhoneResendInput struct {
	UserID      string
	Role        string
	ChallengeID string
}

type AuthChangePhoneResendOutput struct {
	ChallengeID    string
	MaskedNewPhone string
	ExpiresIn      int
	ResendIn       int
}

type AuthChangePhoneResendUsecase struct {
	ChallengeRepo port.PhoneChangeChallengeRepo
	Sender        port.ChangePhoneOTPSender
	Now           func() time.Time
	MockCode      string
}

func (u *AuthChangePhoneResendUsecase) Execute(ctx context.Context, in AuthChangePhoneResendInput) (*AuthChangePhoneResendOutput, error) {
	if in.UserID == "" || in.ChallengeID == "" {
		return nil, domainErr.ErrValidation
	}
	if !allowChangePhoneRole(in.Role) {
		return nil, domainErr.ErrForbidden
	}
	ch, err := u.ChallengeRepo.GetPhoneChangeChallengeByID(ctx, in.ChallengeID)
	if err != nil || ch == nil {
		return nil, domainErr.ErrChallengeNotFoundOrExpired
	}
	if ch.UserID != in.UserID {
		return nil, domainErr.ErrChallengeUserMismatch
	}
	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}
	if ch.VerifiedAt != nil || !now.Before(ch.ExpiresAt) {
		return nil, domainErr.ErrChallengeNotFoundOrExpired
	}
	if now.Before(ch.LastSentAt.Add(changePhoneResendGapSec * time.Second)) {
		return nil, domainErr.ErrValidation
	}
	code := u.MockCode
	if code == "" {
		code = "123456"
	}
	if u.Sender != nil {
		if err := u.Sender.SendChangePhoneOTP(ctx, ch.NewPhone, code); err != nil {
			return nil, err
		}
	}
	ch.OTPHash = hashOTP(code)
	ch.Attempts = 0
	ch.ResendCount++
	ch.LastSentAt = now
	ch.ExpiresAt = now.Add(changePhoneOTPExpireSec * time.Second)
	if err := u.ChallengeRepo.SavePhoneChangeChallenge(ctx, ch); err != nil {
		return nil, err
	}
	return &AuthChangePhoneResendOutput{
		ChallengeID:    ch.ID,
		MaskedNewPhone: maskPhone(ch.NewPhone),
		ExpiresIn:      changePhoneOTPExpireSec,
		ResendIn:       changePhoneResendGapSec,
	}, nil
}

type AuthChangePhoneVerifyInput struct {
	UserID      string
	Role        string
	ChallengeID string
	OTPCode     string
}

type AuthChangePhoneVerifyOutput struct {
	Status            string
	ForceRelogin      bool
	BeforePhoneMasked string
	AfterPhoneMasked  string
}

type AuthChangePhoneVerifyUsecase struct {
	AuthRepo      port.AuthRepo
	PhoneRepo     port.AuthPhoneRepo
	ChallengeRepo port.PhoneChangeChallengeRepo
	Now           func() time.Time
}

func (u *AuthChangePhoneVerifyUsecase) Execute(ctx context.Context, in AuthChangePhoneVerifyInput) (*AuthChangePhoneVerifyOutput, error) {
	if in.UserID == "" || in.ChallengeID == "" || in.OTPCode == "" {
		return nil, domainErr.ErrValidation
	}
	if !allowChangePhoneRole(in.Role) {
		return nil, domainErr.ErrForbidden
	}
	ch, err := u.ChallengeRepo.GetPhoneChangeChallengeByID(ctx, in.ChallengeID)
	if err != nil || ch == nil {
		return nil, domainErr.ErrChallengeNotFoundOrExpired
	}
	if ch.UserID != in.UserID {
		return nil, domainErr.ErrChallengeUserMismatch
	}
	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}
	if ch.VerifiedAt != nil || !now.Before(ch.ExpiresAt) {
		return nil, domainErr.ErrChallengeNotFoundOrExpired
	}
	if ch.Attempts >= changePhoneOTPMaxAttempts {
		return nil, domainErr.ErrOTPAttemptsExceeded
	}
	if hashOTP(in.OTPCode) != ch.OTPHash {
		_ = u.ChallengeRepo.IncreasePhoneChangeChallengeAttempts(ctx, ch.ID)
		if ch.Attempts+1 >= changePhoneOTPMaxAttempts {
			return nil, domainErr.ErrOTPAttemptsExceeded
		}
		return nil, domainErr.ErrOTPInvalidOrExpired
	}
	exists, err := u.PhoneRepo.PhoneExists(ctx, ch.NewPhone)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domainErr.ErrPhoneExists
	}
	beforeMasked := ""
	if u.AuthRepo != nil {
		if user, err := u.AuthRepo.GetUserByID(ctx, in.UserID); err == nil && user != nil {
			if p, ok := normalizeMainlandPhone(user.Phone); ok {
				beforeMasked = maskPhone(p)
			}
		}
	}
	if err := u.PhoneRepo.UpdatePhoneByUserID(ctx, in.UserID, ch.NewPhone); err != nil {
		return nil, err
	}
	if err := u.ChallengeRepo.MarkPhoneChangeChallengeVerified(ctx, ch.ID); err != nil {
		return nil, err
	}
	return &AuthChangePhoneVerifyOutput{
		Status:            "ok",
		ForceRelogin:      true,
		BeforePhoneMasked: beforeMasked,
		AfterPhoneMasked:  maskPhone(ch.NewPhone),
	}, nil
}

func allowChangePhoneRole(role string) bool {
	return role == "platform_op" || role == "tenant_admin" || role == "tenant_member"
}
