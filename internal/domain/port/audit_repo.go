package port

import (
	"context"

	"service/internal/domain/model"
)

type AuditRepo interface {
	Create(ctx context.Context, log *model.AuditLog) error
	ListPage(ctx context.Context, filter AuditFilter) ([]*model.AuditLog, int, error)
	GetByID(ctx context.Context, id string) (*model.AuditLog, error)
}

type AuditFilter struct {
	Keyword        string
	Module         string
	Action         string
	Result         string
	OperatorUserID string
	TargetType     string
	TargetID       string
	RequestID      string
	TenantID       string
	DateFrom       string
	DateTo         string
	NeedTotal      bool
	Page           int
	PageSize       int
}
