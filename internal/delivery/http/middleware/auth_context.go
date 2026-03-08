package middleware

import "github.com/labstack/echo/v4"

const (
	CtxUserID       = "user_id"
	CtxUserName     = "user_name"
	CtxTenantID     = "tenant_id"
	CtxRole         = "role"
	CtxScopeType    = "scope_type"
	CtxTokenVersion = "token_version"
)

// 统一封装鉴权上下文字段与读取方法：
// 1) 避免在各处散落硬编码字符串；
// 2) 方便后续改名/扩展字段时集中维护；
// 3) handler/usecase 只关心取值，不关心存储细节。
func GetUserID(c echo.Context) string    { v, _ := c.Get(CtxUserID).(string); return v }
func GetUserName(c echo.Context) string  { v, _ := c.Get(CtxUserName).(string); return v }
func GetTenantID(c echo.Context) string  { v, _ := c.Get(CtxTenantID).(string); return v }
func GetRole(c echo.Context) string      { v, _ := c.Get(CtxRole).(string); return v }
func GetScopeType(c echo.Context) string { v, _ := c.Get(CtxScopeType).(string); return v }
func GetTokenVersion(c echo.Context) int { v, _ := c.Get(CtxTokenVersion).(int); return v }
