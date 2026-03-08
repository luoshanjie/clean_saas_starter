package usecase

import "strings"

func sanitizeAuditMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		lk := strings.ToLower(strings.TrimSpace(k))
		if isSensitiveField(lk) {
			continue
		}
		switch vv := v.(type) {
		case string:
			if strings.Contains(lk, "phone") {
				out[k] = maskPhoneSimple(vv)
			} else if strings.Contains(lk, "email") {
				out[k] = maskEmailSimple(vv)
			} else {
				out[k] = vv
			}
		case map[string]any:
			out[k] = sanitizeAuditMap(vv)
		default:
			out[k] = v
		}
	}
	return out
}

func isSensitiveField(key string) bool {
	s := []string{"password", "temporary_password", "access_token", "refresh_token", "otp_code", "otp", "token"}
	for _, item := range s {
		if strings.Contains(key, item) {
			return true
		}
	}
	return false
}

func maskPhoneSimple(v string) string {
	if len(v) < 7 {
		return "***"
	}
	return v[:3] + "****" + v[len(v)-4:]
}

func maskEmailSimple(v string) string {
	at := strings.Index(v, "@")
	if at <= 1 {
		return "***"
	}
	return v[:1] + "***" + v[at:]
}
