package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"service/internal/delivery/http/resp"
)

type rateLimitResp struct {
	Code      int    `json:"code"`
	RequestID string `json:"request_id"`
}

func TestRateLimitMiddleware_BlocksAfterLimit(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/limit", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	mw := RateLimitMiddleware(1, time.Minute)
	handler := mw(func(c echo.Context) error {
		return c.JSON(http.StatusOK, resp.OKWithRequestID(GetRequestID(c), map[string]string{"ok": "1"}))
	})

	// first request allowed
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req, rec1)
	c1.Set(CtxRequestID, "rid-1")
	if err := handler(c1); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	// second request blocked
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req, rec2)
	c2.Set(CtxRequestID, "rid-2")
	if err := handler(c2); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var out rateLimitResp
	if err := json.Unmarshal(rec2.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out.Code != resp.CodeRateLimited {
		t.Fatalf("expected code %d, got %d", resp.CodeRateLimited, out.Code)
	}
	if out.RequestID != "rid-2" {
		t.Fatalf("expected request_id rid-2, got %q", out.RequestID)
	}
}
