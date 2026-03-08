package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"service/internal/app/usecase"
	"service/internal/delivery/http/middleware"
	"service/internal/delivery/http/resp"
	"service/internal/domain/authctx"
	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
)

type AuthLoginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type AuthLoginResponse struct {
	RequiresSecondFactor bool    `json:"requires_second_factor"`
	ChallengeID          string  `json:"challenge_id,omitempty"`
	MaskedPhone          string  `json:"masked_phone,omitempty"`
	AccessToken          string  `json:"access_token,omitempty"`
	RefreshToken         string  `json:"refresh_token,omitempty"`
	ExpiresIn            int     `json:"expires_in"`
	MustChangePassword   bool    `json:"must_change_password"`
	PasswordUpdatedAt    *string `json:"password_updated_at,omitempty"`
}

type AuthLoginVerifyRequest struct {
	ChallengeID string `json:"challenge_id"`
	OTPCode     string `json:"otp_code"`
}

type AuthLoginVerifyResponse struct {
	AccessToken        string  `json:"access_token"`
	RefreshToken       string  `json:"refresh_token"`
	ExpiresIn          int     `json:"expires_in"`
	MustChangePassword bool    `json:"must_change_password"`
	PasswordUpdatedAt  *string `json:"password_updated_at,omitempty"`
}

type AuthMeResponse struct {
	User               map[string]any `json:"user"`
	Role               string         `json:"role"`
	ScopeType          string         `json:"scope_type"`
	TenantID           string         `json:"tenant_id"`
	TenantName         string         `json:"tenant_name"`
	Permissions        []string       `json:"permissions"`
	MustChangePassword bool           `json:"must_change_password"`
	PasswordUpdatedAt  *string        `json:"password_updated_at,omitempty"`
	DataScope          map[string]any `json:"data_scope"`
}

type AuthChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type AuthUpdateDisplayNameRequest struct {
	Name string `json:"name"`
}

type AuthChangePhoneChallengeRequest struct {
	NewPhone    string `json:"new_phone"`
	OldPassword string `json:"old_password"`
}

type AuthChangePhoneChallengeResponse struct {
	ChallengeID    string `json:"challenge_id"`
	MaskedNewPhone string `json:"masked_new_phone"`
	ExpiresIn      int    `json:"expires_in"`
	ResendIn       int    `json:"resend_in"`
}

type AuthChangePhoneResendRequest struct {
	ChallengeID string `json:"challenge_id"`
}

type AuthChangePhoneVerifyRequest struct {
	ChallengeID string `json:"challenge_id"`
	OTPCode     string `json:"otp_code"`
}

type AuthChangePhoneVerifyResponse struct {
	Status       string `json:"status"`
	ForceRelogin bool   `json:"force_relogin"`
}

type AuthHandler struct {
	LoginUC                  *usecase.AuthLoginUsecase
	LoginChallengeUC         *usecase.AuthLoginChallengeUsecase
	LoginVerifyUC            *usecase.AuthLoginVerifyUsecase
	RefreshUC                *usecase.AuthRefreshUsecase
	MeUC                     *usecase.AuthMeUsecase
	ChangePasswordUC         *usecase.AuthChangePasswordUsecase
	UpdateDisplayNameUC      *usecase.AuthUpdateDisplayNameUsecase
	ChangePhoneChallengeUC   *usecase.AuthChangePhoneChallengeUsecase
	ChangePhoneResendUC      *usecase.AuthChangePhoneResendUsecase
	ChangePhoneVerifyUC      *usecase.AuthChangePhoneVerifyUsecase
	AuditUC                  *usecase.AuditWriteUsecase
	JWT                      middleware.JWTMiddleware
	LoginSecondFactorEnabled bool
}

