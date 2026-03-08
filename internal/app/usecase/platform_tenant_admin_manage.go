package usecase

import (
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"strings"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/port"
)

type ChangePlatformTenantAdminInput struct {
	TenantID     string
	AdminAccount string
	AdminName    string
	AdminPhone   string
	Role         string
}

type ChangePlatformTenantAdminOutput struct {
	AdminUserID  string
	AdminAccount string
	AdminName    string
	AdminPhone   string
}

type ChangePlatformTenantAdminUsecase struct {
	Repo port.TenantRepo
	Perm port.PermissionChecker
}

func (u *ChangePlatformTenantAdminUsecase) Execute(ctx context.Context, in ChangePlatformTenantAdminInput) (*ChangePlatformTenantAdminOutput, error) {
	if strings.TrimSpace(in.TenantID) == "" || strings.TrimSpace(in.AdminAccount) == "" || strings.TrimSpace(in.AdminName) == "" || strings.TrimSpace(in.AdminPhone) == "" {
		return nil, domainErr.ErrValidation
	}
	adminPhone, ok := normalizeMainlandPhone(in.AdminPhone)
	if !ok {
		return nil, domainErr.ErrInvalidPhone
	}
	if u.Perm != nil {
		allowed, err := u.Perm.Enforce(ctx, in.Role, "platform.tenant.update")
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, domainErr.ErrForbidden
		}
	}
	adminUser, oldAccount, err := u.Repo.GetTenantAdminByTenantID(ctx, strings.TrimSpace(in.TenantID))
	if err != nil {
		if errors.Is(err, domainErr.ErrNotFound) {
			return nil, domainErr.ErrTenantAdminNotFound
		}
		return nil, err
	}
	if oldAccount != strings.TrimSpace(in.AdminAccount) {
		exists, err := u.Repo.TenantAdminAccountExists(ctx, strings.TrimSpace(in.AdminAccount))
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, domainErr.ErrTenantAdminAccountExists
		}
	}
	if adminUser.Phone != adminPhone {
		exists, err := u.Repo.TenantAdminPhoneExists(ctx, adminPhone)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, domainErr.ErrTenantAdminPhoneExists
		}
	}
	if err := u.Repo.UpdateTenantAdminIdentity(ctx, adminUser.ID, strings.TrimSpace(in.AdminAccount), strings.TrimSpace(in.AdminName), adminPhone); err != nil {
		return nil, err
	}
	return &ChangePlatformTenantAdminOutput{
		AdminUserID:  adminUser.ID,
		AdminAccount: strings.TrimSpace(in.AdminAccount),
		AdminName:    strings.TrimSpace(in.AdminName),
		AdminPhone:   adminPhone,
	}, nil
}

type ResetPlatformTenantAdminPasswordInput struct {
	TenantID    string
	AdminUserID string
	Role        string
}

type ResetPlatformTenantAdminPasswordOutput struct {
	AdminUserID        string
	TemporaryPassword  string
	MustChangePassword bool
}

type ResetPlatformTenantAdminPasswordUsecase struct {
	Repo    port.TenantRepo
	Perm    port.PermissionChecker
	PassGen func() string
}

func (u *ResetPlatformTenantAdminPasswordUsecase) Execute(ctx context.Context, in ResetPlatformTenantAdminPasswordInput) (*ResetPlatformTenantAdminPasswordOutput, error) {
	if strings.TrimSpace(in.TenantID) == "" && strings.TrimSpace(in.AdminUserID) == "" {
		return nil, domainErr.ErrMissingTenantAdminTarget
	}
	if u.Perm != nil {
		allowed, err := u.Perm.Enforce(ctx, in.Role, "platform.tenant.reset_auth")
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, domainErr.ErrForbidden
		}
	}
	var (
		adminUserID string
		err         error
	)
	if strings.TrimSpace(in.AdminUserID) != "" {
		adminUser, _, e := u.Repo.GetTenantAdminByUserID(ctx, strings.TrimSpace(in.AdminUserID))
		if e != nil {
			if errors.Is(e, domainErr.ErrNotFound) {
				return nil, domainErr.ErrTenantAdminNotFound
			}
			return nil, e
		}
		adminUserID = adminUser.ID
	} else {
		adminUser, _, e := u.Repo.GetTenantAdminByTenantID(ctx, strings.TrimSpace(in.TenantID))
		if e != nil {
			if errors.Is(e, domainErr.ErrNotFound) {
				return nil, domainErr.ErrTenantAdminNotFound
			}
			return nil, e
		}
		adminUserID = adminUser.ID
	}
	tempPassword := ""
	if u.PassGen != nil {
		tempPassword = strings.TrimSpace(u.PassGen())
	}
	if tempPassword == "" {
		tempPassword = generateTemporaryPassword(10)
	}
	if tempPassword == "" {
		return nil, domainErr.ErrValidation
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	err = u.Repo.ResetTenantAdminPassword(ctx, adminUserID, string(hash))
	if err != nil {
		return nil, err
	}
	return &ResetPlatformTenantAdminPasswordOutput{
		AdminUserID:        adminUserID,
		TemporaryPassword:  tempPassword,
		MustChangePassword: true,
	}, nil
}
