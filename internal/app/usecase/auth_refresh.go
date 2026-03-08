package usecase

import (
	"context"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
	"service/internal/domain/port"
)

type AuthRefreshInput struct {
	RefreshToken string
}

type AuthRefreshOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

type AuthRefreshUsecase struct {
	ParseRefresh func(token string) (*model.User, error)
	TokenGen     func(ctx context.Context, user *model.User) (access, refresh string, expiresIn int, err error)
	Repo         port.AuthRepo
}

// AuthRefresh 校验 refresh_token 并签发新的 token 对。
func (u *AuthRefreshUsecase) Execute(ctx context.Context, in AuthRefreshInput) (*AuthRefreshOutput, error) {
	if in.RefreshToken == "" {
		return nil, domainErr.ErrValidation
	}
	user, err := u.ParseRefresh(in.RefreshToken)
	if err != nil {
		return nil, domainErr.ErrUnauthenticated
	}
	if u.Repo != nil {
		v, err := u.Repo.GetTokenVersionByUserID(ctx, user.ID)
		if err != nil {
			return nil, domainErr.ErrUnauthenticated
		}
		if v != user.TokenVersion {
			return nil, domainErr.ErrUnauthenticated
		}
	}
	access, refresh, expiresIn, err := u.TokenGen(ctx, user)
	if err != nil {
		return nil, err
	}
	return &AuthRefreshOutput{AccessToken: access, RefreshToken: refresh, ExpiresIn: expiresIn}, nil
}
