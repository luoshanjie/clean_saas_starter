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
)

type mockPerm struct {
	allow bool
}

func (m mockPerm) Enforce(ctx context.Context, role, permission string) (bool, error) {
	return m.allow, nil
}
func (m mockPerm) ListByRole(ctx context.Context, role string) ([]string, error) { return nil, nil }

type env struct {
	Code int `json:"code"`
}

func TestRequirePermission_Forbidden(t *testing.T) {
	e := echo.New()
	mw := middleware.RequirePermission(mockPerm{allow: false}, "tenant.audit.view")

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("role", "tenant_admin")

	h := mw(func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	_ = h(c)

	var r env
	_ = json.Unmarshal(rec.Body.Bytes(), &r)
	if r.Code != resp.CodeForbidden {
		t.Fatalf("expected forbidden, got %d", r.Code)
	}
}

func TestRequirePermission_Allows(t *testing.T) {
	e := echo.New()
	mw := middleware.RequirePermission(mockPerm{allow: true}, "tenant.audit.view")

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("role", "tenant_admin")

	called := false
	h := mw(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})
	_ = h(c)
	if !called {
		t.Fatalf("handler should be called")
	}
}
