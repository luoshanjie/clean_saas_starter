package usecase

import (
	"context"
	"strings"
	"unicode/utf8"

	domainErr "service/internal/domain/errors"
)

type AuthDisplayNameRepo interface {
	UpdateDisplayNameByUserID(ctx context.Context, userID, name string) error
}

type AuthUpdateDisplayNameInput struct {
	UserID string
	Name   string
}

type AuthUpdateDisplayNameOutput struct {
	Status string
	Name   string
}

type AuthUpdateDisplayNameUsecase struct {
	Repo AuthDisplayNameRepo
}

func (u *AuthUpdateDisplayNameUsecase) Execute(ctx context.Context, in AuthUpdateDisplayNameInput) (AuthUpdateDisplayNameOutput, error) {
	userID := strings.TrimSpace(in.UserID)
	name := strings.TrimSpace(in.Name)
	if userID == "" || name == "" {
		return AuthUpdateDisplayNameOutput{}, domainErr.ErrValidation
	}
	n := utf8.RuneCountInString(name)
	if n < 1 || n > 30 {
		return AuthUpdateDisplayNameOutput{}, domainErr.ErrInvalidDisplayName
	}
	if u.Repo == nil {
		return AuthUpdateDisplayNameOutput{}, domainErr.ErrUnauthenticated
	}
	if err := u.Repo.UpdateDisplayNameByUserID(ctx, userID, name); err != nil {
		return AuthUpdateDisplayNameOutput{}, domainErr.ErrUnauthenticated
	}
	return AuthUpdateDisplayNameOutput{
		Status: "ok",
		Name:   name,
	}, nil
}
