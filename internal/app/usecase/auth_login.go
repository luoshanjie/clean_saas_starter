package usecase

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
	"service/internal/domain/port"
)

type AuthLoginInput struct {
	Account  string
	Password string
}

type AuthLoginOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	User         *model.User
}

type AuthLoginUsecase struct {
	Repo     port.AuthRepo
	TokenGen func(ctx context.Context, user *model.User) (access, refresh string, expiresIn int, err error)
}

// AuthLogin 负责认证闭环：校验账号密码 -> 生成 JWT。
func (u *AuthLoginUsecase) Execute(ctx context.Context, in AuthLoginInput) (*AuthLoginOutput, error) {
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

	access, refresh, expiresIn, err := u.TokenGen(ctx, user)
	if err != nil {
		return nil, err
	}

	return &AuthLoginOutput{AccessToken: access, RefreshToken: refresh, ExpiresIn: expiresIn, User: user}, nil
}
