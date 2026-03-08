package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
	"service/internal/domain/port"
)

type tenantMockPerm struct {
	allow bool
}

func (m tenantMockPerm) Enforce(ctx context.Context, role, permission string) (bool, error) {
	return m.allow, nil
}
func (m tenantMockPerm) ListByRole(ctx context.Context, role string) ([]string, error) {
	return nil, nil
}

type tenantMockRepo struct {
	createIn               *model.TenantCreateInput
	createOut              *model.TenantCreateOutput
	createErr              error
	listItems              []*model.TenantListItem
	listTotal              int
	listErr                error
	updateOK               bool
	updateErr              error
	toggleOK               bool
	toggleErr              error
	getTenant              *model.Tenant
	getErr                 error
	hasAdmin               bool
	hasAdminErr            error
	displayNameExists      bool
	adminAccountExists     bool
	adminPhoneExists       bool
	adminUser              *model.User
	adminAccount           string
	adminLookupErr         error
	updateIdentityUserID   string
	updateIdentityUsername string
	updateIdentityName     string
	updateIdentityPhone    string
	updateIdentityErr      error
	resetPasswordUserID    string
	resetPasswordHash      string
}

func (m *tenantMockRepo) CreateWithAdmin(ctx context.Context, in *model.TenantCreateInput) (*model.TenantCreateOutput, error) {
	m.createIn = in
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.createOut != nil {
		return m.createOut, nil
	}
	return &model.TenantCreateOutput{
		TenantID:           in.TenantID,
		TenantAdminUserID:  in.TenantAdminUserID,
		TenantAdminAccount: in.TenantAdminAccount,
		TenantAdminName:    in.TenantAdminName,
		Status:             in.Status,
		CreatedAt:          in.CreatedAt,
	}, nil
}
func (m *tenantMockRepo) ListPage(ctx context.Context, filter port.TenantFilter) ([]*model.TenantListItem, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	return m.listItems, m.listTotal, nil
}
func (m *tenantMockRepo) GetByID(ctx context.Context, tenantID string) (*model.Tenant, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.getTenant == nil {
		return nil, errors.New("missing tenant")
	}
	return m.getTenant, nil
}
func (m *tenantMockRepo) Update(ctx context.Context, tenant *model.Tenant) (bool, error) {
	if m.updateErr != nil {
		return false, m.updateErr
	}
	return m.updateOK, nil
}
func (m *tenantMockRepo) ToggleStatus(ctx context.Context, tenantID, status string) (bool, error) {
	if m.toggleErr != nil {
		return false, m.toggleErr
	}
	return m.toggleOK, nil
}
func (m *tenantMockRepo) HasTenantAdmin(ctx context.Context, tenantID string) (bool, error) {
	if m.hasAdminErr != nil {
		return false, m.hasAdminErr
	}
	return m.hasAdmin, nil
}
func (m *tenantMockRepo) DisplayNameExists(ctx context.Context, displayName string) (bool, error) {
	return m.displayNameExists, nil
}
func (m *tenantMockRepo) TenantAdminAccountExists(ctx context.Context, adminAccount string) (bool, error) {
	return m.adminAccountExists, nil
}
func (m *tenantMockRepo) TenantAdminPhoneExists(ctx context.Context, adminPhone string) (bool, error) {
	return m.adminPhoneExists, nil
}
func (m *tenantMockRepo) GetTenantAdminByTenantID(ctx context.Context, tenantID string) (*model.User, string, error) {
	if m.adminLookupErr != nil {
		return nil, "", m.adminLookupErr
	}
	if m.adminUser == nil {
		return nil, "", errors.New("missing admin")
	}
	return m.adminUser, m.adminAccount, nil
}
func (m *tenantMockRepo) GetTenantAdminByUserID(ctx context.Context, adminUserID string) (*model.User, string, error) {
	if m.adminLookupErr != nil {
		return nil, "", m.adminLookupErr
	}
	if m.adminUser == nil {
		return nil, "", errors.New("missing admin")
	}
	return m.adminUser, m.adminAccount, nil
}
func (m *tenantMockRepo) UpdateTenantAdminIdentity(ctx context.Context, adminUserID, adminUsername, adminName, adminPhone string) error {
	if m.updateIdentityErr != nil {
		return m.updateIdentityErr
	}
	m.updateIdentityUserID = adminUserID
	m.updateIdentityUsername = adminUsername
	m.updateIdentityName = adminName
	m.updateIdentityPhone = adminPhone
	m.adminAccount = adminUsername
	if m.adminUser != nil {
		m.adminUser.Name = adminName
		m.adminUser.Phone = adminPhone
	}
	for _, item := range m.listItems {
		if item == nil {
			continue
		}
		item.TenantAdminAccount = adminUsername
		item.TenantAdminName = adminName
		item.TenantAdminPhone = adminPhone
	}
	return nil
}
func (m *tenantMockRepo) ResetTenantAdminPassword(ctx context.Context, adminUserID, passwordHash string) error {
	m.resetPasswordUserID = adminUserID
	m.resetPasswordHash = passwordHash
	return nil
}

