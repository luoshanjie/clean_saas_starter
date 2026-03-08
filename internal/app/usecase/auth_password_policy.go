package usecase

import "service/internal/domain/model"

func ShouldForcePasswordChange(user *model.User) bool {
	if user == nil {
		return false
	}
	if user.Role != "tenant_admin" && user.Role != "tenant_member" {
		return false
	}
	if user.MustChangePassword {
		return true
	}
	return user.PasswordUpdatedAt == nil
}
