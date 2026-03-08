package middleware

import "github.com/labstack/echo/v4"

const (
	CtxLogModule    = "log_module"
	CtxLogAction    = "log_action"
	CtxLogTargetID  = "log_target_id"
	CtxLogBizStep   = "log_biz_step"
	CtxLogFailPoint = "log_fail_point"
)

func SetLogModule(c echo.Context, module string)     { c.Set(CtxLogModule, module) }
func SetLogAction(c echo.Context, action string)     { c.Set(CtxLogAction, action) }
func SetLogTargetID(c echo.Context, targetID string) { c.Set(CtxLogTargetID, targetID) }
func SetLogBizStep(c echo.Context, step string)      { c.Set(CtxLogBizStep, step) }
func SetLogFailPoint(c echo.Context, point string)   { c.Set(CtxLogFailPoint, point) }

func GetLogModule(c echo.Context) string {
	v, _ := c.Get(CtxLogModule).(string)
	return v
}

func GetLogAction(c echo.Context) string {
	v, _ := c.Get(CtxLogAction).(string)
	return v
}

func GetLogTargetID(c echo.Context) string {
	v, _ := c.Get(CtxLogTargetID).(string)
	return v
}

func GetLogBizStep(c echo.Context) string {
	v, _ := c.Get(CtxLogBizStep).(string)
	return v
}

func GetLogFailPoint(c echo.Context) string {
	v, _ := c.Get(CtxLogFailPoint).(string)
	return v
}
