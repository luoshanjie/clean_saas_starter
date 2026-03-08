package handler

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"service/internal/app/usecase"
	"service/internal/delivery/http/middleware"
	"service/internal/delivery/http/resp"
	domainErr "service/internal/domain/errors"
)

type PlatformTenantCreateRequest struct {
	DisplayName  string `json:"display_name"`
	Province     string `json:"province"`
	City         string `json:"city"`
	District     string `json:"district"`
	Address      string `json:"address"`
	ContactName  string `json:"contact_name"`
	ContactPhone string `json:"contact_phone"`
	Status       string `json:"status"`
	AdminAccount string `json:"admin_account"`
	AdminName    string `json:"admin_name"`
	AdminPhone   string `json:"admin_phone"`
	AdminEmail   string `json:"admin_email"`
	Remark       string `json:"remark"`
}

type PlatformTenantListRequest struct {
	Keyword   string `json:"keyword"`
	Province  string `json:"province"`
	City      string `json:"city"`
	District  string `json:"district"`
	Status    string `json:"status"`
	NeedTotal *bool  `json:"need_total"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
}

type PlatformTenantUpdateRequest struct {
	TenantID     string `json:"tenant_id"`
	DisplayName  string `json:"display_name"`
	Province     string `json:"province"`
	City         string `json:"city"`
	District     string `json:"district"`
	Address      string `json:"address"`
	ContactName  string `json:"contact_name"`
	ContactPhone string `json:"contact_phone"`
	Remark       string `json:"remark"`
}

type PlatformTenantToggleStatusRequest struct {
	TenantID string `json:"tenant_id"`
	Status   string `json:"status"`
}

type PlatformTenantAdminResetAuthRequest struct {
	TenantID string `json:"tenant_id"`
	Action   string `json:"action"`
}

type PlatformTenantChangeAdminRequest struct {
	TenantID     string `json:"tenant_id"`
	AdminAccount string `json:"admin_account"`
	AdminName    string `json:"admin_name"`
	AdminPhone   string `json:"admin_phone"`
}

type PlatformTenantAdminResetPasswordRequest struct {
	TenantID    string `json:"tenant_id"`
	AdminUserID string `json:"admin_user_id"`
}

type PlatformTenantCheckDisplayNameRequest struct {
	DisplayName string `json:"display_name"`
}

type PlatformTenantCheckAdminAccountRequest struct {
	AdminAccount string `json:"admin_account"`
}

type PlatformTenantCheckAdminPhoneRequest struct {
	AdminPhone string `json:"admin_phone"`
}

type PlatformTenantHandler struct {
	CreateUC            *usecase.CreatePlatformTenantUsecase
	ListUC              *usecase.ListPlatformTenantsUsecase
	UpdateUC            *usecase.UpdatePlatformTenantUsecase
	ToggleUC            *usecase.TogglePlatformTenantStatusUsecase
	ResetAuthUC         *usecase.ResetPlatformTenantAdminAuthUsecase
	CheckDisplayNameUC  *usecase.CheckPlatformTenantDisplayNameUsecase
	CheckAdminAccountUC *usecase.CheckPlatformTenantAdminAccountUsecase
	CheckAdminPhoneUC   *usecase.CheckPlatformTenantAdminPhoneUsecase
	ChangeAdminUC       *usecase.ChangePlatformTenantAdminUsecase
	ResetPasswordUC     *usecase.ResetPlatformTenantAdminPasswordUsecase
	AuditUC             *usecase.AuditWriteUsecase
}

// @Summary      Create tenant
// @Tags         platform
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  PlatformTenantCreateRequest  true  "request"
// @Success      200   {object}  resp.APIEnvelope
// @Failure      200   {object}  resp.APIEnvelope
// @Router       /api/v1/platform/tenant/create [post]
func (h *PlatformTenantHandler) Create(c echo.Context) error {
	var req PlatformTenantCreateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	out, err := h.CreateUC.Execute(c.Request().Context(), usecase.CreatePlatformTenantInput{
		DisplayName:  req.DisplayName,
		Province:     req.Province,
		City:         req.City,
		District:     req.District,
		Address:      req.Address,
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		Status:       req.Status,
		AdminAccount: req.AdminAccount,
		AdminName:    req.AdminName,
		AdminPhone:   req.AdminPhone,
		Remark:       req.Remark,
		Role:         middleware.GetRole(c),
	})
	if err != nil {
		logAudit(c, h.AuditUC, usecase.AuditWriteInput{
			TargetType: "tenant",
			TargetName: req.DisplayName,
			Action:     "create",
			Module:     "platform_tenant",
			Result:     "fail",
			ErrorCode:  err.Error(),
		})
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), mapTenantErrCode(err), err.Error()))
	}
	logAudit(c, h.AuditUC, usecase.AuditWriteInput{
		TargetType: "tenant",
		TargetID:   out.TenantID,
		TargetName: req.DisplayName,
		Action:     "create",
		Module:     "platform_tenant",
		Result:     "success",
		AfterJSON: map[string]any{
			"tenant_id":            out.TenantID,
			"tenant_admin_user_id": out.TenantAdminUserID,
			"tenant_admin_account": out.TenantAdminAccount,
			"tenant_admin_name":    out.TenantAdminName,
			"status":               out.Status,
		},
	})
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{
		"tenant_id":            out.TenantID,
		"tenant_admin_user_id": out.TenantAdminUserID,
		"tenant_admin_account": out.TenantAdminAccount,
		"tenant_admin_name":    out.TenantAdminName,
		"status":               out.Status,
		"created_at":           out.CreatedAt,
	}))
}

// @Summary      List tenants
// @Tags         platform
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200   {object}  resp.APIEnvelope
// @Failure      200   {object}  resp.APIEnvelope
// @Router       /api/v1/platform/tenant/list [post]
func (h *PlatformTenantHandler) List(c echo.Context) error {
	var req PlatformTenantListRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	needTotal := true
	if req.NeedTotal != nil {
		needTotal = *req.NeedTotal
	}
	out, err := h.ListUC.Execute(c.Request().Context(), usecase.ListPlatformTenantsInput{
		Keyword:   req.Keyword,
		Province:  req.Province,
		City:      req.City,
		District:  req.District,
		Status:    req.Status,
		NeedTotal: needTotal,
		Page:      req.Page,
		PageSize:  req.PageSize,
		Role:      middleware.GetRole(c),
	})
	if err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), mapTenantErrCode(err), err.Error()))
	}
	items := make([]map[string]any, 0, len(out.Items))
	for _, it := range out.Items {
		items = append(items, map[string]any{
			"tenant_id":            it.Tenant.ID,
			"display_name":         it.Tenant.DisplayName,
			"province":             it.Tenant.Province,
			"city":                 it.Tenant.City,
			"district":             it.Tenant.District,
			"address":              it.Tenant.Address,
			"contact_name":         it.Tenant.ContactName,
			"contact_phone":        it.Tenant.ContactPhone,
			"status":               it.Tenant.Status,
			"tenant_admin_user_id": it.TenantAdminUserID,
			"tenant_admin_account": it.TenantAdminAccount,
			"tenant_admin_name":    it.TenantAdminName,
			"tenant_admin_phone":   it.TenantAdminPhone,
			"created_at":           it.Tenant.CreatedAt,
			"updated_at":           it.Tenant.UpdatedAt,
		})
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{
		"items":     items,
		"page":      page,
		"page_size": pageSize,
		"total":     out.Total,
	}))
}

// @Summary      Update tenant
// @Tags         platform
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  PlatformTenantUpdateRequest  true  "request"
// @Success      200   {object}  resp.APIEnvelope
// @Failure      200   {object}  resp.APIEnvelope
// @Router       /api/v1/platform/tenant/update [post]
func (h *PlatformTenantHandler) Update(c echo.Context) error {
	var req PlatformTenantUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	err := h.UpdateUC.Execute(c.Request().Context(), usecase.UpdatePlatformTenantInput{
		TenantID:     req.TenantID,
		DisplayName:  req.DisplayName,
		Province:     req.Province,
		City:         req.City,
		District:     req.District,
		Address:      req.Address,
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		Remark:       req.Remark,
		Role:         middleware.GetRole(c),
	})
	if err != nil {
		logAudit(c, h.AuditUC, usecase.AuditWriteInput{
			TargetType: "tenant",
			TargetID:   req.TenantID,
			TargetName: req.DisplayName,
			Action:     "update",
			Module:     "platform_tenant",
			Result:     "fail",
			ErrorCode:  err.Error(),
			AfterJSON: map[string]any{
				"display_name":  req.DisplayName,
				"province":      req.Province,
				"city":          req.City,
				"district":      req.District,
				"address":       req.Address,
				"contact_name":  req.ContactName,
				"contact_phone": req.ContactPhone,
				"remark":        req.Remark,
			},
		})
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), mapTenantErrCode(err), err.Error()))
	}
	logAudit(c, h.AuditUC, usecase.AuditWriteInput{
		TargetType: "tenant",
		TargetID:   req.TenantID,
		TargetName: req.DisplayName,
		Action:     "update",
		Module:     "platform_tenant",
		Result:     "success",
		AfterJSON: map[string]any{
			"display_name":  req.DisplayName,
			"province":      req.Province,
			"city":          req.City,
			"district":      req.District,
			"address":       req.Address,
			"contact_name":  req.ContactName,
			"contact_phone": req.ContactPhone,
			"remark":        req.Remark,
		},
		ChangedFields: []string{"display_name", "province", "city", "district", "address", "contact_name", "contact_phone", "remark"},
	})
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{"status": "ok"}))
}

// @Summary      Toggle tenant status
// @Tags         platform
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  PlatformTenantToggleStatusRequest  true  "request"
// @Success      200   {object}  resp.APIEnvelope
// @Failure      200   {object}  resp.APIEnvelope
// @Router       /api/v1/platform/tenant/toggle-status [post]
func (h *PlatformTenantHandler) ToggleStatus(c echo.Context) error {
	var req PlatformTenantToggleStatusRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	err := h.ToggleUC.Execute(c.Request().Context(), usecase.TogglePlatformTenantStatusInput{
		TenantID: req.TenantID,
		Status:   req.Status,
		Role:     middleware.GetRole(c),
	})
	if err != nil {
		logAudit(c, h.AuditUC, usecase.AuditWriteInput{
			TargetType: "tenant",
			TargetID:   req.TenantID,
			TargetName: h.auditTenantName(c, req.TenantID),
			Action:     "toggle_status",
			Module:     "platform_tenant",
			Result:     "fail",
			ErrorCode:  err.Error(),
			AfterJSON: map[string]any{
				"status": req.Status,
			},
		})
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), mapTenantErrCode(err), err.Error()))
	}
	logAudit(c, h.AuditUC, usecase.AuditWriteInput{
		TargetType: "tenant",
		TargetID:   req.TenantID,
		TargetName: h.auditTenantName(c, req.TenantID),
		Action:     "toggle_status",
		Module:     "platform_tenant",
		Result:     "success",
		AfterJSON: map[string]any{
			"status": req.Status,
		},
		ChangedFields: []string{"status"},
	})
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{"status": req.Status}))
}

// @Summary      Reset tenant admin auth
// @Tags         platform
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  PlatformTenantAdminResetAuthRequest  true  "request"
// @Success      200   {object}  resp.APIEnvelope
// @Failure      200   {object}  resp.APIEnvelope
// @Router       /api/v1/platform/tenant/admin/reset-auth [post]
func (h *PlatformTenantHandler) ResetAdminAuth(c echo.Context) error {
	var req PlatformTenantAdminResetAuthRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	err := h.ResetAuthUC.Execute(c.Request().Context(), usecase.ResetPlatformTenantAdminAuthInput{
		TenantID: req.TenantID,
		Action:   req.Action,
		Role:     middleware.GetRole(c),
	})
	if err != nil {
		logAudit(c, h.AuditUC, usecase.AuditWriteInput{
			TargetType: "tenant",
			TargetID:   req.TenantID,
			TargetName: h.auditTenantName(c, req.TenantID),
			Action:     "reset_auth",
			Module:     "platform_tenant",
			Result:     "fail",
			ErrorCode:  err.Error(),
			AfterJSON: map[string]any{
				"action": req.Action,
			},
		})
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), mapTenantErrCode(err), err.Error()))
	}
	logAudit(c, h.AuditUC, usecase.AuditWriteInput{
		TargetType: "tenant",
		TargetID:   req.TenantID,
		TargetName: h.auditTenantName(c, req.TenantID),
		Action:     "reset_auth",
		Module:     "platform_tenant",
		Result:     "success",
		AfterJSON: map[string]any{
			"action": req.Action,
		},
		ChangedFields: []string{"auth_state"},
	})
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{"status": "sent"}))
}

// @Summary      Change tenant admin identity
// @Tags         platform
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  PlatformTenantChangeAdminRequest  true  "request"
// @Success      200   {object}  resp.APIEnvelope
// @Failure      200   {object}  resp.APIEnvelope
// @Router       /api/v1/platform/tenant/change-admin [post]
func (h *PlatformTenantHandler) ChangeAdmin(c echo.Context) error {
	var req PlatformTenantChangeAdminRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	out, err := h.ChangeAdminUC.Execute(c.Request().Context(), usecase.ChangePlatformTenantAdminInput{
		TenantID:     req.TenantID,
		AdminAccount: req.AdminAccount,
		AdminName:    req.AdminName,
		AdminPhone:   req.AdminPhone,
		Role:         middleware.GetRole(c),
	})
	if err != nil {
		logAudit(c, h.AuditUC, usecase.AuditWriteInput{
			TargetType: "tenant",
			TargetID:   req.TenantID,
			TargetName: h.auditTenantName(c, req.TenantID),
			Action:     "change_admin",
			Module:     "platform_tenant",
			Result:     "fail",
			ErrorCode:  err.Error(),
		})
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), mapTenantErrCode(err), err.Error()))
	}
	logAudit(c, h.AuditUC, usecase.AuditWriteInput{
		TargetType: "tenant",
		TargetID:   req.TenantID,
		TargetName: h.auditTenantName(c, req.TenantID),
		Action:     "change_admin",
		Module:     "platform_tenant",
		Result:     "success",
		AfterJSON: map[string]any{
			"admin_user_id": out.AdminUserID,
			"admin_account": out.AdminAccount,
			"admin_name":    out.AdminName,
			"admin_phone":   out.AdminPhone,
		},
		ChangedFields: []string{"admin_account", "admin_name", "admin_phone"},
	})
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{
		"admin_user_id": out.AdminUserID,
		"admin_account": out.AdminAccount,
		"admin_name":    out.AdminName,
		"admin_phone":   out.AdminPhone,
	}))
}

// @Summary      Reset tenant admin password
// @Tags         platform
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  PlatformTenantAdminResetPasswordRequest  true  "request"
// @Success      200   {object}  resp.APIEnvelope
// @Failure      200   {object}  resp.APIEnvelope
// @Router       /api/v1/platform/tenant/admin/reset-password [post]
func (h *PlatformTenantHandler) ResetAdminPassword(c echo.Context) error {
	var req PlatformTenantAdminResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	out, err := h.ResetPasswordUC.Execute(c.Request().Context(), usecase.ResetPlatformTenantAdminPasswordInput{
		TenantID:    req.TenantID,
		AdminUserID: req.AdminUserID,
		Role:        middleware.GetRole(c),
	})
	if err != nil {
		targetID := req.AdminUserID
		if targetID == "" {
			targetID = req.TenantID
		}
		logAudit(c, h.AuditUC, usecase.AuditWriteInput{
			TargetType: "user",
			TargetID:   targetID,
			Action:     "reset_password",
			Module:     "platform_tenant",
			Result:     "fail",
			ErrorCode:  err.Error(),
		})
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), mapTenantErrCode(err), err.Error()))
	}
	logAudit(c, h.AuditUC, usecase.AuditWriteInput{
		TargetType: "user",
		TargetID:   out.AdminUserID,
		Action:     "reset_password",
		Module:     "platform_tenant",
		Result:     "success",
		ChangedFields: []string{
			"password_hash",
			"must_change_password",
			"password_updated_at",
		},
	})
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{
		"admin_user_id":        out.AdminUserID,
		"temporary_password":   out.TemporaryPassword,
		"must_change_password": out.MustChangePassword,
	}))
}

// @Summary      Check tenant display name availability
// @Tags         platform
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  PlatformTenantCheckDisplayNameRequest  true  "request"
// @Success      200   {object}  resp.APIEnvelope
// @Failure      200   {object}  resp.APIEnvelope
// @Router       /api/v1/platform/tenant/check-display-name [post]
func (h *PlatformTenantHandler) CheckDisplayName(c echo.Context) error {
	var req PlatformTenantCheckDisplayNameRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	out, err := h.CheckDisplayNameUC.Execute(c.Request().Context(), usecase.CheckPlatformTenantDisplayNameInput{
		DisplayName: req.DisplayName,
		Role:        middleware.GetRole(c),
	})
	if err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), mapTenantErrCode(err), err.Error()))
	}
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{
		"available": out.Available,
		"reason":    out.Reason,
	}))
}

// @Summary      Check tenant admin account availability
// @Tags         platform
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  PlatformTenantCheckAdminAccountRequest  true  "request"
// @Success      200   {object}  resp.APIEnvelope
// @Failure      200   {object}  resp.APIEnvelope
// @Router       /api/v1/platform/tenant/check-admin-account [post]
func (h *PlatformTenantHandler) CheckAdminAccount(c echo.Context) error {
	var req PlatformTenantCheckAdminAccountRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	out, err := h.CheckAdminAccountUC.Execute(c.Request().Context(), usecase.CheckPlatformTenantAdminAccountInput{
		AdminAccount: req.AdminAccount,
		Role:         middleware.GetRole(c),
	})
	if err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), mapTenantErrCode(err), err.Error()))
	}
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{
		"available": out.Available,
		"reason":    out.Reason,
	}))
}

// @Summary      Check admin phone availability
// @Tags         platform
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  PlatformTenantCheckAdminPhoneRequest  true  "request"
// @Success      200   {object}  resp.APIEnvelope
// @Failure      200   {object}  resp.APIEnvelope
// @Router       /api/v1/platform/tenant/check-admin-phone [post]
func (h *PlatformTenantHandler) CheckAdminPhone(c echo.Context) error {
	var req PlatformTenantCheckAdminPhoneRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	out, err := h.CheckAdminPhoneUC.Execute(c.Request().Context(), usecase.CheckPlatformTenantAdminPhoneInput{
		AdminPhone: req.AdminPhone,
		Role:       middleware.GetRole(c),
	})
	if err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), mapTenantErrCode(err), err.Error()))
	}
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{
		"available": out.Available,
		"reason":    out.Reason,
	}))
}

func mapTenantErrCode(err error) int {
	if err == domainErr.ErrForbidden {
		return resp.CodeForbidden
	}
	if err == domainErr.ErrValidation ||
		err == domainErr.ErrTenantNotFound ||
		err == domainErr.ErrTenantDisplayNameExists ||
		err == domainErr.ErrTenantAdminAccountExists ||
		err == domainErr.ErrTenantAdminPhoneExists ||
		err == domainErr.ErrInvalidTenantStatus ||
		err == domainErr.ErrInvalidPhone ||
		err == domainErr.ErrInvalidResetAction ||
		err == domainErr.ErrTenantAdminNotFound ||
		err == domainErr.ErrMissingTenantAdminTarget ||
		err == domainErr.ErrNotFound {
		return resp.CodeValidation
	}
	return resp.CodeServerError
}

func (h *PlatformTenantHandler) auditTenantName(c echo.Context, tenantID string) string {
	if h.ListUC == nil || strings.TrimSpace(tenantID) == "" {
		return ""
	}
	page := 1
	for {
		out, err := h.ListUC.Execute(c.Request().Context(), usecase.ListPlatformTenantsInput{
			Role:      middleware.GetRole(c),
			NeedTotal: true,
			Page:      page,
			PageSize:  200,
		})
		if err != nil || out == nil {
			return ""
		}
		for _, it := range out.Items {
			if it != nil && strings.TrimSpace(it.Tenant.ID) == strings.TrimSpace(tenantID) {
				return it.Tenant.DisplayName
			}
		}
		if len(out.Items) == 0 || page*200 >= out.Total {
			break
		}
		page++
	}
	return ""
}
