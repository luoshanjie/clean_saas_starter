package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"service/internal/delivery/http/resp"
	"service/internal/domain/port"
)

// RequirePermission enforces a permission before entering the handler.
func RequirePermission(checker port.PermissionChecker, permission string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if checker == nil {
				return c.JSON(http.StatusOK, resp.Error(resp.CodeForbidden, "permission checker missing"))
			}
			ok, err := checker.Enforce(c.Request().Context(), GetRole(c), permission)
			if err != nil {
				return c.JSON(http.StatusOK, resp.Error(resp.CodeForbidden, err.Error()))
			}
			if !ok {
				return c.JSON(http.StatusOK, resp.Error(resp.CodeForbidden, "forbidden"))
			}
			return next(c)
		}
	}
}
