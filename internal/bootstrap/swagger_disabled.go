//go:build !swagger

package bootstrap

import "github.com/labstack/echo/v4"

// 默认禁用 swagger（生产环境编译不引入第三方依赖）。
func RegisterSwagger(_ *echo.Echo) {}
