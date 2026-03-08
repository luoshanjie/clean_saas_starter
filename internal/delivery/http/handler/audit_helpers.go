package handler

import (
	"service/internal/app/usecase"
	"service/internal/delivery/http/middleware"

	"github.com/labstack/echo/v4"
)

func logAudit(c echo.Context, uc *usecase.AuditWriteUsecase, in usecase.AuditWriteInput) {
	if uc == nil {
		return
	}
	if in.RequestID == "" {
		in.RequestID = middleware.GetRequestID(c)
	}
	if in.OperatorUserID == "" {
		in.OperatorUserID = middleware.GetUserID(c)
	}
	if in.OperatorRole == "" {
		in.OperatorRole = middleware.GetRole(c)
	}
	if in.OperatorTenantID == "" {
		in.OperatorTenantID = middleware.GetTenantID(c)
	}
	if in.OperatorDisplayName == "" {
		in.OperatorDisplayName = middleware.GetUserName(c)
	}
	if in.IP == "" {
		in.IP = c.RealIP()
	}
	if in.UserAgent == "" {
		in.UserAgent = c.Request().UserAgent()
	}
	uc.LogSafe(c.Request().Context(), in)
}