// @Summary      Login
// @Description  Verify account/password. Returns OTP challenge when second factor is enabled, otherwise returns tokens directly.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  AuthLoginRequest  true  "request"
// @Success      200   {object}  resp.AuthLoginEnvelope
// @Failure      200   {object}  resp.AuthLoginEnvelope
// @Router        /api/v1/auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req AuthLoginRequest
	if err := c.Bind(&req); err != nil {
		logAudit(c, h.AuditUC, usecase.AuditWriteInput{
			TargetType: "auth",
			Action:     "login",
			Module:     "auth",
			Result:     "fail",
			ErrorCode:  "bad_request",
		})
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}

	ctx := authctx.With(c.Request().Context(), authctx.Info{ScopeType: "platform"})
	if h.LoginSecondFactorEnabled {
		if h.LoginChallengeUC == nil {
			return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeServerError, "login_second_factor_not_configured"))
		}
		out, err := h.LoginChallengeUC.Execute(ctx, usecase.AuthLoginChallengeInput{
			Account:  req.Account,
			Password: req.Password,
		})
		if err != nil {
			logAudit(c, h.AuditUC, usecase.AuditWriteInput{
				TargetType: "auth",
				Action:     "login",
				Module:     "auth",
				Result:     "fail",
				ErrorCode:  err.Error(),
			})
			return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeUnauthorized, err.Error()))
		}
		logAudit(c, h.AuditUC, usecase.AuditWriteInput{
			TargetType: "auth",
			Action:     "login",
			Module:     "auth",
			Result:     "success",
			AfterJSON: map[string]any{
				"requires_second_factor": true,
			},
		})

		return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), AuthLoginResponse{
			RequiresSecondFactor: true,
			ChallengeID:          out.ChallengeID,
			MaskedPhone:          out.MaskedPhone,
			ExpiresIn:            out.ExpiresIn,
		}))
	}

	if h.LoginUC == nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeServerError, "login_not_configured"))
	}
	out, err := h.LoginUC.Execute(ctx, usecase.AuthLoginInput{
		Account:  req.Account,
		Password: req.Password,
	})
	if err != nil {
		logAudit(c, h.AuditUC, usecase.AuditWriteInput{
			TargetType: "auth",
			Action:     "login",
			Module:     "auth",
			Result:     "fail",
			ErrorCode:  err.Error(),
		})
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeUnauthorized, err.Error()))
	}
	var passwordUpdatedAt *string
	if out.User != nil && out.User.PasswordUpdatedAt != nil {
		s := out.User.PasswordUpdatedAt.UTC().Format(time.RFC3339)
		passwordUpdatedAt = &s
	}
	mustChange := usecase.ShouldForcePasswordChange(out.User)
	targetID := ""
	if out.User != nil {
		targetID = out.User.ID
	}
	logAudit(c, h.AuditUC, usecase.AuditWriteInput{
		TargetType: "user",
		TargetID:   targetID,
		Action:     "login",
		Module:     "auth",
		Result:     "success",
		AfterJSON: map[string]any{
			"requires_second_factor": false,
			"must_change_password":   mustChange,
		},
	})
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), AuthLoginResponse{
		RequiresSecondFactor: false,
		AccessToken:          out.AccessToken,
		RefreshToken:         out.RefreshToken,
		ExpiresIn:            out.ExpiresIn,
		MustChangePassword:   mustChange,
		PasswordUpdatedAt:    passwordUpdatedAt,
	}))
}

// @Summary      Verify login OTP
// @Description  Verify challenge and issue tokens.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  AuthLoginVerifyRequest  true  "request"
// @Success      200   {object}  resp.AuthLoginEnvelope
// @Failure      200   {object}  resp.AuthLoginEnvelope
// @Router        /api/v1/auth/login/verify [post]
func (h *AuthHandler) LoginVerify(c echo.Context) error {
	if !h.LoginSecondFactorEnabled {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "login_second_factor_not_enabled"))
	}
	if h.LoginVerifyUC == nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeServerError, "login_second_factor_not_configured"))
	}
	var req AuthLoginVerifyRequest
	if err := c.Bind(&req); err != nil {
		logAudit(c, h.AuditUC, usecase.AuditWriteInput{
			TargetType: "auth",
			Action:     "login_verify",
			Module:     "auth",
			Result:     "fail",
			ErrorCode:  "bad_request",
		})
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}

	ctx := authctx.With(c.Request().Context(), authctx.Info{ScopeType: "platform"})
	out, err := h.LoginVerifyUC.Execute(ctx, usecase.AuthLoginVerifyInput{
		ChallengeID: req.ChallengeID,
		OTPCode:     req.OTPCode,
	})
	if err != nil {
		logAudit(c, h.AuditUC, usecase.AuditWriteInput{
			TargetType: "auth",
			Action:     "login_verify",
			Module:     "auth",
			Result:     "fail",
			ErrorCode:  err.Error(),
		})
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeUnauthorized, err.Error()))
	}
	var passwordUpdatedAt *string
	if out.User != nil && out.User.PasswordUpdatedAt != nil {
		s := out.User.PasswordUpdatedAt.UTC().Format(time.RFC3339)
		passwordUpdatedAt = &s
	}
	mustChange := usecase.ShouldForcePasswordChange(out.User)
	targetID := ""
	if out.User != nil {
		targetID = out.User.ID
	}
	logAudit(c, h.AuditUC, usecase.AuditWriteInput{
		TargetType: "user",
		TargetID:   targetID,
		Action:     "login_verify",
		Module:     "auth",
		Result:     "success",
		AfterJSON: map[string]any{
			"must_change_password": mustChange,
		},
	})

	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), AuthLoginVerifyResponse{
		AccessToken:        out.AccessToken,
		RefreshToken:       out.RefreshToken,
		ExpiresIn:          out.ExpiresIn,
		MustChangePassword: mustChange,
		PasswordUpdatedAt:  passwordUpdatedAt,
	}))
}

