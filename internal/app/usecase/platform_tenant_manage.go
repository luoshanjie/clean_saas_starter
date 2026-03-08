package usecase

import (
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
	"service/internal/domain/port"
)

type CreatePlatformTenantInput struct {
	DisplayName  string
	Province     string
	City         string
	District     string
	Address      string
	ContactName  string
	ContactPhone string
	Status       string
	AdminAccount string
	AdminName    string
	AdminPhone   string
	Remark       string
	Role         string
}

type CreatePlatformTenantOutput struct {
	TenantID           string
	TenantAdminUserID  string
	TenantAdminAccount string
	TenantAdminName    string
	Status             string
	CreatedAt          time.Time
}

type CreatePlatformTenantUsecase struct {
	Repo           port.TenantRepo
	Perm           port.PermissionChecker
	IDGen          func() string
	Now            func() time.Time
	InitialPassGen func() string
}

func (u *CreatePlatformTenantUsecase) Execute(ctx context.Context, in CreatePlatformTenantInput) (*CreatePlatformTenantOutput, error) {
	if strings.TrimSpace(in.DisplayName) == "" || strings.TrimSpace(in.AdminAccount) == "" || strings.TrimSpace(in.AdminName) == "" || strings.TrimSpace(in.AdminPhone) == "" {
		return nil, domainErr.ErrValidation
	}
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "inactive" {
		return nil, domainErr.ErrInvalidTenantStatus
	}
	adminPhone, ok := normalizeMainlandPhone(in.AdminPhone)
	if !ok {
		return nil, domainErr.ErrInvalidPhone
	}
	if p := strings.TrimSpace(in.ContactPhone); p != "" {
		if _, ok := normalizeMainlandPhone(p); !ok {
			return nil, domainErr.ErrInvalidPhone
		}
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
	if u.Repo == nil || u.IDGen == nil {
		return nil, domainErr.ErrValidation
	}
	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}
	password := "pass123"
	if u.InitialPassGen != nil {
		password = strings.TrimSpace(u.InitialPassGen())
	}
	if password == "" {
		return nil, domainErr.ErrValidation
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	tenantID, err := canonicalUUIDText(u.IDGen())
	if err != nil {
		return nil, domainErr.ErrValidation
	}
	adminUserID, err := canonicalUUIDText(u.IDGen())
	if err != nil {
		return nil, domainErr.ErrValidation
	}
	out, err := u.Repo.CreateWithAdmin(ctx, &model.TenantCreateInput{
		TenantID:             tenantID,
		DisplayName:          strings.TrimSpace(in.DisplayName),
		Province:             strings.TrimSpace(in.Province),
		City:                 strings.TrimSpace(in.City),
		District:             strings.TrimSpace(in.District),
		Address:              strings.TrimSpace(in.Address),
		ContactName:          strings.TrimSpace(in.ContactName),
		ContactPhone:         strings.TrimSpace(in.ContactPhone),
		Remark:               strings.TrimSpace(in.Remark),
		Status:               status,
		CreatedAt:            now,
		UpdatedAt:            now,
		TenantAdminUserID:    adminUserID,
		TenantAdminAccount:   strings.TrimSpace(in.AdminAccount),
		TenantAdminName:      strings.TrimSpace(in.AdminName),
		TenantAdminPhone:     adminPhone,
		TenantAdminPassword:  string(hash),
		TenantAdminCreatedAt: now,
	})
	if err != nil {
		return nil, err
	}
	return &CreatePlatformTenantOutput{
		TenantID:           out.TenantID,
		TenantAdminUserID:  out.TenantAdminUserID,
		TenantAdminAccount: out.TenantAdminAccount,
		TenantAdminName:    out.TenantAdminName,
		Status:             out.Status,
		CreatedAt:          out.CreatedAt,
	}, nil
}

type ListPlatformTenantsInput struct {
	Keyword   string
	Province  string
	City      string
	District  string
	Status    string
	NeedTotal bool
	Page      int
	PageSize  int
	Role      string
}

type ListPlatformTenantsOutput struct {
	Items []*model.TenantListItem
	Total int
}

type ListPlatformTenantsUsecase struct {
	Repo port.TenantRepo
	Perm port.PermissionChecker
}

func (u *ListPlatformTenantsUsecase) Execute(ctx context.Context, in ListPlatformTenantsInput) (*ListPlatformTenantsOutput, error) {
	if in.Status != "" && in.Status != "active" && in.Status != "inactive" {
		return nil, domainErr.ErrInvalidTenantStatus
	}
	if u.Perm != nil {
		allowed, err := u.Perm.Enforce(ctx, in.Role, "platform.tenant.list")
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, domainErr.ErrForbidden
		}
	}
	items, total, err := u.Repo.ListPage(ctx, port.TenantFilter{
		Keyword:   strings.TrimSpace(in.Keyword),
		Province:  strings.TrimSpace(in.Province),
		City:      strings.TrimSpace(in.City),
		District:  strings.TrimSpace(in.District),
		Status:    in.Status,
		NeedTotal: in.NeedTotal,
		Page:      in.Page,
		PageSize:  in.PageSize,
	})
	if err != nil {
		return nil, err
	}
	return &ListPlatformTenantsOutput{Items: items, Total: total}, nil
}

