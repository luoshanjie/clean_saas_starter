package model

import "time"

type User struct {
	ID                 string
	TenantID           string
	TenantName         string
	Name               string
	Phone              string
	Role               string
	ScopeType          string
	TokenVersion       int
	MustChangePassword bool
	PasswordUpdatedAt  *time.Time
}
