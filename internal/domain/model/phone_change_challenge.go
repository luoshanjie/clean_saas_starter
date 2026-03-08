package model

import "time"

type PhoneChangeChallenge struct {
	ID          string
	UserID      string
	NewPhone    string
	OTPHash     string
	ExpiresAt   time.Time
	Attempts    int
	ResendCount int
	LastSentAt  time.Time
	VerifiedAt  *time.Time
	CreatedAt   time.Time
}
