package usecase

import (
	"context"
	"testing"
)

func TestCheckPlatformTenantDisplayName_InvalidParam(t *testing.T) {
	u := &CheckPlatformTenantDisplayNameUsecase{Repo: &tenantMockRepo{}}
	out, err := u.Execute(context.Background(), CheckPlatformTenantDisplayNameInput{DisplayName: "  "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Available || out.Reason != "invalid_param" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestCheckPlatformTenantDisplayName_Exists(t *testing.T) {
	u := &CheckPlatformTenantDisplayNameUsecase{Repo: &tenantMockRepo{displayNameExists: true}}
	out, err := u.Execute(context.Background(), CheckPlatformTenantDisplayNameInput{DisplayName: "Tenant A"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Available || out.Reason != "tenant_display_name_exists" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestCheckPlatformTenantAdminAccount_Exists(t *testing.T) {
	u := &CheckPlatformTenantAdminAccountUsecase{Repo: &tenantMockRepo{adminAccountExists: true}}
	out, err := u.Execute(context.Background(), CheckPlatformTenantAdminAccountInput{AdminAccount: "admin1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Available || out.Reason != "tenant_admin_account_exists" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestCheckPlatformTenantAdminPhone_InvalidPhone(t *testing.T) {
	u := &CheckPlatformTenantAdminPhoneUsecase{Repo: &tenantMockRepo{}}
	out, err := u.Execute(context.Background(), CheckPlatformTenantAdminPhoneInput{AdminPhone: "123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Available || out.Reason != "invalid_phone" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestCheckPlatformTenantAdminPhone_Exists(t *testing.T) {
	u := &CheckPlatformTenantAdminPhoneUsecase{Repo: &tenantMockRepo{adminPhoneExists: true}}
	out, err := u.Execute(context.Background(), CheckPlatformTenantAdminPhoneInput{AdminPhone: "13800138000"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Available || out.Reason != "tenant_admin_phone_exists" {
		t.Fatalf("unexpected output: %+v", out)
	}
}
