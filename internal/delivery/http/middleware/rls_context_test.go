package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"service/internal/delivery/http/middleware"
	"service/internal/delivery/http/resp"
	"service/internal/domain/model"
)

type mockAuthRepo struct {
	v    int
	user *model.User
}

func (m mockAuthRepo) GetUserByAccount(ctx context.Context, account string) (*model.User, string, error) {
	return nil, "", nil
}
func (m mockAuthRepo) GetTokenVersionByUserID(ctx context.Context, userID string) (int, error) {
	return m.v, nil
}
func (m mockAuthRepo) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	if m.user != nil {
		return m.user, nil
	}
	return &model.User{ID: userID}, nil
}
func (m mockAuthRepo) GetPasswordHashByUserID(ctx context.Context, userID string) (string, error) {
	return "", nil
}
func (m mockAuthRepo) UpdatePasswordByUserID(ctx context.Context, userID, passwordHash string, mustChange bool) error {
	return nil
}

type envRLS struct {
	Code int `json:"code"`
}

func TestAuthContextMiddleware_MissingScope(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := middleware.AuthContextMiddleware(mockAuthRepo{v: 1})
	_ = mw(func(c echo.Context) error { return c.NoContent(http.StatusOK) })(c)

	var r envRLS
	_ = json.Unmarshal(rec.Body.Bytes(), &r)
	if r.Code != resp.CodeUnauthorized {
		t.Fatalf("expected unauthorized, got %d", r.Code)
	}
}

func TestAuthContextMiddleware_TokenVersionMismatch(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("scope_type", "tenant")
	c.Set("user_id", "u1")
	c.Set("token_version", 1)

	mw := middleware.AuthContextMiddleware(mockAuthRepo{v: 2})
	_ = mw(func(c echo.Context) error { return c.NoContent(http.StatusOK) })(c)

	var r envRLS
	_ = json.Unmarshal(rec.Body.Bytes(), &r)
	if r.Code != resp.CodeUnauthorized {
		t.Fatalf("expected unauthorized, got %d", r.Code)
	}
}

func TestAuthContextMiddleware_Pass(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("scope_type", "tenant")
	c.Set("user_id", "u1")
	c.Set("token_version", 1)

	mw := middleware.AuthContextMiddleware(mockAuthRepo{v: 1})
	called := false
	_ = mw(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})(c)

	if !called {
		t.Fatalf("handler should be called")
	}
}

func TestAuthContextMiddleware_MustChangePasswordBlocked(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/tenant/list", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("scope_type", "tenant")
	c.Set("user_id", "u1")
	c.Set("token_version", 1)

	mw := middleware.AuthContextMiddleware(mockAuthRepo{
		v: 1,
		user: &model.User{
			ID:   "u1",
			Role: "tenant_member",
		},
	})
	_ = mw(func(c echo.Context) error { return c.NoContent(http.StatusOK) })(c)

	var r envRLS
	_ = json.Unmarshal(rec.Body.Bytes(), &r)
	if r.Code != resp.CodeForbidden {
		t.Fatalf("expected forbidden, got %d", r.Code)
	}
}

func TestAuthContextMiddleware_MustChangePasswordAllowChangePassword(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("scope_type", "tenant")
	c.Set("user_id", "u1")
	c.Set("token_version", 1)

	mw := middleware.AuthContextMiddleware(mockAuthRepo{
		v: 1,
		user: &model.User{
			ID:   "u1",
			Role: "tenant_member",
		},
	})
	called := false
	_ = mw(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})(c)
	if !called {
		t.Fatalf("handler should be called for change-password")
	}
}
