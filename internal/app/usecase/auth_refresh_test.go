package usecase_test

import (
	"context"
	"errors"
	"testing"

	"service/internal/app/usecase"
	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
)

type mockAuthRepoRefresh struct{ v int }

func (m mockAuthRepoRefresh) GetUserByAccount(ctx context.Context, account string) (*model.User, string, error) {
	return nil, "", nil
}
func (m mockAuthRepoRefresh) GetTokenVersionByUserID(ctx context.Context, userID string) (int, error) {
	return m.v, nil
}
func (m mockAuthRepoRefresh) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	return &model.User{ID: userID}, nil
}
func (m mockAuthRepoRefresh) GetPasswordHashByUserID(ctx context.Context, userID string) (string, error) {
	return "", nil
}
func (m mockAuthRepoRefresh) UpdatePasswordByUserID(ctx context.Context, userID, passwordHash string, mustChange bool) error {
	return nil
}

func TestAuthRefresh_Success(t *testing.T) {
	u := &usecase.AuthRefreshUsecase{
		ParseRefresh: func(token string) (*model.User, error) {
			return &model.User{ID: "u1", TokenVersion: 1}, nil
		},
		TokenGen: func(ctx context.Context, user *model.User) (string, string, int, error) {
			return "a", "r", 3600, nil
		},
		Repo: mockAuthRepoRefresh{v: 1},
	}

	out, err := u.Execute(context.Background(), usecase.AuthRefreshInput{RefreshToken: "rt"})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if out.AccessToken != "a" || out.RefreshToken != "r" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestAuthRefresh_EmptyToken(t *testing.T) {
	u := &usecase.AuthRefreshUsecase{}
	_, err := u.Execute(context.Background(), usecase.AuthRefreshInput{})
	if err != domainErr.ErrValidation {
		t.Fatalf("expected validation error, got: %v", err)
	}
}

func TestAuthRefresh_ParseError(t *testing.T) {
	u := &usecase.AuthRefreshUsecase{
		ParseRefresh: func(token string) (*model.User, error) { return nil, errors.New("bad") },
		TokenGen:     func(ctx context.Context, user *model.User) (string, string, int, error) { return "", "", 3600, nil },
	}
	_, err := u.Execute(context.Background(), usecase.AuthRefreshInput{RefreshToken: "rt"})
	if err != domainErr.ErrUnauthenticated {
		t.Fatalf("expected unauthenticated, got: %v", err)
	}
}

func TestAuthRefresh_TokenGenError(t *testing.T) {
	u := &usecase.AuthRefreshUsecase{
		ParseRefresh: func(token string) (*model.User, error) { return &model.User{ID: "u1", TokenVersion: 1}, nil },
		Repo:         mockAuthRepoRefresh{v: 1},
		TokenGen: func(ctx context.Context, user *model.User) (string, string, int, error) {
			return "", "", 0, errors.New("sign")
		},
	}
	_, err := u.Execute(context.Background(), usecase.AuthRefreshInput{RefreshToken: "rt"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestAuthRefresh_TokenVersionMismatch(t *testing.T) {
	u := &usecase.AuthRefreshUsecase{
		ParseRefresh: func(token string) (*model.User, error) { return &model.User{ID: "u1", TokenVersion: 1}, nil },
		TokenGen:     func(ctx context.Context, user *model.User) (string, string, int, error) { return "", "", 3600, nil },
		Repo:         mockAuthRepoRefresh{v: 2},
	}
	_, err := u.Execute(context.Background(), usecase.AuthRefreshInput{RefreshToken: "rt"})
	if err != domainErr.ErrUnauthenticated {
		t.Fatalf("expected unauthenticated, got: %v", err)
	}
}
