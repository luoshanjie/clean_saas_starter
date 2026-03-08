package model

import "time"

type LoginChallenge struct {
	ID         string
	UserID     string
	OTPHash    string
	ExpiresAt  time.Time
	Attempts   int
	VerifiedAt *time.Time
	CreatedAt  time.Time
}
