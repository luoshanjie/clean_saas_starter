package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

type logEntry struct {
	level string
	msg   string
	args  map[string]any
}

type spyLogger struct {
	entries []logEntry
}

func (s *spyLogger) Debug(msg string, args ...any) {
	s.entries = append(s.entries, makeEntry("debug", msg, args...))
}
func (s *spyLogger) Info(msg string, args ...any) {
	s.entries = append(s.entries, makeEntry("info", msg, args...))
}
func (s *spyLogger) Warn(msg string, args ...any) {
	s.entries = append(s.entries, makeEntry("warn", msg, args...))
}
func (s *spyLogger) Error(msg string, args ...any) {
	s.entries = append(s.entries, makeEntry("error", msg, args...))
}
func (s *spyLogger) Errorf(format string, args ...any) {
	s.entries = append(s.entries, logEntry{level: "error", msg: format})
}

func makeEntry(level, msg string, args ...any) logEntry {
	m := map[string]any{}
	for i := 0; i+1 < len(args); i += 2 {
		k, ok := args[i].(string)
		if !ok {
			continue
		}
		m[k] = args[i+1]
	}
	return logEntry{level: level, msg: msg, args: m}
}

func TestRequestLogger_RedactsRequestAndResponseBody(t *testing.T) {
	e := echo.New()
	l := &spyLogger{}
	mw := RequestLogger(l)

	handler := mw(func(c echo.Context) error {
		var body map[string]any
		if err := c.Bind(&body); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]any{
			"name":         "role_platform_op",
			"display_name": body["display_name"],
			"access_token": "token-xyz",
			"ok":           true,
		})
	})

	reqBody := []byte(`{"account":"coo","password":"123456789","display_name":"诸葛亮","phone":"13512345678","x":"ok"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/profile/update-name", bytes.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(CtxRequestID, "rid-1")
	c.Set(CtxUserID, "u1")
	c.Set(CtxTenantID, "t1")
	c.Set(CtxRole, "tenant_admin")

	if err := handler(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if len(l.entries) != 4 {
		t.Fatalf("expected 4 log entries, got %d", len(l.entries))
	}
	inEntry := l.entries[0]
	outEntry := l.entries[1]
	summaryEntry := l.entries[2]
	eventEntry := l.entries[3]
	if inEntry.msg != "request_in" || outEntry.msg != "request_out" {
		t.Fatalf("unexpected log messages: in=%s out=%s", inEntry.msg, outEntry.msg)
	}
	if summaryEntry.msg != "request_summary" {
		t.Fatalf("unexpected summary log message: %s", summaryEntry.msg)
	}
	if eventEntry.msg != "biz_event" {
		t.Fatalf("unexpected biz event log message: %s", eventEntry.msg)
	}
	if inEntry.level != "info" || outEntry.level != "info" {
		t.Fatalf("expected info/info level, got %s/%s", inEntry.level, outEntry.level)
	}
	reqLogged, _ := inEntry.args["request_body"].(string)
	respLogged, _ := outEntry.args["response_body"].(string)
	if reqLogged == "" || respLogged == "" {
		t.Fatalf("expected request/response body in logs: in=%+v out=%+v", inEntry.args, outEntry.args)
	}
	if containsAny(reqLogged, "123456789", "13512345678", "诸葛亮", "coo") {
		t.Fatalf("request body not redacted: %s", reqLogged)
	}
	if containsAny(respLogged, "token-xyz", "诸葛亮") {
		t.Fatalf("response body not redacted: %s", respLogged)
	}
	if !containsAny(respLogged, "role_platform_op") {
		t.Fatalf("expected non-sensitive name field to remain visible: %s", respLogged)
	}
	if summaryEntry.args["error_type"] != "none" {
		t.Fatalf("expected summary error_type none, got %v", summaryEntry.args["error_type"])
	}
	if eventEntry.args["event_status"] != "succeeded" {
		t.Fatalf("expected event_status succeeded, got %v", eventEntry.args["event_status"])
	}
}

func TestRequestLogger_UsesWarnFor4xx(t *testing.T) {
	e := echo.New()
	l := &spyLogger{}
	mw := RequestLogger(l)

	handler := mw(func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/not-found", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if len(l.entries) != 4 {
		t.Fatalf("expected 4 log entry, got %d", len(l.entries))
	}
	inEntry := l.entries[0]
	outEntry := l.entries[1]
	summaryEntry := l.entries[2]
	eventEntry := l.entries[3]
	if inEntry.msg != "request_in" || outEntry.msg != "request_out" {
		t.Fatalf("unexpected log messages: in=%s out=%s", inEntry.msg, outEntry.msg)
	}
	if inEntry.level != "info" {
		t.Fatalf("expected in info level, got %s", inEntry.level)
	}
	if outEntry.level != "warn" {
		t.Fatalf("expected out warn level, got %s", outEntry.level)
	}
	status, _ := outEntry.args["status"].(int)
	if status != http.StatusNotFound {
		t.Fatalf("expected status 404, got %v", outEntry.args["status"])
	}
	if errStr, _ := outEntry.args["error"].(string); errStr == "" {
		t.Fatalf("expected non-empty error field")
	}
	if summaryEntry.msg != "request_summary" || summaryEntry.level != "warn" {
		t.Fatalf("expected warn request_summary, got %s/%s", summaryEntry.level, summaryEntry.msg)
	}
	if eventEntry.msg != "biz_event" || eventEntry.level != "warn" {
		t.Fatalf("expected warn biz_event, got %s/%s", eventEntry.level, eventEntry.msg)
	}
}

func TestRequestLogger_WarnsOnBusinessCodeInEnvelope(t *testing.T) {
	e := echo.New()
	l := &spyLogger{}
	mw := RequestLogger(l)

	handler := mw(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{
			"code":       42201,
			"message":    "validation_error",
			"data":       nil,
			"request_id": "rid-x",
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/tenant/update", bytes.NewReader([]byte(`{"tenant_id":"t1"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(CtxRequestID, "rid-x")

	if err := handler(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if len(l.entries) != 4 {
		t.Fatalf("expected 4 log entries, got %d", len(l.entries))
	}
	outEntry := l.entries[1]
	summaryEntry := l.entries[2]
	eventEntry := l.entries[3]
	if outEntry.level != "warn" {
		t.Fatalf("expected request_out warn, got %s", outEntry.level)
	}
	if summaryEntry.level != "warn" {
		t.Fatalf("expected request_summary warn, got %s", summaryEntry.level)
	}
	if outEntry.args["api_code"] != float64(42201) && outEntry.args["api_code"] != 42201 {
		t.Fatalf("expected api_code 42201, got %v", outEntry.args["api_code"])
	}
	if outEntry.args["error_type"] != "business" {
		t.Fatalf("expected business error_type, got %v", outEntry.args["error_type"])
	}
	if outEntry.args["error_code"] != "validation_error" {
		t.Fatalf("expected validation_error code, got %v", outEntry.args["error_code"])
	}
	if outEntry.args["module"] != "platform_tenant" {
		t.Fatalf("expected auto module platform_tenant, got %v", outEntry.args["module"])
	}
	if outEntry.args["action"] != "update" {
		t.Fatalf("expected auto action update, got %v", outEntry.args["action"])
	}
	if outEntry.args["target_id"] != "t1" {
		t.Fatalf("expected target_id t1, got %v", outEntry.args["target_id"])
	}
	if eventEntry.args["event_name"] != "platform_tenant.update.failed" {
		t.Fatalf("expected event_name platform_tenant.update.failed, got %v", eventEntry.args["event_name"])
	}
}

func containsAny(s string, values ...string) bool {
	for _, v := range values {
		if v != "" && bytes.Contains([]byte(s), []byte(v)) {
			return true
		}
	}
	return false
}

func TestSanitizeJSON_Invalid(t *testing.T) {
	out := sanitizeJSON([]byte(`{`))
	if out == "" {
		t.Fatal(errors.New("expected placeholder for invalid json"))
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(`{"x":"ok"}`), &obj); err != nil {
		t.Fatalf("precondition failed: %v", err)
	}
}

func TestSanitizeJSON_KeywordNotRedacted(t *testing.T) {
	out := sanitizeJSON([]byte(`{"keyword":"西游记","access_key":"abc123"}`))
	if !containsAny(out, "西游记") {
		t.Fatalf("expected keyword to remain visible, got: %s", out)
	}
	if containsAny(out, "abc123") {
		t.Fatalf("expected access_key to be redacted, got: %s", out)
	}
}
