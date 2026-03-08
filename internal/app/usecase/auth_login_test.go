package usecase_test

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"service/internal/app/usecase"
	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
)

type mockAuthRepo struct {
	user *model.User
	hash string
	err  error
}

func (m mockAuthRepo) GetUserByAccount(ctx context.Context, account string) (*model.User, string, error) {
	if m.err != nil {
		return nil, "", m.err
	}
	return m.user, m.hash, nil
}
func (m mockAuthRepo) GetTokenVersionByUserID(ctx context.Context, userID string) (int, error) {
	return 0, nil
}
func (m mockAuthRepo) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	return m.user, nil
}
func (m mockAuthRepo) GetPasswordHashByUserID(ctx context.Context, userID string) (string, error) {
	return m.hash, nil
}
func (m mockAuthRepo) UpdatePasswordByUserID(ctx context.Context, userID, passwordHash string, mustChange bool) error {
	return nil
}

func TestAuthLogin_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	repo := mockAuthRepo{
		user: &model.User{ID: "u1", Role: "tenant_member", ScopeType: "tenant", TenantID: "t1"},
		hash: string(hash),
	}

	u := &usecase.AuthLoginUsecase{
		Repo: repo,
		TokenGen: func(ctx context.Context, user *model.User) (string, string, int, error) {
			return "tok", "rt", 3600, nil
		},
	}

	out, err := u.Execute(context.Background(), usecase.AuthLoginInput{Account: "a", Password: "pass"})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if out.AccessToken != "tok" || out.RefreshToken != "rt" {
		t.Fatalf("unexpected token: %v", out.AccessToken)
	}
	if out.User.ID != "u1" {
		t.Fatalf("unexpected user: %+v", out.User)
	}
}

func TestAuthLogin_WrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	repo := mockAuthRepo{
		user: &model.User{ID: "u1"},
		hash: string(hash),
	}

	u := &usecase.AuthLoginUsecase{
		Repo:     repo,
		TokenGen: func(ctx context.Context, user *model.User) (string, string, int, error) { return "", "", 3600, nil },
	}

	_, err := u.Execute(context.Background(), usecase.AuthLoginInput{Account: "a", Password: "bad"})
	if err != domainErr.ErrUnauthenticated {
		t.Fatalf("expected unauthenticated, got: %v", err)
	}
}

func TestAuthLogin_ValidationError(t *testing.T) {
	u := &usecase.AuthLoginUsecase{}
	_, err := u.Execute(context.Background(), usecase.AuthLoginInput{Account: "", Password: ""})
	if err != domainErr.ErrValidation {
		t.Fatalf("expected validation error, got: %v", err)
	}
}

func TestAuthLogin_RepoError(t *testing.T) {
	repo := mockAuthRepo{err: errors.New("db")}
	u := &usecase.AuthLoginUsecase{Repo: repo, TokenGen: func(ctx context.Context, user *model.User) (string, string, int, error) { return "", "", 3600, nil }}
	_, err := u.Execute(context.Background(), usecase.AuthLoginInput{Account: "a", Password: "pass"})
	if err != domainErr.ErrUnauthenticated {
		t.Fatalf("expected unauthenticated, got: %v", err)
	}
}

func TestAuthLogin_TokenGenError(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	repo := mockAuthRepo{
		user: &model.User{ID: "u1"},
		hash: string(hash),
	}

	u := &usecase.AuthLoginUsecase{
		Repo: repo,
		TokenGen: func(ctx context.Context, user *model.User) (string, string, int, error) {
			return "", "", 0, errors.New("sign error")
		},
	}

	_, err := u.Execute(context.Background(), usecase.AuthLoginInput{Account: "a", Password: "pass"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
