package bootstrap

import (
	"time"

	"github.com/labstack/echo/v4"

	"service/internal/delivery/http/middleware"
	"service/pkg/logger"
)

func NewEcho(l logger.Logger) *echo.Echo {
	e := echo.New()
	// request id + request log
	e.Use(middleware.RecoverMiddleware(l))
	e.Use(middleware.RequestIDMiddleware())
	e.Use(middleware.RequestLogger(l))
	e.Use(middleware.RateLimitMiddleware(100, time.Minute))
	return e
}