// @Summary      Refresh token
// @Description  Refresh access token using refresh token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  AuthRefreshRequest  true  "request"
// @Success      200   {object}  resp.AuthRefreshEnvelope
// @Failure      200   {object}  resp.AuthRefreshEnvelope
// @Router        /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c echo.Context) error {
	var req AuthRefreshRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}

	ctx := authctx.With(c.Request().Context(), authctx.Info{ScopeType: "platform"})
	out, err := h.RefreshUC.Execute(ctx, usecase.AuthRefreshInput{RefreshToken: req.RefreshToken})
	if err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeUnauthorized, err.Error()))
	}

	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), AuthRefreshResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		ExpiresIn:    out.ExpiresIn,
	}))
}

// @Summary      Me
// @Description  Return current user info.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200   {object}  resp.AuthMeEnvelope
// @Failure      200   {object}  resp.AuthMeEnvelope
// @Router        /api/v1/auth/me [post]
func (h *AuthHandler) Me(c echo.Context) error {
	user := &model.User{
		ID:        middleware.GetUserID(c),
		Name:      middleware.GetUserName(c),
		TenantID:  middleware.GetTenantID(c),
		Role:      middleware.GetRole(c),
		ScopeType: middleware.GetScopeType(c),
	}

	out := h.MeUC.Execute(c.Request().Context(), usecase.AuthMeInput{User: user})
	var passwordUpdatedAt *string
	if out.PasswordUpdatedAt != nil {
		s := out.PasswordUpdatedAt.UTC().Format(time.RFC3339)
		passwordUpdatedAt = &s
	}

	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), AuthMeResponse{
		User: map[string]any{
			"id":   out.User.ID,
			"name": out.User.Name,
		},
		Role:               out.Role,
		ScopeType:          out.ScopeType,
		TenantID:           out.TenantID,
		TenantName:         out.TenantName,
		Permissions:        out.Permissions,
		MustChangePassword: out.MustChangePassword,
		PasswordUpdatedAt:  passwordUpdatedAt,
		DataScope:          map[string]any{},
	}))
}

// @Summary      Change password
// @Description  Change current user's password.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  AuthChangePasswordRequest  true  "request"
// @Success      200   {object}  resp.AuthMeEnvelope
// @Failure      200   {object}  resp.AuthMeEnvelope
// @Router       /api/v1/auth/change-password [post]
func (h *AuthHandler) ChangePassword(c echo.Context) error {
	var req AuthChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	if h.ChangePasswordUC == nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeServerError, "change_password_not_enabled"))
	}
	err := h.ChangePasswordUC.Execute(c.Request().Context(), usecase.AuthChangePasswordInput{
		UserID:      middleware.GetUserID(c),
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		logAudit(c, h.AuditUC, usecase.AuditWriteInput{
			TargetType: "user",
			TargetID:   middleware.GetUserID(c),
			Action:     "change_password",
			Module:     "auth",
			Result:     "fail",
			ErrorCode:  err.Error(),
		})
		code := resp.CodeServerError
		if err == domainErr.ErrValidation || err == domainErr.ErrInvalidOldPassword || err == domainErr.ErrInvalidNewPassword {
			code = resp.CodeValidation
		} else if err == domainErr.ErrUnauthenticated {
			code = resp.CodeUnauthorized
		}
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), code, err.Error()))
	}
	logAudit(c, h.AuditUC, usecase.AuditWriteInput{
		TargetType: "user",
		TargetID:   middleware.GetUserID(c),
		Action:     "change_password",
		Module:     "auth",
		Result:     "success",
		ChangedFields: []string{
			"password_hash",
			"must_change_password",
			"password_updated_at",
		},
	})
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{"status": "ok"}))
}