func TestCreatePlatformTenant_Success(t *testing.T) {
	now := time.Unix(100, 0)
	repo := &tenantMockRepo{}
	u := &CreatePlatformTenantUsecase{
		Repo:  repo,
		Perm:  tenantMockPerm{allow: true},
		IDGen: func() string { return "id-1" },
		Now:   func() time.Time { return now },
	}
	out, err := u.Execute(context.Background(), CreatePlatformTenantInput{
		DisplayName:  "Tenant A",
		AdminAccount: "tenant_admin_a",
		AdminName:    "Tenant Admin",
		AdminPhone:   "13800138000",
		Status:       "active",
		Role:         "platform_op",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil || out.TenantID == "" || out.TenantAdminUserID == "" {
		t.Fatalf("unexpected output: %+v", out)
	}
	if repo.createIn == nil || repo.createIn.TenantAdminPassword == "" {
		t.Fatalf("expected hashed password in create input")
	}
	if repo.createIn.TenantAdminPhone != "13800138000" {
		t.Fatalf("expected normalized phone, got %q", repo.createIn.TenantAdminPhone)
	}
}

func TestCreatePlatformTenant_Conflict(t *testing.T) {
	repo := &tenantMockRepo{createErr: domainErr.ErrTenantAdminAccountExists}
	u := &CreatePlatformTenantUsecase{Repo: repo, IDGen: func() string { return "id-1" }}
	_, err := u.Execute(context.Background(), CreatePlatformTenantInput{
		DisplayName:  "Tenant A",
		AdminAccount: "dup",
		AdminName:    "Tenant Admin",
		AdminPhone:   "13800138000",
	})
	if err != domainErr.ErrTenantAdminAccountExists {
		t.Fatalf("expected tenant_admin_account_exists, got %v", err)
	}
}

func TestListPlatformTenants_InvalidStatus(t *testing.T) {
	u := &ListPlatformTenantsUsecase{Repo: &tenantMockRepo{}}
	_, err := u.Execute(context.Background(), ListPlatformTenantsInput{Status: "bad"})
	if err != domainErr.ErrInvalidTenantStatus {
		t.Fatalf("expected invalid_tenant_status, got %v", err)
	}
}

func TestUpdatePlatformTenant_NotFound(t *testing.T) {
	u := &UpdatePlatformTenantUsecase{Repo: &tenantMockRepo{updateOK: false}}
	err := u.Execute(context.Background(), UpdatePlatformTenantInput{TenantID: "t1", DisplayName: "Tenant A"})
	if err != domainErr.ErrTenantNotFound {
		t.Fatalf("expected tenant_not_found, got %v", err)
	}
}

func TestTogglePlatformTenantStatus_InvalidStatus(t *testing.T) {
	u := &TogglePlatformTenantStatusUsecase{Repo: &tenantMockRepo{}}
	err := u.Execute(context.Background(), TogglePlatformTenantStatusInput{TenantID: "t1", Status: "bad"})
	if err != domainErr.ErrInvalidTenantStatus {
		t.Fatalf("expected invalid_tenant_status, got %v", err)
	}
}

func TestResetPlatformTenantAdminAuth_Success(t *testing.T) {
	u := &ResetPlatformTenantAdminAuthUsecase{Repo: &tenantMockRepo{
		getTenant: &model.Tenant{ID: "t1"},
		hasAdmin:  true,
	}}
	err := u.Execute(context.Background(), ResetPlatformTenantAdminAuthInput{TenantID: "t1", Action: "resend_activate"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
