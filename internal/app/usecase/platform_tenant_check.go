package usecase

import (
	"context"
	"strings"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/port"
)

type CheckPlatformTenantDisplayNameInput struct {
	DisplayName string
	Role        string
}

type CheckPlatformTenantDisplayNameOutput struct {
	Available bool
	Reason    string
}

type CheckPlatformTenantDisplayNameUsecase struct {
	Repo port.TenantRepo
	Perm port.PermissionChecker
}

func (u *CheckPlatformTenantDisplayNameUsecase) Execute(ctx context.Context, in CheckPlatformTenantDisplayNameInput) (*CheckPlatformTenantDisplayNameOutput, error) {
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		return &CheckPlatformTenantDisplayNameOutput{Available: false, Reason: "invalid_param"}, nil
	}
	if u.Perm != nil {
		allowed, err := u.Perm.Enforce(ctx, in.Role, "platform.tenant.create")
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, domainErr.ErrForbidden
		}
	}
	exists, err := u.Repo.DisplayNameExists(ctx, displayName)
	if err != nil {
		return nil, err
	}
	if exists {
		return &CheckPlatformTenantDisplayNameOutput{Available: false, Reason: "tenant_display_name_exists"}, nil
	}
	return &CheckPlatformTenantDisplayNameOutput{Available: true}, nil
}

type CheckPlatformTenantAdminAccountInput struct {
	AdminAccount string
	Role         string
}

type CheckPlatformTenantAdminAccountOutput struct {
	Available bool
	Reason    string
}

type CheckPlatformTenantAdminAccountUsecase struct {
	Repo port.TenantRepo
	Perm port.PermissionChecker
}

func (u *CheckPlatformTenantAdminAccountUsecase) Execute(ctx context.Context, in CheckPlatformTenantAdminAccountInput) (*CheckPlatformTenantAdminAccountOutput, error) {
	adminAccount := strings.TrimSpace(in.AdminAccount)
	if adminAccount == "" {
		return &CheckPlatformTenantAdminAccountOutput{Available: false, Reason: "invalid_param"}, nil
	}
	if u.Perm != nil {
		allowed, err := u.Perm.Enforce(ctx, in.Role, "platform.tenant.create")
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, domainErr.ErrForbidden
		}
	}
	exists, err := u.Repo.TenantAdminAccountExists(ctx, adminAccount)
	if err != nil {
		return nil, err
	}
	if exists {
		return &CheckPlatformTenantAdminAccountOutput{Available: false, Reason: "tenant_admin_account_exists"}, nil
	}
	return &CheckPlatformTenantAdminAccountOutput{Available: true}, nil
}

type CheckPlatformTenantAdminPhoneInput struct {
	AdminPhone string
	Role       string
}

type CheckPlatformTenantAdminPhoneOutput struct {
	Available bool
	Reason    string
}

type CheckPlatformTenantAdminPhoneUsecase struct {
	Repo port.TenantRepo
	Perm port.PermissionChecker
}

func (u *CheckPlatformTenantAdminPhoneUsecase) Execute(ctx context.Context, in CheckPlatformTenantAdminPhoneInput) (*CheckPlatformTenantAdminPhoneOutput, error) {
	adminPhone := strings.TrimSpace(in.AdminPhone)
	if adminPhone == "" {
		return &CheckPlatformTenantAdminPhoneOutput{Available: false, Reason: "invalid_param"}, nil
	}
	normalizedPhone, ok := normalizeMainlandPhone(adminPhone)
	if !ok {
		return &CheckPlatformTenantAdminPhoneOutput{Available: false, Reason: "invalid_phone"}, nil
	}
	if u.Perm != nil {
		allowed, err := u.Perm.Enforce(ctx, in.Role, "platform.tenant.create")
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, domainErr.ErrForbidden
		}
	}
	exists, err := u.Repo.TenantAdminPhoneExists(ctx, normalizedPhone)
	if err != nil {
		return nil, err
	}
	if exists {
		return &CheckPlatformTenantAdminPhoneOutput{Available: false, Reason: "tenant_admin_phone_exists"}, nil
	}
	return &CheckPlatformTenantAdminPhoneOutput{Available: true}, nil
}
