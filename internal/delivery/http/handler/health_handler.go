package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"service/internal/delivery/http/middleware"
	"service/internal/delivery/http/resp"
)

type HealthHandler struct{}

func (h HealthHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]string{
		"status": "ok",
	}))
}