// @Summary      Update display name
// @Description  Update current user's display name (account/login name remains immutable).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  AuthUpdateDisplayNameRequest  true  "request"
// @Success      200   {object}  resp.AuthMeEnvelope
// @Failure      200   {object}  resp.AuthMeEnvelope
// @Router       /api/v1/auth/profile/update-name [post]
func (h *AuthHandler) UpdateDisplayName(c echo.Context) error {
	var req AuthUpdateDisplayNameRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	if h.UpdateDisplayNameUC == nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeServerError, "update_display_name_not_enabled"))
	}
	out, err := h.UpdateDisplayNameUC.Execute(c.Request().Context(), usecase.AuthUpdateDisplayNameInput{
		UserID: middleware.GetUserID(c),
		Name:   req.Name,
	})
	if err != nil {
		logAudit(c, h.AuditUC, usecase.AuditWriteInput{
			TargetType: "user",
			TargetID:   middleware.GetUserID(c),
			Action:     "update_display_name",
			Module:     "auth",
			Result:     "fail",
			ErrorCode:  err.Error(),
		})
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), mapUpdateDisplayNameErrCode(err), err.Error()))
	}
	logAudit(c, h.AuditUC, usecase.AuditWriteInput{
		TargetType: "user",
		TargetID:   middleware.GetUserID(c),
		Action:     "update_display_name",
		Module:     "auth",
		Result:     "success",
		BeforeJSON: map[string]any{
			"name": middleware.GetUserName(c),
		},
		AfterJSON: map[string]any{
			"name": out.Name,
		},
		ChangedFields: []string{"name"},
	})
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{
		"status": out.Status,
		"name":   out.Name,
	}))
}

// @Summary      Change phone challenge
// @Description  Verify old password and send OTP to new phone.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  AuthChangePhoneChallengeRequest  true  "request"
// @Success      200   {object}  resp.AuthMeEnvelope
// @Failure      200   {object}  resp.AuthMeEnvelope
// @Router       /api/v1/auth/change-phone/challenge [post]
func (h *AuthHandler) ChangePhoneChallenge(c echo.Context) error {
	var req AuthChangePhoneChallengeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	if h.ChangePhoneChallengeUC == nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeServerError, "change_phone_not_enabled"))
	}
	out, err := h.ChangePhoneChallengeUC.Execute(c.Request().Context(), usecase.AuthChangePhoneChallengeInput{
		UserID:      middleware.GetUserID(c),
		Role:        middleware.GetRole(c),
		NewPhone:    req.NewPhone,
		OldPassword: req.OldPassword,
	})
	if err != nil {
		logAudit(c, h.AuditUC, usecase.AuditWriteInput{
			TargetType: "user",
			TargetID:   middleware.GetUserID(c),
			Action:     "change_phone_challenge",
			Module:     "auth",
			Result:     "fail",
			ErrorCode:  err.Error(),
		})
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), mapChangePhoneErrCode(err), err.Error()))
	}
	logAudit(c, h.AuditUC, usecase.AuditWriteInput{
		TargetType: "user",
		TargetID:   middleware.GetUserID(c),
		Action:     "change_phone_challenge",
		Module:     "auth",
		Result:     "success",
		AfterJSON: map[string]any{
			"after_phone_masked": out.MaskedNewPhone,
		},
		ChangedFields: []string{"phone"},
	})
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), AuthChangePhoneChallengeResponse{
		ChallengeID:    out.ChallengeID,
		MaskedNewPhone: out.MaskedNewPhone,
		ExpiresIn:      out.ExpiresIn,
		ResendIn:       out.ResendIn,
	}))
}

