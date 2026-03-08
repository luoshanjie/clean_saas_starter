package usecase

import (
	"context"
	"testing"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
	"service/internal/domain/port"
)

type auditRepoMock struct {
	created     *model.AuditLog
	items       []*model.AuditLog
	item        *model.AuditLog
	lastFilter  port.AuditFilter
	listPageHit bool
}

func (m *auditRepoMock) Create(ctx context.Context, log *model.AuditLog) error {
	m.created = log
	return nil
}
func (m *auditRepoMock) ListPage(ctx context.Context, filter port.AuditFilter) ([]*model.AuditLog, int, error) {
	m.lastFilter = filter
	m.listPageHit = true
	return m.items, len(m.items), nil
}
func (m *auditRepoMock) GetByID(ctx context.Context, id string) (*model.AuditLog, error) {
	if m.item == nil {
		return nil, domainErr.ErrNotFound
	}
	return m.item, nil
}

func TestAuditWriteUsecase_LogSafe(t *testing.T) {
	repo := &auditRepoMock{}
	u := &AuditWriteUsecase{Repo: repo, IDGen: func() string { return "a1" }}
	u.LogSafe(context.Background(), AuditWriteInput{
		TargetType: "tenant",
		Action:     "update",
		Module:     "platform_tenant",
		AfterJSON: map[string]any{
			"admin_phone": "13800138000",
		},
	})
	if repo.created == nil {
		t.Fatalf("expected audit created")
	}
	if repo.created.AfterJSON["admin_phone"] == "13800138000" {
		t.Fatalf("expected masked phone")
	}
}

func TestListAuditLogsUsecase_TenantMemberScope(t *testing.T) {
	repo := &auditRepoMock{items: []*model.AuditLog{{ID: "a1"}}}
	u := &ListAuditLogsUsecase{Repo: repo}
	out, err := u.Execute(context.Background(), ListAuditLogsInput{Role: "tenant_member", UserID: "u1", TenantID: "t1"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.Total != 1 {
		t.Fatalf("unexpected total: %d", out.Total)
	}
	if !repo.listPageHit || repo.lastFilter.TenantID != "t1" || repo.lastFilter.OperatorUserID != "u1" {
		t.Fatalf("unexpected tenant member filter: %+v", repo.lastFilter)
	}
}

func TestListAuditLogsUsecase_TenantAdminForcesTenantFilter(t *testing.T) {
	repo := &auditRepoMock{items: []*model.AuditLog{{ID: "a1"}}}
	u := &ListAuditLogsUsecase{Repo: repo}
	_, err := u.Execute(context.Background(), ListAuditLogsInput{
		Role:           "tenant_admin",
		UserID:         "u-admin",
		TenantID:       "t-token",
		FilterTenantID: "t-other",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.lastFilter.TenantID != "t-token" {
		t.Fatalf("expected tenant filter forced to token tenant, got %q", repo.lastFilter.TenantID)
	}
	if repo.lastFilter.OperatorUserID != "" {
		t.Fatalf("expected no operator restriction for tenant admin, got %q", repo.lastFilter.OperatorUserID)
	}
}

func TestGetAuditDetailUsecase_Forbidden(t *testing.T) {
	u := &GetAuditDetailUsecase{Repo: &auditRepoMock{item: &model.AuditLog{ID: "a1", OperatorTenantID: "t2", OperatorUserID: "u2"}}}
	_, err := u.Execute(context.Background(), GetAuditDetailInput{Role: "tenant_member", UserID: "u1", TenantID: "t1", AuditID: "a1"})
	if err != domainErr.ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestGetAuditDetailUsecase_TenantAdminCannotViewPlatformLog(t *testing.T) {
	u := &GetAuditDetailUsecase{Repo: &auditRepoMock{item: &model.AuditLog{ID: "a1", OperatorTenantID: ""}}}
	_, err := u.Execute(context.Background(), GetAuditDetailInput{Role: "tenant_admin", TenantID: "t1", AuditID: "a1"})
	if err != domainErr.ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}
