package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"service/pkg/logger"
)

const maxLoggedBodyBytes = 4096

const (
	errorTypeNone   = "none"
	errorTypeBiz    = "business"
	errorTypeSystem = "system"
)

type envelopeLite struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// RequestLogger writes one log entry per request with useful context.
func RequestLogger(l logger.Logger) echo.MiddlewareFunc {
	if l == nil {
		l = logger.NewNopLogger()
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			resp := c.Response()
			reqBodyRaw := readAndRestoreBody(req)
			capture := &bodyCaptureWriter{ResponseWriter: resp.Writer, status: http.StatusOK}
			resp.Writer = capture
			path := c.Path()
			if path == "" {
				path = req.URL.Path
			}

			inArgs := []any{
				"method", req.Method,
				"path", path,
				"query", req.URL.RawQuery,
				"host", req.Host,
				"remote_ip", c.RealIP(),
				"bytes_in", req.ContentLength,
				"request_body", sanitizePayload(reqBodyRaw, req.Header.Get(echo.HeaderContentType)),
				"request_id", GetRequestID(c),
				"user_id", GetUserID(c),
				"tenant_id", GetTenantID(c),
				"role", GetRole(c),
			}
			l.Info("request_in", inArgs...)

			start := time.Now()
			err := next(c)
			errMsg := ""
			if err != nil {
				errMsg = err.Error()
			}
			if err != nil {
				c.Error(err)
				err = nil
			}
			latency := time.Since(start)
			apiCode, apiMessage, hasEnvelope := parseResponseEnvelope(capture.buf.Bytes(), resp.Header().Get(echo.HeaderContentType))
			module := strings.TrimSpace(GetLogModule(c))
			action := strings.TrimSpace(GetLogAction(c))
			targetID := strings.TrimSpace(GetLogTargetID(c))
			bizStep := strings.TrimSpace(GetLogBizStep(c))
			failPoint := strings.TrimSpace(GetLogFailPoint(c))
			if module == "" || action == "" {
				autoModule, autoAction := inferModuleAction(path)
				if module == "" {
					module = autoModule
				}
				if action == "" {
					action = autoAction
				}
			}
			if bizStep == "" {
				bizStep = action + ".handler"
			}
			errType := classifyErrorType(resp.Status, apiCode, hasEnvelope)
			errCode := buildErrorCode(resp.Status, apiCode, apiMessage, errMsg)
			if targetID == "" {
				targetID = firstNonEmpty(
					extractTargetID(reqBodyRaw, req.Header.Get(echo.HeaderContentType)),
					extractTargetID(capture.buf.Bytes(), resp.Header().Get(echo.HeaderContentType)),
				)
			}
			if failPoint == "" && errType != errorTypeNone {
				failPoint = deriveFailPoint(errCode)
			}
			outArgs := []any{
				"method", req.Method,
				"path", path,
				"query", req.URL.RawQuery,
				"status", resp.Status,
				"latency_ms", latency.Milliseconds(),
				"host", req.Host,
				"remote_ip", c.RealIP(),
				"bytes_out", resp.Size,
				"response_body", sanitizePayload(capture.buf.Bytes(), resp.Header().Get(echo.HeaderContentType)),
				"request_id", GetRequestID(c),
				"user_id", GetUserID(c),
				"tenant_id", GetTenantID(c),
				"role", GetRole(c),
				"module", module,
				"action", action,
				"target_id", targetID,
				"biz_step", bizStep,
				"fail_point", failPoint,
				"api_code", apiCode,
				"error_type", errType,
				"error_code", errCode,
			}
			if errMsg != "" {
				outArgs = append(outArgs, "error", errMsg)
			} else if c.Response().Status >= 400 {
				// Keep log shape stable even when echo handler has already consumed the error.
				outArgs = append(outArgs, "error", http.StatusText(c.Response().Status))
			} else if errCode != "" {
				outArgs = append(outArgs, "error", errCode)
			}
			switch classifyLogLevel(resp.Status, apiCode, hasEnvelope) {
			case "error":
				l.Error("request_out", outArgs...)
			case "warn":
				l.Warn("request_out", outArgs...)
			default:
				l.Info("request_out", outArgs...)
			}

			summaryArgs := []any{
				"method", req.Method,
				"path", path,
				"status", resp.Status,
				"latency_ms", latency.Milliseconds(),
				"request_id", GetRequestID(c),
				"user_id", GetUserID(c),
				"tenant_id", GetTenantID(c),
				"role", GetRole(c),
				"module", module,
				"action", action,
				"target_id", targetID,
				"biz_step", bizStep,
				"fail_point", failPoint,
				"api_code", apiCode,
				"error_type", errType,
				"error_code", errCode,
			}
			switch classifyLogLevel(resp.Status, apiCode, hasEnvelope) {
			case "error":
				l.Error("request_summary", summaryArgs...)
			case "warn":
				l.Warn("request_summary", summaryArgs...)
			default:
				l.Info("request_summary", summaryArgs...)
			}

			eventStatus := "succeeded"
			if errType != errorTypeNone {
				eventStatus = "failed"
			}
			bizEventArgs := []any{
				"event_name", buildBizEventName(module, action, eventStatus),
				"event_status", eventStatus,
				"request_id", GetRequestID(c),
				"user_id", GetUserID(c),
				"tenant_id", GetTenantID(c),
				"role", GetRole(c),
				"module", module,
				"action", action,
				"target_id", targetID,
				"biz_step", bizStep,
				"fail_point", failPoint,
				"api_code", apiCode,
				"error_type", errType,
				"error_code", errCode,
				"latency_ms", latency.Milliseconds(),
			}
			switch classifyLogLevel(resp.Status, apiCode, hasEnvelope) {
			case "error":
				l.Error("biz_event", bizEventArgs...)
			case "warn":
				l.Warn("biz_event", bizEventArgs...)
			default:
				l.Info("biz_event", bizEventArgs...)
			}

			return err
		}
	}
}

