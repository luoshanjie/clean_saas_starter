package usecase

import (
	"context"
	"strings"

	"golang.org/x/crypto/bcrypt"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/port"
)

type AuthChangePasswordInput struct {
	UserID      string
	OldPassword string
	NewPassword string
}

type AuthChangePasswordUsecase struct {
	Repo port.AuthRepo
}

func (u *AuthChangePasswordUsecase) Execute(ctx context.Context, in AuthChangePasswordInput) error {
	if strings.TrimSpace(in.UserID) == "" || strings.TrimSpace(in.OldPassword) == "" || strings.TrimSpace(in.NewPassword) == "" {
		return domainErr.ErrValidation
	}
	if len(in.NewPassword) < 8 || len(in.NewPassword) > 64 {
		return domainErr.ErrInvalidNewPassword
	}
	if in.OldPassword == in.NewPassword {
		return domainErr.ErrInvalidNewPassword
	}
	hash, err := u.Repo.GetPasswordHashByUserID(ctx, strings.TrimSpace(in.UserID))
	if err != nil {
		return domainErr.ErrUnauthenticated
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.OldPassword)); err != nil {
		return domainErr.ErrInvalidOldPassword
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return u.Repo.UpdatePasswordByUserID(ctx, strings.TrimSpace(in.UserID), string(newHash), false)
}
