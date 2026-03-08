package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	"service/internal/delivery/http/middleware"
)

type envelope struct {
	Code int `json:"code"`
}

func TestJWTMiddleware_MissingToken(t *testing.T) {
	e := echo.New()
	mw := middleware.JWTMiddleware{Secret: []byte("s")}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	h := mw.MiddlewareFunc(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})

	_ = h(c)
	if called {
		t.Fatalf("handler should not be called")
	}
	var resp envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp.Code != 40101 {
		t.Fatalf("expected code 40101, got %d", resp.Code)
	}
}

func TestJWTMiddleware_ValidToken(t *testing.T) {
	e := echo.New()
	mw := middleware.JWTMiddleware{Secret: []byte("s")}

	claims := middleware.Claims{
		UserID:       "u1",
		TokenVersion: 1,
		Role:         "tenant_member",
		ScopeType:    "tenant",
		TenantID:     "t1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, _ := token.SignedString([]byte("s"))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+ss)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	h := mw.MiddlewareFunc(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})

	_ = h(c)
	if !called {
		t.Fatalf("handler should be called")
	}
}

func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	e := echo.New()
	mw := middleware.JWTMiddleware{Secret: []byte("s")}

	claims := middleware.Claims{
		UserID: "u1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, _ := token.SignedString([]byte("s"))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+ss)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = mw.MiddlewareFunc(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(c)

	var resp envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp.Code != 40101 {
		t.Fatalf("expected code 40101, got %d", resp.Code)
	}
}

func TestJWTMiddleware_InvalidSignature(t *testing.T) {
	e := echo.New()
	mw := middleware.JWTMiddleware{Secret: []byte("s")}

	claims := middleware.Claims{
		UserID: "u1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, _ := token.SignedString([]byte("other"))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+ss)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = mw.MiddlewareFunc(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(c)

	var resp envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp.Code != 40101 {
		t.Fatalf("expected code 40101, got %d", resp.Code)
	}
}

func TestJWTMiddleware_RefreshTokenRejected(t *testing.T) {
	e := echo.New()
	mw := middleware.JWTMiddleware{Secret: []byte("s")}

	claims := middleware.Claims{
		UserID:       "u1",
		TokenVersion: 1,
		TokenType:    "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, _ := token.SignedString([]byte("s"))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+ss)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = mw.MiddlewareFunc(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(c)

	var resp envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp.Code != 40101 {
		t.Fatalf("expected code 40101, got %d", resp.Code)
	}
}
