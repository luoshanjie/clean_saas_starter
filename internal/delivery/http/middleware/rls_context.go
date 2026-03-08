package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"service/internal/app/usecase"
	"service/internal/delivery/http/resp"
	"service/internal/domain/authctx"
	"service/internal/domain/port"
)

// AuthContextMiddleware 把鉴权信息写入标准 context.Context 里，供 repo 设置 RLS。
// 同时校验 token_version，确保改密后旧 token 立刻失效。
func AuthContextMiddleware(repo port.AuthRepo) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			scope := GetScopeType(c)
			if scope == "" {
				return c.JSON(http.StatusOK, resp.ErrorWithRequestID(GetRequestID(c), resp.CodeUnauthorized, "unauthorized"))
			}
			info := authctx.Info{
				UserID:    GetUserID(c),
				TenantID:  GetTenantID(c),
				ScopeType: scope,
			}
			ctx := authctx.With(c.Request().Context(), info)
			c.SetRequest(c.Request().WithContext(ctx))

			// token_version check (skip when repo is nil)
			if repo != nil {
				tv, err := repo.GetTokenVersionByUserID(ctx, info.UserID)
				if err != nil {
					return c.JSON(http.StatusOK, resp.ErrorWithRequestID(GetRequestID(c), resp.CodeUnauthorized, "unauthorized"))
				}
				if tv != GetTokenVersion(c) {
					return c.JSON(http.StatusOK, resp.ErrorWithRequestID(GetRequestID(c), resp.CodeUnauthorized, "unauthorized"))
				}

				user, err := repo.GetUserByID(ctx, info.UserID)
				if err != nil {
					return c.JSON(http.StatusOK, resp.ErrorWithRequestID(GetRequestID(c), resp.CodeUnauthorized, "unauthorized"))
				}
				if usecase.ShouldForcePasswordChange(user) {
					path := c.Path()
					if path == "" {
						path = c.Request().URL.Path
					}
					if !strings.HasSuffix(path, "/auth/me") && !strings.HasSuffix(path, "/auth/change-password") {
						return c.JSON(http.StatusOK, resp.ErrorWithRequestID(GetRequestID(c), resp.CodeForbidden, "must_change_password"))
					}
				}
			}
			return next(c)
		}
	}
}
