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

type changePhoneAuthRepo struct {
	user *model.User
	hash string
}

func (m changePhoneAuthRepo) GetUserByAccount(ctx context.Context, account string) (*model.User, string, error) {
	return nil, "", errors.New("not used")
}
func (m changePhoneAuthRepo) GetTokenVersionByUserID(ctx context.Context, userID string) (int, error) {
	return 0, nil
}
func (m changePhoneAuthRepo) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	if m.user == nil {
		return nil, errors.New("not found")
	}
	return m.user, nil
}
func (m changePhoneAuthRepo) GetPasswordHashByUserID(ctx context.Context, userID string) (string, error) {
	return m.hash, nil
}
func (m changePhoneAuthRepo) UpdatePasswordByUserID(ctx context.Context, userID, passwordHash string, mustChange bool) error {
	return nil
}

type changePhoneRepo struct {
	exists       bool
	updatedUser  string
	updatedPhone string
}

func (m *changePhoneRepo) PhoneExists(ctx context.Context, phone string) (bool, error) {
	return m.exists, nil
}
func (m *changePhoneRepo) UpdatePhoneByUserID(ctx context.Context, userID, phone string) error {
	m.updatedUser = userID
	m.updatedPhone = phone
	return nil
}

type changePhoneChallengeRepo struct {
	byID map[string]*model.PhoneChangeChallenge
}

type noopChangePhoneSender struct{}

func (noopChangePhoneSender) SendChangePhoneOTP(ctx context.Context, phone, code string) error {
	return nil
}

type captureChangePhoneSender struct {
	calls int
	phone string
	code  string
}

func (m *captureChangePhoneSender) SendChangePhoneOTP(ctx context.Context, phone, code string) error {
	m.calls++
	m.phone = phone
	m.code = code
	return nil
}

func (m *changePhoneChallengeRepo) SavePhoneChangeChallenge(ctx context.Context, challenge *model.PhoneChangeChallenge) error {
	if m.byID == nil {
		m.byID = map[string]*model.PhoneChangeChallenge{}
	}
	for id, item := range m.byID {
		if item.UserID == challenge.UserID && item.VerifiedAt == nil {
			delete(m.byID, id)
		}
	}
	cp := *challenge
	m.byID[challenge.ID] = &cp
	return nil
}

func (m *changePhoneChallengeRepo) GetPhoneChangeChallengeByID(ctx context.Context, challengeID string) (*model.PhoneChangeChallenge, error) {
	if m.byID == nil {
		return nil, errors.New("not found")
	}
	v, ok := m.byID[challengeID]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := *v
	return &cp, nil
}

func (m *changePhoneChallengeRepo) IncreasePhoneChangeChallengeAttempts(ctx context.Context, challengeID string) error {
	if m.byID != nil && m.byID[challengeID] != nil {
		m.byID[challengeID].Attempts++
	}
	return nil
}

func (m *changePhoneChallengeRepo) MarkPhoneChangeChallengeVerified(ctx context.Context, challengeID string) error {
	if m.byID != nil && m.byID[challengeID] != nil {
		now := time.Now()
		m.byID[challengeID].VerifiedAt = &now
	}
	return nil
}

