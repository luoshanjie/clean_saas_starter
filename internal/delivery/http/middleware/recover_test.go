package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"service/internal/delivery/http/resp"
	"service/pkg/logger"
)

type recoverResp struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func TestRecoverMiddleware_ReturnsUnifiedError(t *testing.T) {
	e := echo.New()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	c := e.NewContext(req, rec)
	c.Set(CtxRequestID, "rid-1")

	mw := RecoverMiddleware(logger.NewNopLogger())
	handler := mw(func(c echo.Context) error {
		panic("boom")
	})

	_ = handler(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var out recoverResp
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out.Code != resp.CodeServerError {
		t.Fatalf("expected code %d, got %d", resp.CodeServerError, out.Code)
	}
	if out.RequestID != "rid-1" {
		t.Fatalf("expected request_id rid-1, got %q", out.RequestID)
	}
}