// @Summary      Change phone resend OTP
// @Description  Resend OTP for phone change challenge.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  AuthChangePhoneResendRequest  true  "request"
// @Success      200   {object}  resp.AuthMeEnvelope
// @Failure      200   {object}  resp.AuthMeEnvelope
// @Router       /api/v1/auth/change-phone/resend [post]
func (h *AuthHandler) ChangePhoneResend(c echo.Context) error {
	var req AuthChangePhoneResendRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	if h.ChangePhoneResendUC == nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeServerError, "change_phone_not_enabled"))
	}
	out, err := h.ChangePhoneResendUC.Execute(c.Request().Context(), usecase.AuthChangePhoneResendInput{
		UserID:      middleware.GetUserID(c),
		Role:        middleware.GetRole(c),
		ChallengeID: req.ChallengeID,
	})
	if err != nil {
		logAudit(c, h.AuditUC, usecase.AuditWriteInput{
			TargetType: "user",
			TargetID:   middleware.GetUserID(c),
			Action:     "change_phone_resend",
			Module:     "auth",
			Result:     "fail",
			ErrorCode:  err.Error(),
		})
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), mapChangePhoneErrCode(err), err.Error()))
	}
	logAudit(c, h.AuditUC, usecase.AuditWriteInput{
		TargetType: "user",
		TargetID:   middleware.GetUserID(c),
		Action:     "change_phone_resend",
		Module:     "auth",
		Result:     "success",
		AfterJSON: map[string]any{
			"after_phone_masked": out.MaskedNewPhone,
		},
	})
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), AuthChangePhoneChallengeResponse{
		ChallengeID:    out.ChallengeID,
		MaskedNewPhone: out.MaskedNewPhone,
		ExpiresIn:      out.ExpiresIn,
		ResendIn:       out.ResendIn,
	}))
}

// @Summary      Change phone verify OTP
// @Description  Verify OTP and update current user's phone.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  AuthChangePhoneVerifyRequest  true  "request"
// @Success      200   {object}  resp.AuthMeEnvelope
// @Failure      200   {object}  resp.AuthMeEnvelope
// @Router       /api/v1/auth/change-phone/verify [post]
func (h *AuthHandler) ChangePhoneVerify(c echo.Context) error {
	var req AuthChangePhoneVerifyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	if h.ChangePhoneVerifyUC == nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeServerError, "change_phone_not_enabled"))
	}
	out, err := h.ChangePhoneVerifyUC.Execute(c.Request().Context(), usecase.AuthChangePhoneVerifyInput{
		UserID:      middleware.GetUserID(c),
		Role:        middleware.GetRole(c),
		ChallengeID: req.ChallengeID,
		OTPCode:     req.OTPCode,
	})
	if err != nil {
		logAudit(c, h.AuditUC, usecase.AuditWriteInput{
			TargetType: "user",
			TargetID:   middleware.GetUserID(c),
			Action:     "change_phone_verify",
			Module:     "auth",
			Result:     "fail",
			ErrorCode:  err.Error(),
		})
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), mapChangePhoneErrCode(err), err.Error()))
	}
	logAudit(c, h.AuditUC, usecase.AuditWriteInput{
		TargetType: "user",
		TargetID:   middleware.GetUserID(c),
		Action:     "change_phone_verify",
		Module:     "auth",
		Result:     "success",
		BeforeJSON: map[string]any{
			"before_phone_masked": out.BeforePhoneMasked,
			"request_id":          middleware.GetRequestID(c),
		},
		AfterJSON: map[string]any{
			"after_phone_masked": out.AfterPhoneMasked,
			"request_id":         middleware.GetRequestID(c),
		},
		ChangedFields: []string{"phone", "token_version"},
	})
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), AuthChangePhoneVerifyResponse{
		Status:       out.Status,
		ForceRelogin: out.ForceRelogin,
	}))
}

func mapChangePhoneErrCode(err error) int {
	switch err {
	case domainErr.ErrForbidden, domainErr.ErrChallengeUserMismatch:
		return resp.CodeForbidden
	case domainErr.ErrValidation,
		domainErr.ErrInvalidPhone,
		domainErr.ErrInvalidOldPassword,
		domainErr.ErrPhoneExists,
		domainErr.ErrChallengeNotFoundOrExpired,
		domainErr.ErrOTPInvalidOrExpired,
		domainErr.ErrOTPAttemptsExceeded:
		return resp.CodeValidation
	case domainErr.ErrUnauthenticated:
		return resp.CodeUnauthorized
	default:
		return resp.CodeServerError
	}
}

func mapUpdateDisplayNameErrCode(err error) int {
	switch err {
	case domainErr.ErrValidation, domainErr.ErrInvalidDisplayName:
		return resp.CodeValidation
	case domainErr.ErrUnauthenticated:
		return resp.CodeUnauthorized
	default:
		return resp.CodeServerError
	}
}