func TestAuthChangePhoneChallenge_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.DefaultCost)
	authRepo := changePhoneAuthRepo{
		user: &model.User{ID: "u1", Role: "tenant_member", Phone: "13800000003"},
		hash: string(hash),
	}
	phoneRepo := &changePhoneRepo{}
	chRepo := &changePhoneChallengeRepo{}
	u := &usecase.AuthChangePhoneChallengeUsecase{
		AuthRepo:      authRepo,
		PhoneRepo:     phoneRepo,
		ChallengeRepo: chRepo,
		Sender:        noopChangePhoneSender{},
		IDGen:         func() string { return "00000000-0000-0000-0000-000000000999" },
		Now:           func() time.Time { return time.Unix(1, 0) },
		MockCode:      "123456",
	}
	out, err := u.Execute(context.Background(), usecase.AuthChangePhoneChallengeInput{
		UserID:      "u1",
		Role:        "tenant_member",
		NewPhone:    "13800000004",
		OldPassword: "pass123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ChallengeID == "" || out.MaskedNewPhone == "" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestAuthChangePhoneChallenge_SendsOTP(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.DefaultCost)
	authRepo := changePhoneAuthRepo{
		user: &model.User{ID: "u1", Role: "tenant_member", Phone: "13800000003"},
		hash: string(hash),
	}
	phoneRepo := &changePhoneRepo{}
	chRepo := &changePhoneChallengeRepo{}
	sender := &captureChangePhoneSender{}
	u := &usecase.AuthChangePhoneChallengeUsecase{
		AuthRepo:      authRepo,
		PhoneRepo:     phoneRepo,
		ChallengeRepo: chRepo,
		Sender:        sender,
		IDGen:         func() string { return "00000000-0000-0000-0000-000000000997" },
		Now:           func() time.Time { return time.Unix(1, 0) },
		MockCode:      "123456",
	}
	_, err := u.Execute(context.Background(), usecase.AuthChangePhoneChallengeInput{
		UserID:      "u1",
		Role:        "tenant_member",
		NewPhone:    "13800000004",
		OldPassword: "pass123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sender.calls != 1 {
		t.Fatalf("expected sender called once, got %d", sender.calls)
	}
	if sender.phone != "13800000004" || sender.code != "123456" {
		t.Fatalf("unexpected sender payload: phone=%s code=%s", sender.phone, sender.code)
	}
}

func TestAuthChangePhoneChallenge_InvalidOldPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.DefaultCost)
	authRepo := changePhoneAuthRepo{
		user: &model.User{ID: "u1", Role: "tenant_member", Phone: "13800000003"},
		hash: string(hash),
	}
	u := &usecase.AuthChangePhoneChallengeUsecase{
		AuthRepo:      authRepo,
		PhoneRepo:     &changePhoneRepo{},
		ChallengeRepo: &changePhoneChallengeRepo{},
		Sender:        noopChangePhoneSender{},
		IDGen:         func() string { return "00000000-0000-0000-0000-000000000998" },
		Now:           time.Now,
	}
	_, err := u.Execute(context.Background(), usecase.AuthChangePhoneChallengeInput{
		UserID:      "u1",
		Role:        "tenant_member",
		NewPhone:    "13800000004",
		OldPassword: "wrong",
	})
	if err != domainErr.ErrInvalidOldPassword {
		t.Fatalf("expected invalid old password, got %v", err)
	}
}

func TestAuthChangePhoneVerify_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.DefaultCost)
	authRepo := changePhoneAuthRepo{
		user: &model.User{ID: "u1", Role: "tenant_member", Phone: "13800000003"},
		hash: string(hash),
	}
	phoneRepo := &changePhoneRepo{}
	chRepo := &changePhoneChallengeRepo{
		byID: map[string]*model.PhoneChangeChallenge{
			"c1": {
				ID:        "c1",
				UserID:    "u1",
				NewPhone:  "13800000004",
				OTPHash:   "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92",
				ExpiresAt: time.Now().Add(time.Minute),
			},
		},
	}
	u := &usecase.AuthChangePhoneVerifyUsecase{
		AuthRepo:      authRepo,
		PhoneRepo:     phoneRepo,
		ChallengeRepo: chRepo,
		Now:           time.Now,
	}
	out, err := u.Execute(context.Background(), usecase.AuthChangePhoneVerifyInput{
		UserID:      "u1",
		Role:        "tenant_member",
		ChallengeID: "c1",
		OTPCode:     "123456",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.ForceRelogin || out.Status != "ok" {
		t.Fatalf("unexpected output: %+v", out)
	}
	if phoneRepo.updatedPhone != "13800000004" {
		t.Fatalf("phone not updated")
	}
}

func TestAuthChangePhoneVerify_UserMismatch(t *testing.T) {
	chRepo := &changePhoneChallengeRepo{
		byID: map[string]*model.PhoneChangeChallenge{
			"c1": {
				ID:        "c1",
				UserID:    "u2",
				NewPhone:  "13800000004",
				OTPHash:   "x",
				ExpiresAt: time.Now().Add(time.Minute),
			},
		},
	}
	u := &usecase.AuthChangePhoneVerifyUsecase{
		PhoneRepo:     &changePhoneRepo{},
		ChallengeRepo: chRepo,
		Now:           time.Now,
	}
	_, err := u.Execute(context.Background(), usecase.AuthChangePhoneVerifyInput{
		UserID:      "u1",
		Role:        "tenant_member",
		ChallengeID: "c1",
		OTPCode:     "123456",
	})
	if err != domainErr.ErrChallengeUserMismatch {
		t.Fatalf("expected mismatch, got %v", err)
	}
}
