package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"service/internal/app/usecase"
	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
)

type mockAuthChallengeRepo struct {
	challenge *model.LoginChallenge
	user      *model.User
}

func (m *mockAuthChallengeRepo) CreateLoginChallenge(ctx context.Context, challenge *model.LoginChallenge) error {
	m.challenge = challenge
	return nil
}
func (m *mockAuthChallengeRepo) GetLoginChallengeByID(ctx context.Context, challengeID string) (*model.LoginChallenge, error) {
	if m.challenge == nil {
		return nil, errors.New("not found")
	}
	return m.challenge, nil
}
func (m *mockAuthChallengeRepo) IncreaseLoginChallengeAttempts(ctx context.Context, challengeID string) error {
	if m.challenge != nil {
		m.challenge.Attempts++
	}
	return nil
}
func (m *mockAuthChallengeRepo) MarkLoginChallengeVerified(ctx context.Context, challengeID string) error {
	now := time.Now()
	if m.challenge != nil {
		m.challenge.VerifiedAt = &now
	}
	return nil
}
func (m *mockAuthChallengeRepo) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	return m.user, nil
}

type mockOTPSender struct{}

func (m mockOTPSender) SendLoginOTP(ctx context.Context, phone, code string) error { return nil }

type captureLoginOTPSender struct {
	calls int
	phone string
	code  string
}

func (m *captureLoginOTPSender) SendLoginOTP(ctx context.Context, phone, code string) error {
	m.calls++
	m.phone = phone
	m.code = code
	return nil
}

func TestAuthLoginChallenge_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	authRepo := mockAuthRepo{
		user: &model.User{ID: "u1", Phone: "13800000003"},
		hash: string(hash),
	}
	chRepo := &mockAuthChallengeRepo{}
	u := &usecase.AuthLoginChallengeUsecase{
		Repo:      authRepo,
		Challenge: chRepo,
		Sender:    mockOTPSender{},
		IDGen:     func() string { return "c1" },
		Now:       func() time.Time { return time.Unix(1, 0) },
		MockCode:  "123456",
	}
	out, err := u.Execute(context.Background(), usecase.AuthLoginChallengeInput{
		Account:  "a",
		Password: "pass",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ChallengeID != "c1" || out.MaskedPhone == "" {
		t.Fatalf("unexpected output: %+v", out)
	}
	if chRepo.challenge == nil || chRepo.challenge.OTPHash == "" {
		t.Fatalf("challenge not created")
	}
}

func TestAuthLoginChallenge_SendsOTP(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	authRepo := mockAuthRepo{
		user: &model.User{ID: "u1", Phone: "13800000003"},
		hash: string(hash),
	}
	chRepo := &mockAuthChallengeRepo{}
	sender := &captureLoginOTPSender{}
	u := &usecase.AuthLoginChallengeUsecase{
		Repo:      authRepo,
		Challenge: chRepo,
		Sender:    sender,
		IDGen:     func() string { return "c1" },
		Now:       func() time.Time { return time.Unix(1, 0) },
		MockCode:  "123456",
	}
	_, err := u.Execute(context.Background(), usecase.AuthLoginChallengeInput{
		Account:  "a",
		Password: "pass",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sender.calls != 1 {
		t.Fatalf("expected sender called once, got %d", sender.calls)
	}
	if sender.phone != "13800000003" || sender.code != "123456" {
		t.Fatalf("unexpected sender payload: phone=%s code=%s", sender.phone, sender.code)
	}
}

func TestAuthLoginVerify_Success(t *testing.T) {
	chRepo := &mockAuthChallengeRepo{
		challenge: &model.LoginChallenge{
			ID:        "c1",
			UserID:    "u1",
			OTPHash:   "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92",
			ExpiresAt: time.Now().Add(time.Minute),
		},
		user: &model.User{ID: "u1"},
	}
	u := &usecase.AuthLoginVerifyUsecase{
		Challenge: chRepo,
		TokenGen: func(ctx context.Context, user *model.User) (string, string, int, error) {
			return "tok", "rt", 3600, nil
		},
		Now: time.Now,
	}
	out, err := u.Execute(context.Background(), usecase.AuthLoginVerifyInput{
		ChallengeID: "c1",
		OTPCode:     "123456",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AccessToken != "tok" {
		t.Fatalf("unexpected token: %+v", out)
	}
}

func TestAuthLoginVerify_WrongOTP(t *testing.T) {
	chRepo := &mockAuthChallengeRepo{
		challenge: &model.LoginChallenge{
			ID:        "c1",
			UserID:    "u1",
			OTPHash:   "x",
			ExpiresAt: time.Now().Add(time.Minute),
		},
	}
	u := &usecase.AuthLoginVerifyUsecase{Challenge: chRepo, TokenGen: func(ctx context.Context, user *model.User) (string, string, int, error) {
		return "", "", 3600, nil
	}, Now: time.Now}
	_, err := u.Execute(context.Background(), usecase.AuthLoginVerifyInput{ChallengeID: "c1", OTPCode: "123456"})
	if err != domainErr.ErrUnauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}
