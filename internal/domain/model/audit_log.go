package model

import "time"

type AuditLog struct {
	ID                  string
	RequestID           string
	OperatorUserID      string
	OperatorRole        string
	OperatorTenantID    string
	OperatorUsername    string
	OperatorDisplayName string
	TargetType          string
	TargetID            string
	TargetName          string
	Action              string
	Module              string
	Result              string
	ErrorCode           string
	BeforeJSON          map[string]any
	AfterJSON           map[string]any
	ChangedFields       []string
	IP                  string
	UserAgent           string
	CreatedAt           time.Time
}