type bodyCaptureWriter struct {
	http.ResponseWriter
	buf    bytes.Buffer
	status int
}

func (w *bodyCaptureWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyCaptureWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	remaining := maxLoggedBodyBytes - w.buf.Len()
	if remaining > 0 {
		if len(p) > remaining {
			_, _ = w.buf.Write(p[:remaining])
		} else {
			_, _ = w.buf.Write(p)
		}
	}
	return w.ResponseWriter.Write(p)
}

func readAndRestoreBody(req *http.Request) []byte {
	if req == nil || req.Body == nil {
		return nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		req.Body = io.NopCloser(bytes.NewReader(nil))
		return nil
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

func sanitizePayload(raw []byte, contentType string) string {
	if len(raw) == 0 {
		return ""
	}
	if strings.Contains(strings.ToLower(contentType), "application/json") {
		return sanitizeJSON(raw)
	}
	// Current APIs are JSON; for non-JSON payloads only keep minimal diagnostic text.
	return "[non-json payload omitted]"
}

func sanitizeJSON(raw []byte) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return ""
	}
	limited := trimmed
	if len(limited) > maxLoggedBodyBytes {
		limited = limited[:maxLoggedBodyBytes]
	}
	var v any
	if err := json.Unmarshal(limited, &v); err != nil {
		return "[invalid-json payload omitted]"
	}
	safe := sanitizeJSONValue(v, "")
	b, err := json.Marshal(safe)
	if err != nil {
		return "[json-marshal-error]"
	}
	return string(b)
}

func sanitizeJSONValue(v any, key string) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[k] = sanitizeJSONValue(vv, k)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i := range val {
			out[i] = sanitizeJSONValue(val[i], key)
		}
		return out
	case string:
		if isSensitiveKey(key) {
			return "***"
		}
		return truncate(val, 256)
	default:
		if isSensitiveKey(key) {
			return "***"
		}
		return val
	}
}

func isSensitiveKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" {
		return false
	}
	// Keep search keyword visible for troubleshooting query issues.
	if k == "keyword" {
		return false
	}
	// Keep llm/vl api keys unmasked by explicit product decision in dev stage.
	if k == "llm_api_key" || k == "vl_api_key" {
		return false
	}
	if k == "account" || k == "display_name" || k == "real_name" || k == "full_name" {
		return true
	}
	if strings.Contains(k, "password") ||
		strings.Contains(k, "token") ||
		strings.Contains(k, "key") ||
		strings.Contains(k, "secret") ||
		strings.Contains(k, "otp") ||
		strings.Contains(k, "phone") ||
		strings.Contains(k, "mobile") ||
		k == "authorization" {
		return true
	}
	return false
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}

func parseResponseEnvelope(raw []byte, contentType string) (int, string, bool) {
	if len(raw) == 0 || !strings.Contains(strings.ToLower(contentType), "application/json") {
		return 0, "", false
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return 0, "", false
	}
	if len(trimmed) > maxLoggedBodyBytes {
		trimmed = trimmed[:maxLoggedBodyBytes]
	}
	var env envelopeLite
	if err := json.Unmarshal(trimmed, &env); err != nil {
		return 0, "", false
	}
	return env.Code, strings.TrimSpace(env.Message), true
}

