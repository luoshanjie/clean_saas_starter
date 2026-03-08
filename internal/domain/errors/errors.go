package errors

import "errors"

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrForbidden       = errors.New("forbidden")
	ErrValidation      = errors.New("validation_error")
	ErrNotFound        = errors.New("not_found")
	ErrConflict        = errors.New("conflict")
	ErrTenantNotFound  = errors.New("tenant_not_found")

	ErrTenantDisplayNameExists    = errors.New("tenant_display_name_exists")
	ErrTenantAdminAccountExists   = errors.New("tenant_admin_account_exists")
	ErrTenantAdminPhoneExists     = errors.New("tenant_admin_phone_exists")
	ErrInvalidTenantStatus        = errors.New("invalid_tenant_status")
	ErrInvalidPhone               = errors.New("invalid_phone")
	ErrInvalidResetAction         = errors.New("invalid_reset_action")
	ErrTenantAdminNotFound        = errors.New("tenant_admin_not_found")
	ErrInvalidOldPassword         = errors.New("invalid_old_password")
	ErrInvalidNewPassword         = errors.New("invalid_new_password")
	ErrInvalidDisplayName         = errors.New("invalid_display_name")
	ErrMissingTenantAdminTarget   = errors.New("missing_tenant_admin_target")
	ErrPhoneExists                = errors.New("phone_exists")
	ErrChallengeNotFoundOrExpired = errors.New("challenge_not_found_or_expired")
	ErrChallengeUserMismatch      = errors.New("challenge_user_mismatch")
	ErrOTPInvalidOrExpired        = errors.New("otp_invalid_or_expired")
	ErrOTPAttemptsExceeded        = errors.New("otp_attempts_exceeded")
)
