//go:build swagger

package bootstrap

import (
	"os"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"

	_ "service/docs/swagger"
)

func RegisterSwagger(e *echo.Echo) {
	if !swaggerEnabled() {
		return
	}
	e.GET("/swagger/*", echoSwagger.WrapHandler)
}

func swaggerEnabled() bool {
	return os.Getenv("SWAGGER") == "1" || os.Getenv("APP_ENV") == "dev"
}
