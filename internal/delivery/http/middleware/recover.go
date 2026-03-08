package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"service/internal/delivery/http/resp"
	"service/pkg/logger"
)

// RecoverMiddleware catches panic and returns a unified 500 response.
func RecoverMiddleware(l logger.Logger) echo.MiddlewareFunc {
	if l == nil {
		l = logger.NewNopLogger()
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					l.Error("panic recovered", "error", r, "request_id", GetRequestID(c))
					_ = c.JSON(http.StatusOK, resp.ErrorWithRequestID(GetRequestID(c), resp.CodeServerError, "server error"))
				}
			}()
			return next(c)
		}
	}
}
