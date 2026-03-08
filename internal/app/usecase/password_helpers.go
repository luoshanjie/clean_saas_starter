package usecase

import (
	"crypto/rand"
)

const tempPasswordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

func generateTemporaryPassword(length int) string {
	if length <= 0 {
		return ""
	}
	buf := make([]byte, length)
	randBuf := make([]byte, length)
	if _, err := rand.Read(randBuf); err != nil {
		return ""
	}
	for i := 0; i < length; i++ {
		buf[i] = tempPasswordAlphabet[int(randBuf[i])%len(tempPasswordAlphabet)]
	}
	return string(buf)
}