type UpdatePlatformTenantInput struct {
	TenantID     string
	DisplayName  string
	Province     string
	City         string
	District     string
	Address      string
	ContactName  string
	ContactPhone string
	Remark       string
	Role         string
}

type UpdatePlatformTenantUsecase struct {
	Repo port.TenantRepo
	Perm port.PermissionChecker
	Now  func() time.Time
}

func (u *UpdatePlatformTenantUsecase) Execute(ctx context.Context, in UpdatePlatformTenantInput) error {
	if strings.TrimSpace(in.TenantID) == "" || strings.TrimSpace(in.DisplayName) == "" {
		return domainErr.ErrValidation
	}
	if p := strings.TrimSpace(in.ContactPhone); p != "" {
		if _, ok := normalizeMainlandPhone(p); !ok {
			return domainErr.ErrInvalidPhone
		}
	}
	if u.Perm != nil {
		allowed, err := u.Perm.Enforce(ctx, in.Role, "platform.tenant.update")
		if err != nil {
			return err
		}
		if !allowed {
			return domainErr.ErrForbidden
		}
	}
	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}
	updated, err := u.Repo.Update(ctx, &model.Tenant{
		ID:           strings.TrimSpace(in.TenantID),
		DisplayName:  strings.TrimSpace(in.DisplayName),
		Province:     strings.TrimSpace(in.Province),
		City:         strings.TrimSpace(in.City),
		District:     strings.TrimSpace(in.District),
		Address:      strings.TrimSpace(in.Address),
		ContactName:  strings.TrimSpace(in.ContactName),
		ContactPhone: strings.TrimSpace(in.ContactPhone),
		Remark:       strings.TrimSpace(in.Remark),
		UpdatedAt:    now,
	})
	if err != nil {
		return err
	}
	if !updated {
		return domainErr.ErrTenantNotFound
	}
	return nil
}

type TogglePlatformTenantStatusInput struct {
	TenantID string
	Status   string
	Role     string
}

type TogglePlatformTenantStatusUsecase struct {
	Repo port.TenantRepo
	Perm port.PermissionChecker
}

func (u *TogglePlatformTenantStatusUsecase) Execute(ctx context.Context, in TogglePlatformTenantStatusInput) error {
	if strings.TrimSpace(in.TenantID) == "" {
		return domainErr.ErrValidation
	}
	if in.Status != "active" && in.Status != "inactive" {
		return domainErr.ErrInvalidTenantStatus
	}
	if u.Perm != nil {
		allowed, err := u.Perm.Enforce(ctx, in.Role, "platform.tenant.disable")
		if err != nil {
			return err
		}
		if !allowed {
			return domainErr.ErrForbidden
		}
	}
	updated, err := u.Repo.ToggleStatus(ctx, strings.TrimSpace(in.TenantID), in.Status)
	if err != nil {
		return err
	}
	if !updated {
		return domainErr.ErrTenantNotFound
	}
	return nil
}

type ResetPlatformTenantAdminAuthInput struct {
	TenantID string
	Action   string
	Role     string
}

type ResetPlatformTenantAdminAuthUsecase struct {
	Repo port.TenantRepo
	Perm port.PermissionChecker
}

func (u *ResetPlatformTenantAdminAuthUsecase) Execute(ctx context.Context, in ResetPlatformTenantAdminAuthInput) error {
	if strings.TrimSpace(in.TenantID) == "" {
		return domainErr.ErrValidation
	}
	if in.Action != "resend_activate" {
		return domainErr.ErrInvalidResetAction
	}
	if u.Perm != nil {
		allowed, err := u.Perm.Enforce(ctx, in.Role, "platform.tenant.reset_auth")
		if err != nil {
			return err
		}
		if !allowed {
			return domainErr.ErrForbidden
		}
	}
	_, err := u.Repo.GetByID(ctx, strings.TrimSpace(in.TenantID))
	if err != nil {
		if errors.Is(err, domainErr.ErrNotFound) {
			return domainErr.ErrTenantNotFound
		}
		return err
	}
	hasAdmin, err := u.Repo.HasTenantAdmin(ctx, strings.TrimSpace(in.TenantID))
	if err != nil {
		return err
	}
	if !hasAdmin {
		return domainErr.ErrNotFound
	}
	return nil
}
