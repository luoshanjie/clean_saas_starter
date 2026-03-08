package usecase_test

import (
	"context"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"service/internal/app/usecase"
	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
)

type mockAuthRepoChangePassword struct {
	hash       string
	updated    bool
	mustChange bool
}

func (m *mockAuthRepoChangePassword) GetUserByAccount(ctx context.Context, account string) (*model.User, string, error) {
	return nil, "", nil
}
func (m *mockAuthRepoChangePassword) GetTokenVersionByUserID(ctx context.Context, userID string) (int, error) {
	return 0, nil
}
func (m *mockAuthRepoChangePassword) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	return &model.User{ID: userID}, nil
}
func (m *mockAuthRepoChangePassword) GetPasswordHashByUserID(ctx context.Context, userID string) (string, error) {
	return m.hash, nil
}
func (m *mockAuthRepoChangePassword) UpdatePasswordByUserID(ctx context.Context, userID, passwordHash string, mustChange bool) error {
	m.updated = true
	m.hash = passwordHash
	m.mustChange = mustChange
	return nil
}

func TestAuthChangePassword_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("oldpass123"), bcrypt.DefaultCost)
	repo := &mockAuthRepoChangePassword{hash: string(hash)}
	u := &usecase.AuthChangePasswordUsecase{Repo: repo}
	err := u.Execute(context.Background(), usecase.AuthChangePasswordInput{UserID: "u1", OldPassword: "oldpass123", NewPassword: "newpass123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.updated || repo.mustChange {
		t.Fatalf("expected updated with mustChange=false")
	}
}

func TestAuthChangePassword_InvalidOldPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("oldpass123"), bcrypt.DefaultCost)
	repo := &mockAuthRepoChangePassword{hash: string(hash)}
	u := &usecase.AuthChangePasswordUsecase{Repo: repo}
	err := u.Execute(context.Background(), usecase.AuthChangePasswordInput{UserID: "u1", OldPassword: "wrong", NewPassword: "newpass123"})
	if err != domainErr.ErrInvalidOldPassword {
		t.Fatalf("expected invalid_old_password, got %v", err)
	}
}
