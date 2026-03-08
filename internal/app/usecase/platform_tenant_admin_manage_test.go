package usecase

import (
	"context"
	"testing"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
)

func TestChangePlatformTenantAdmin_Success(t *testing.T) {
	repo := &tenantMockRepo{
		adminUser:    &model.User{ID: "u1", Phone: "13800138000", Name: "旧管理员"},
		adminAccount: "old_admin",
	}
	u := &ChangePlatformTenantAdminUsecase{Repo: repo}
	out, err := u.Execute(context.Background(), ChangePlatformTenantAdminInput{
		TenantID:     "t1",
		AdminAccount: "new_admin",
		AdminName:    "新管理员",
		AdminPhone:   "13900139000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AdminUserID != "u1" || out.AdminAccount != "new_admin" {
		t.Fatalf("unexpected output: %+v", out)
	}
	if repo.updateIdentityName != "新管理员" {
		t.Fatalf("expected admin name to be updated, got %q", repo.updateIdentityName)
	}
}

func TestChangePlatformTenantAdmin_NameOnlyThenListShowsNewName(t *testing.T) {
	repo := &tenantMockRepo{
		adminUser:    &model.User{ID: "u1", Phone: "13800138000", Name: "旧管理员"},
		adminAccount: "km_admin",
		listItems: []*model.TenantListItem{
			{
				Tenant:             &model.Tenant{ID: "t1", DisplayName: "Tenant A"},
				TenantAdminUserID:  "u1",
				TenantAdminAccount: "km_admin",
				TenantAdminName:    "旧管理员",
				TenantAdminPhone:   "13800138000",
			},
		},
		listTotal: 1,
	}
	changeUC := &ChangePlatformTenantAdminUsecase{Repo: repo}
	if _, err := changeUC.Execute(context.Background(), ChangePlatformTenantAdminInput{
		TenantID:     "t1",
		AdminAccount: "km_admin",
		AdminName:    "新管理员",
		AdminPhone:   "13800138000",
	}); err != nil {
		t.Fatalf("unexpected change-admin error: %v", err)
	}
	listUC := &ListPlatformTenantsUsecase{Repo: repo}
	out, err := listUC.Execute(context.Background(), ListPlatformTenantsInput{NeedTotal: true, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].TenantAdminName != "新管理员" {
		t.Fatalf("expected updated tenant_admin_name, got %+v", out.Items)
	}
}

func TestChangePlatformTenantAdmin_AllIdentityFieldsThenListShowsNewName(t *testing.T) {
	repo := &tenantMockRepo{
		adminUser:    &model.User{ID: "u1", Phone: "13800138000", Name: "旧管理员"},
		adminAccount: "old_admin",
		listItems: []*model.TenantListItem{
			{
				Tenant:             &model.Tenant{ID: "t1", DisplayName: "Tenant A"},
				TenantAdminUserID:  "u1",
				TenantAdminAccount: "old_admin",
				TenantAdminName:    "旧管理员",
				TenantAdminPhone:   "13800138000",
			},
		},
		listTotal: 1,
	}
	changeUC := &ChangePlatformTenantAdminUsecase{Repo: repo}
	if _, err := changeUC.Execute(context.Background(), ChangePlatformTenantAdminInput{
		TenantID:     "t1",
		AdminAccount: "new_admin",
		AdminName:    "新管理员",
		AdminPhone:   "13900139000",
	}); err != nil {
		t.Fatalf("unexpected change-admin error: %v", err)
	}
	listUC := &ListPlatformTenantsUsecase{Repo: repo}
	out, err := listUC.Execute(context.Background(), ListPlatformTenantsInput{NeedTotal: true, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].TenantAdminName != "新管理员" {
		t.Fatalf("expected updated tenant_admin_name, got %+v", out.Items)
	}
}

func TestResetPlatformTenantAdminPassword_Success(t *testing.T) {
	repo := &tenantMockRepo{
		adminUser: &model.User{ID: "u1"},
	}
	u := &ResetPlatformTenantAdminPasswordUsecase{
		Repo: repo,
		PassGen: func() string {
			return "Temp12345"
		},
	}
	out, err := u.Execute(context.Background(), ResetPlatformTenantAdminPasswordInput{TenantID: "t1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.TemporaryPassword != "Temp12345" || !out.MustChangePassword {
		t.Fatalf("unexpected output: %+v", out)
	}
	if repo.resetPasswordUserID != "u1" || repo.resetPasswordHash == "" {
		t.Fatalf("expected password reset write")
	}
}

func TestResetPlatformTenantAdminPassword_MissingTarget(t *testing.T) {
	u := &ResetPlatformTenantAdminPasswordUsecase{Repo: &tenantMockRepo{}}
	_, err := u.Execute(context.Background(), ResetPlatformTenantAdminPasswordInput{})
	if err != domainErr.ErrMissingTenantAdminTarget {
		t.Fatalf("expected missing_tenant_admin_target, got %v", err)
	}
}
