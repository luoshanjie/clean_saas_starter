package usecase

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var mainlandPhonePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

func hashOTP(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func normalizeMainlandPhone(phone string) (string, bool) {
	p := strings.TrimSpace(phone)
	p = strings.TrimPrefix(p, "+86")
	p = strings.TrimPrefix(p, "86")
	if !mainlandPhonePattern.MatchString(p) {
		return "", false
	}
	return p, true
}

func maskPhone(phone string) string {
	if len(phone) != 11 {
		return ""
	}
	return phone[:3] + "****" + phone[7:]
}
