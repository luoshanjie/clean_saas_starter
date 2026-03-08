package port

import (
	"context"

	"service/internal/domain/model"
)

type TenantRepo interface {
	CreateWithAdmin(ctx context.Context, in *model.TenantCreateInput) (*model.TenantCreateOutput, error)
	ListPage(ctx context.Context, filter TenantFilter) ([]*model.TenantListItem, int, error)
	GetByID(ctx context.Context, tenantID string) (*model.Tenant, error)
	Update(ctx context.Context, tenant *model.Tenant) (bool, error)
	ToggleStatus(ctx context.Context, tenantID, status string) (bool, error)
	HasTenantAdmin(ctx context.Context, tenantID string) (bool, error)
	DisplayNameExists(ctx context.Context, displayName string) (bool, error)
	TenantAdminAccountExists(ctx context.Context, account string) (bool, error)
	TenantAdminPhoneExists(ctx context.Context, phone string) (bool, error)
	GetTenantAdminByTenantID(ctx context.Context, tenantID string) (*model.User, string, error)
	GetTenantAdminByUserID(ctx context.Context, adminUserID string) (*model.User, string, error)
	UpdateTenantAdminIdentity(ctx context.Context, adminUserID, account, adminName, phone string) error
	ResetTenantAdminPassword(ctx context.Context, adminUserID, passwordHash string) error
}

type TenantFilter struct {
	Keyword   string
	Province  string
	City      string
	District  string
	Status    string
	NeedTotal bool
	Page      int
	PageSize  int
}
