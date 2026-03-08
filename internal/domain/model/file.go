package model

import "time"

type File struct {
	ID        string
	TenantID  string
	Bucket    string
	ObjectKey string
	Size      int64
	Mime      string
	OwnerType string
	OwnerID   string
	CreatedAt time.Time
}
