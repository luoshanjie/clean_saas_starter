package usecase

import (
	"context"
	"errors"
	"testing"

	domainErr "service/internal/domain/errors"
)

type mockAuthDisplayNameRepo struct {
	updatedUserID string
	updatedName   string
	err           error
}

func (m *mockAuthDisplayNameRepo) UpdateDisplayNameByUserID(ctx context.Context, userID, name string) error {
	m.updatedUserID = userID
	m.updatedName = name
	return m.err
}

func TestAuthUpdateDisplayNameUsecase_Success(t *testing.T) {
	repo := &mockAuthDisplayNameRepo{}
	u := &AuthUpdateDisplayNameUsecase{Repo: repo}
	out, err := u.Execute(context.Background(), AuthUpdateDisplayNameInput{
		UserID: "u1",
		Name:   "诸葛亮",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Status != "ok" || out.Name != "诸葛亮" {
		t.Fatalf("unexpected output: %+v", out)
	}
	if repo.updatedUserID != "u1" || repo.updatedName != "诸葛亮" {
		t.Fatalf("repo not called as expected: %+v", repo)
	}
}

func TestAuthUpdateDisplayNameUsecase_Invalid(t *testing.T) {
	u := &AuthUpdateDisplayNameUsecase{Repo: &mockAuthDisplayNameRepo{}}
	if _, err := u.Execute(context.Background(), AuthUpdateDisplayNameInput{
		UserID: "",
		Name:   "诸葛亮",
	}); err != domainErr.ErrValidation {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	if _, err := u.Execute(context.Background(), AuthUpdateDisplayNameInput{
		UserID: "u1",
		Name:   "",
	}); err != domainErr.ErrValidation {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	if _, err := u.Execute(context.Background(), AuthUpdateDisplayNameInput{
		UserID: "u1",
		Name:   "1234567890123456789012345678901",
	}); err != domainErr.ErrInvalidDisplayName {
		t.Fatalf("expected ErrInvalidDisplayName, got %v", err)
	}
}

func TestAuthUpdateDisplayNameUsecase_RepoError(t *testing.T) {
	repo := &mockAuthDisplayNameRepo{err: errors.New("db")}
	u := &AuthUpdateDisplayNameUsecase{Repo: repo}
	_, err := u.Execute(context.Background(), AuthUpdateDisplayNameInput{
		UserID: "u1",
		Name:   "诸葛亮",
	})
	if err != domainErr.ErrUnauthenticated {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}
}
