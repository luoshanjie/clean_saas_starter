package handler_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"service/internal/app/usecase"
	"service/internal/delivery/http/handler"
	"service/internal/delivery/http/middleware"
	"service/internal/domain/model"
	smsrepo "service/internal/repo/sms"
)

type mockAuthRepo struct {
	user                 *model.User
	hash                 string
	challenge            *model.LoginChallenge
	phoneChangeChallenge *model.PhoneChangeChallenge
	phoneExists          bool
	updatedPhone         string
	updatedDisplayName   string
}

func (m *mockAuthRepo) GetUserByAccount(ctx context.Context, account string) (*model.User, string, error) {
	return m.user, m.hash, nil
}
func (m *mockAuthRepo) GetTokenVersionByUserID(ctx context.Context, userID string) (int, error) {
	return 0, nil
}
func (m *mockAuthRepo) CreateLoginChallenge(ctx context.Context, challenge *model.LoginChallenge) error {
	m.challenge = challenge
	return nil
}
func (m *mockAuthRepo) GetLoginChallengeByID(ctx context.Context, challengeID string) (*model.LoginChallenge, error) {
	return m.challenge, nil
}
func (m *mockAuthRepo) IncreaseLoginChallengeAttempts(ctx context.Context, challengeID string) error {
	if m.challenge != nil {
		m.challenge.Attempts++
	}
	return nil
}
func (m *mockAuthRepo) MarkLoginChallengeVerified(ctx context.Context, challengeID string) error {
	now := time.Now()
	if m.challenge != nil {
		m.challenge.VerifiedAt = &now
	}
	return nil
}
func (m *mockAuthRepo) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	return m.user, nil
}
func (m *mockAuthRepo) GetPasswordHashByUserID(ctx context.Context, userID string) (string, error) {
	return m.hash, nil
}
func (m *mockAuthRepo) UpdatePasswordByUserID(ctx context.Context, userID, passwordHash string, mustChange bool) error {
	m.hash = passwordHash
	return nil
}
func (m *mockAuthRepo) PhoneExists(ctx context.Context, phone string) (bool, error) {
	return m.phoneExists, nil
}
func (m *mockAuthRepo) UpdatePhoneByUserID(ctx context.Context, userID, phone string) error {
	m.updatedPhone = phone
	if m.user != nil {
		m.user.Phone = phone
	}
	return nil
}
func (m *mockAuthRepo) UpdateDisplayNameByUserID(ctx context.Context, userID, name string) error {
	m.updatedDisplayName = name
	if m.user != nil {
		m.user.Name = name
	}
	return nil
}
func (m *mockAuthRepo) SavePhoneChangeChallenge(ctx context.Context, challenge *model.PhoneChangeChallenge) error {
	cp := *challenge
	m.phoneChangeChallenge = &cp
	return nil
}
func (m *mockAuthRepo) GetPhoneChangeChallengeByID(ctx context.Context, challengeID string) (*model.PhoneChangeChallenge, error) {
	return m.phoneChangeChallenge, nil
}
func (m *mockAuthRepo) IncreasePhoneChangeChallengeAttempts(ctx context.Context, challengeID string) error {
	if m.phoneChangeChallenge != nil {
		m.phoneChangeChallenge.Attempts++
	}
	return nil
}
func (m *mockAuthRepo) MarkPhoneChangeChallengeVerified(ctx context.Context, challengeID string) error {
	now := time.Now()
	if m.phoneChangeChallenge != nil {
		m.phoneChangeChallenge.VerifiedAt = &now
	}
	return nil
}

type loginChallengeResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		ChallengeID string `json:"challenge_id"`
		MaskedPhone string `json:"masked_phone"`
	} `json:"data"`
}

type loginVerifyResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		AccessToken string `json:"access_token"`
	} `json:"data"`
}

