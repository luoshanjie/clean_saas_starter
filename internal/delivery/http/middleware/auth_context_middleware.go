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

// AuthContextMiddleware writes authenticated user, tenant, and scope data into
// standard context.Context for usecases and repository adapters. PostgreSQL
// adapters may use the same context to configure RLS, but RLS is not part of
// this middleware's contract. It also checks token_version so old tokens become
// invalid immediately after password changes.
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
