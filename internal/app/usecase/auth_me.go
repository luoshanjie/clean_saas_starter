package usecase

import (
	"context"
	"time"

	"service/internal/domain/model"
	"service/internal/domain/port"
)

type AuthMeInput struct {
	User *model.User
}

type AuthMeOutput struct {
	User               *model.User
	Role               string
	ScopeType          string
	TenantID           string
	TenantName         string
	Permissions        []string
	MustChangePassword bool
	PasswordUpdatedAt  *time.Time
}

// AuthMe 直接返回 token 中的身份信息（Phase1 最小闭环）。
func (u *AuthMeUsecase) Execute(ctx context.Context, in AuthMeInput) *AuthMeOutput {
	user := in.User
	if u.AuthRepo != nil && in.User != nil && in.User.ID != "" {
		if latest, err := u.AuthRepo.GetUserByID(ctx, in.User.ID); err == nil && latest != nil {
			user = latest
		}
	}
	perms := []string{}
	if u.Perm != nil && user != nil {
		if p, err := u.Perm.ListByRole(ctx, user.Role); err == nil {
			perms = p
		}
	}
	return &AuthMeOutput{
		User:               user,
		Role:               user.Role,
		ScopeType:          user.ScopeType,
		TenantID:           user.TenantID,
		TenantName:         user.TenantName,
		Permissions:        perms,
		MustChangePassword: ShouldForcePasswordChange(user),
		PasswordUpdatedAt:  user.PasswordUpdatedAt,
	}
}

type AuthMeUsecase struct {
	Perm     port.PermissionChecker
	AuthRepo port.AuthRepo
}
