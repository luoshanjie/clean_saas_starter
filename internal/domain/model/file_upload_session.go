package model

import "time"

type FileUploadSession struct {
	ID          string
	TenantID    string
	UploadedBy  string
	ScopeType   string
	BizType     string
	FileName    string
	ContentType string
	SizeBytes   int64
	FileURL     string
	Status      string
	ExpiresAt   time.Time
	ConfirmedAt *time.Time
	MimeType    string
	DurationSec int
	DeletedAt   *time.Time
	LastError   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