func classifyLogLevel(httpStatus, apiCode int, hasEnvelope bool) string {
	if httpStatus >= 500 {
		return "error"
	}
	if httpStatus >= 400 {
		return "warn"
	}
	if hasEnvelope && apiCode != 0 {
		if apiCode >= 50000 {
			return "error"
		}
		return "warn"
	}
	return "info"
}

func classifyErrorType(httpStatus, apiCode int, hasEnvelope bool) string {
	if httpStatus >= 500 {
		return errorTypeSystem
	}
	if httpStatus >= 400 {
		return errorTypeBiz
	}
	if hasEnvelope && apiCode != 0 {
		if apiCode >= 50000 {
			return errorTypeSystem
		}
		return errorTypeBiz
	}
	return errorTypeNone
}

func buildErrorCode(httpStatus, apiCode int, apiMessage, handlerErr string) string {
	if strings.TrimSpace(handlerErr) != "" {
		return strings.TrimSpace(handlerErr)
	}
	if apiCode != 0 {
		if strings.TrimSpace(apiMessage) != "" {
			return strings.TrimSpace(apiMessage)
		}
		return "code_" + strconv.Itoa(apiCode)
	}
	if httpStatus >= 400 {
		return http.StatusText(httpStatus)
	}
	return ""
}

func inferModuleAction(reqPath string) (string, string) {
	p := strings.TrimSpace(reqPath)
	if p == "" {
		return "unknown", "unknown"
	}
	clean := path.Clean("/" + strings.TrimPrefix(p, "/"))
	if clean == "/" || clean == "." {
		return "unknown", "unknown"
	}
	parts := strings.Split(strings.TrimPrefix(clean, "/"), "/")
	if len(parts) >= 2 && parts[0] == "api" && parts[1] == "v1" {
		parts = parts[2:]
	}
	if len(parts) == 0 {
		return "root", "root"
	}
	if len(parts) == 1 {
		return sanitizeToken(parts[0]), sanitizeToken(parts[0])
	}
	module := sanitizeToken(parts[0])
	if len(parts) >= 2 {
		module = module + "_" + sanitizeToken(parts[1])
	}
	actionParts := make([]string, 0, len(parts)-2)
	for _, p := range parts[2:] {
		if strings.TrimSpace(p) == "" {
			continue
		}
		actionParts = append(actionParts, sanitizeToken(p))
	}
	if len(actionParts) == 0 {
		actionParts = append(actionParts, "root")
	}
	return module, strings.Join(actionParts, ".")
}

func sanitizeToken(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "unknown"
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-' || r == '.':
			b.WriteRune('_')
		default:
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "unknown"
	}
	return out
}

func extractTargetID(raw []byte, contentType string) string {
	if len(raw) == 0 || !strings.Contains(strings.ToLower(contentType), "application/json") {
		return ""
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return ""
	}
	var anyObj any
	if err := json.Unmarshal(trimmed, &anyObj); err != nil {
		return ""
	}
	keys := []string{
		"tenant_id", "user_id", "upload_id", "file_id", "id", "target_id",
	}
	return deepPickID(anyObj, keys)
}

func deepPickID(v any, keys []string) string {
	switch vv := v.(type) {
	case map[string]any:
		for _, k := range keys {
			if s, ok := vv[k].(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
		if data, ok := vv["data"]; ok {
			if got := deepPickID(data, keys); got != "" {
				return got
			}
		}
		for _, x := range vv {
			if got := deepPickID(x, keys); got != "" {
				return got
			}
		}
	case []any:
		for _, x := range vv {
			if got := deepPickID(x, keys); got != "" {
				return got
			}
		}
	}
	return ""
}

func deriveFailPoint(errCode string) string {
	code := strings.TrimSpace(errCode)
	if code == "" {
		return ""
	}
	code = strings.ToLower(code)
	code = strings.ReplaceAll(code, " ", "_")
	code = strings.ReplaceAll(code, ":", "_")
	return sanitizeToken(code)
}

func firstNonEmpty(items ...string) string {
	for _, it := range items {
		if strings.TrimSpace(it) != "" {
			return strings.TrimSpace(it)
		}
	}
	return ""
}

func buildBizEventName(module, action, status string) string {
	m := sanitizeToken(module)
	a := strings.ReplaceAll(sanitizeToken(action), "_", ".")
	s := sanitizeToken(status)
	if a == "unknown" {
		a = "unknown"
	}
	return m + "." + a + "." + s
}