type meResp struct {
	Code int `json:"code"`
	Data struct {
		User struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"user"`
		Role        string   `json:"role"`
		ScopeType   string   `json:"scope_type"`
		TenantID    string   `json:"tenant_id"`
		TenantName  string   `json:"tenant_name"`
		Permissions []string `json:"permissions"`
	} `json:"data"`
}

type mockPermHandler struct{ perms []string }

func (m mockPermHandler) Enforce(ctx context.Context, role, permission string) (bool, error) {
	return true, nil
}
func (m mockPermHandler) ListByRole(ctx context.Context, role string) ([]string, error) {
	return m.perms, nil
}

type refreshResp struct {
	Code int `json:"code"`
	Data struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	} `json:"data"`
}

type changePhoneChallengeResp struct {
	Code int `json:"code"`
	Data struct {
		ChallengeID    string `json:"challenge_id"`
		MaskedNewPhone string `json:"masked_new_phone"`
		ExpiresIn      int    `json:"expires_in"`
		ResendIn       int    `json:"resend_in"`
	} `json:"data"`
}

type changePhoneVerifyResp struct {
	Code int `json:"code"`
	Data struct {
		Status       string `json:"status"`
		ForceRelogin bool   `json:"force_relogin"`
	} `json:"data"`
}

type updateDisplayNameResp struct {
	Code int `json:"code"`
	Data struct {
		Status string `json:"status"`
		Name   string `json:"name"`
	} `json:"data"`
}

func TestAuthHandler_LoginChallenge(t *testing.T) {
	e := echo.New()

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	repo := &mockAuthRepo{
		user: &model.User{ID: "u1", Phone: "13800000003"},
		hash: string(hash),
	}
	h := &handler.AuthHandler{
		LoginChallengeUC: &usecase.AuthLoginChallengeUsecase{
			Repo:      repo,
			Challenge: repo,
			Sender:    smsrepo.MockSMSSender{},
			IDGen:     func() string { return "c1" },
			Now:       func() time.Time { return time.Unix(1, 0) },
			MockCode:  "123456",
		},
		MeUC: &usecase.AuthMeUsecase{},
		JWT:  middleware.JWTMiddleware{Secret: []byte("s")},
	}

	body := []byte(`{"account":"a","password":"pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.Login(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var resp loginChallengeResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if resp.Data.ChallengeID == "" || resp.Data.MaskedPhone == "" {
		t.Fatalf("expected challenge response, got: %+v", resp.Data)
	}
}

func TestAuthHandler_LoginVerify(t *testing.T) {
	e := echo.New()
	sum := sha256.Sum256([]byte("123456"))
	otpHash := hex.EncodeToString(sum[:])

	repo := &mockAuthRepo{
		user: &model.User{ID: "u1", Role: "tenant_member", ScopeType: "tenant", TenantID: "t1"},
		challenge: &model.LoginChallenge{
			ID:        "c1",
			UserID:    "u1",
			OTPHash:   otpHash,
			ExpiresAt: time.Now().Add(time.Minute),
		},
	}

	h := &handler.AuthHandler{
		LoginVerifyUC: &usecase.AuthLoginVerifyUsecase{
			Challenge: repo,
			TokenGen: func(ctx context.Context, user *model.User) (string, string, int, error) {
				return "tok", "rt", 1800, nil
			},
			Now: time.Now,
		},
		MeUC: &usecase.AuthMeUsecase{},
		JWT:  middleware.JWTMiddleware{Secret: []byte("s")},
	}

	body := []byte(`{"challenge_id":"c1","otp_code":"123456"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/login/verify", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.LoginVerify(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var resp loginVerifyResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if resp.Data.AccessToken == "" {
		t.Fatalf("expected access_token")
	}
}

func TestAuthHandler_Me(t *testing.T) {
	e := echo.New()
	jwtMW := middleware.JWTMiddleware{Secret: []byte("s")}

	h := &handler.AuthHandler{
		MeUC: &usecase.AuthMeUsecase{Perm: mockPermHandler{perms: []string{"p1"}}},
		JWT:  jwtMW,
	}

	claims := middleware.Claims{
		UserID:    "u1",
		Name:      "n",
		Role:      "tenant_member",
		ScopeType: "tenant",
		TenantID:  "t1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, _ := token.SignedString([]byte("s"))

	req := httptest.NewRequest(http.MethodPost, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+ss)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handlerFunc := jwtMW.MiddlewareFunc(h.Me)
	if err := handlerFunc(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var resp meResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if resp.Data.User.ID != "u1" || resp.Data.User.Name != "n" {
		t.Fatalf("unexpected user: %+v", resp.Data.User)
	}
	if resp.Data.TenantName != "" {
		t.Fatalf("expected empty tenant_name fallback, got %q", resp.Data.TenantName)
	}
}

func TestAuthHandler_Refresh(t *testing.T) {
	e := echo.New()

	refreshUC := &usecase.AuthRefreshUsecase{
		ParseRefresh: func(token string) (*model.User, error) {
			return &model.User{ID: "u1"}, nil
		},
		TokenGen: func(ctx context.Context, user *model.User) (string, string, int, error) {
			return "a", "r", 1200, nil
		},
	}

	h := &handler.AuthHandler{
		RefreshUC: refreshUC,
		MeUC:      &usecase.AuthMeUsecase{},
		JWT:       middleware.JWTMiddleware{Secret: []byte("s")},
	}

	body := []byte(`{"refresh_token":"rt"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.Refresh(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var resp refreshResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestAuthHandler_ChangePhoneChallenge(t *testing.T) {
	e := echo.New()
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.DefaultCost)
	repo := &mockAuthRepo{
		user: &model.User{ID: "u1", Role: "tenant_member", Phone: "13800000003"},
		hash: string(hash),
	}
	h := &handler.AuthHandler{
		ChangePhoneChallengeUC: &usecase.AuthChangePhoneChallengeUsecase{
			AuthRepo:      repo,
			PhoneRepo:     repo,
			ChallengeRepo: repo,
			Sender:        smsrepo.MockSMSSender{},
			IDGen:         func() string { return "00000000-0000-0000-0000-000000000901" },
			Now:           time.Now,
			MockCode:      "123456",
		},
	}
	body := []byte(`{"new_phone":"13800000004","old_password":"pass123"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/change-phone/challenge", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxUserID, "u1")
	c.Set(middleware.CtxRole, "tenant_member")
	c.Set(middleware.CtxRequestID, "rid-1")

	if err := h.ChangePhoneChallenge(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	var resp changePhoneChallengeResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp.Code != 0 || resp.Data.ChallengeID == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestAuthHandler_ChangePhoneVerify(t *testing.T) {
	e := echo.New()
	repo := &mockAuthRepo{
		user: &model.User{ID: "u1", Role: "tenant_member", Phone: "13800000003"},
		phoneChangeChallenge: &model.PhoneChangeChallenge{
			ID:        "00000000-0000-0000-0000-000000000902",
			UserID:    "u1",
			NewPhone:  "13800000004",
			OTPHash:   "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92",
			ExpiresAt: time.Now().Add(time.Minute),
		},
	}
	h := &handler.AuthHandler{
		ChangePhoneVerifyUC: &usecase.AuthChangePhoneVerifyUsecase{
			AuthRepo:      repo,
			PhoneRepo:     repo,
			ChallengeRepo: repo,
			Now:           time.Now,
		},
	}
	body := []byte(`{"challenge_id":"00000000-0000-0000-0000-000000000902","otp_code":"123456"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/change-phone/verify", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxUserID, "u1")
	c.Set(middleware.CtxRole, "tenant_member")
	c.Set(middleware.CtxRequestID, "rid-2")

	if err := h.ChangePhoneVerify(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	var resp changePhoneVerifyResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp.Code != 0 || !resp.Data.ForceRelogin {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestAuthHandler_UpdateDisplayName(t *testing.T) {
	e := echo.New()
	repo := &mockAuthRepo{
		user: &model.User{ID: "u1", Name: "old"},
	}
	h := &handler.AuthHandler{
		UpdateDisplayNameUC: &usecase.AuthUpdateDisplayNameUsecase{Repo: repo},
	}

	body := []byte(`{"name":"诸葛亮"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/profile/update-name", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxUserID, "u1")
	c.Set(middleware.CtxUserName, "old")
	c.Set(middleware.CtxRequestID, "rid-3")

	if err := h.UpdateDisplayName(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var resp updateDisplayNameResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp.Code != 0 || resp.Data.Status != "ok" || resp.Data.Name != "诸葛亮" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if repo.updatedDisplayName != "诸葛亮" {
		t.Fatalf("expected repo updated name, got %q", repo.updatedDisplayName)
	}
}
