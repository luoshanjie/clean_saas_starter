package bootstrap

import (
	"github.com/labstack/echo/v4"

	"service/internal/delivery/http/middleware"
	casbinrepo "service/internal/repo/casbin"
)

func registerRoutes(e *echo.Echo, hs *bootstrapHandlers, repos *bootstrapRepos, permChecker *casbinrepo.PermissionChecker) {
	e.GET("/health", hs.healthHandler.Health)
	api := e.Group("/api/v1")
	api.POST("/auth/login", hs.authHandler.Login)
	api.POST("/auth/login/verify", hs.authHandler.LoginVerify)
	api.POST("/auth/refresh", hs.authHandler.Refresh)
	protected := api.Group("", hs.authHandler.JWT.MiddlewareFunc, middleware.AuthContextMiddleware(repos.authRepo))
	protected.POST("/auth/me", hs.authHandler.Me)
	protected.POST("/auth/change-password", hs.authHandler.ChangePassword)
	protected.POST("/auth/profile/update-name", hs.authHandler.UpdateDisplayName)
	protected.POST("/auth/change-phone/challenge", hs.authHandler.ChangePhoneChallenge)
	protected.POST("/auth/change-phone/resend", hs.authHandler.ChangePhoneResend)
	protected.POST("/auth/change-phone/verify", hs.authHandler.ChangePhoneVerify)
	protected.POST("/platform/tenant/create", hs.platformTenantHandler.Create)
	protected.POST("/platform/tenant/list", hs.platformTenantHandler.List)
	protected.POST("/platform/tenant/update", hs.platformTenantHandler.Update)
	protected.POST("/platform/tenant/toggle-status", hs.platformTenantHandler.ToggleStatus)
	protected.POST("/platform/tenant/admin/reset-auth", hs.platformTenantHandler.ResetAdminAuth)
	protected.POST("/platform/tenant/change-admin", hs.platformTenantHandler.ChangeAdmin)
	protected.POST("/platform/tenant/admin/reset-password", hs.platformTenantHandler.ResetAdminPassword)
	protected.POST("/platform/tenant/check-display-name", hs.platformTenantHandler.CheckDisplayName)
	protected.POST("/platform/tenant/check-admin-account", hs.platformTenantHandler.CheckAdminAccount)
	protected.POST("/platform/tenant/check-admin-phone", hs.platformTenantHandler.CheckAdminPhone)
	if hs.fileHandler != nil {
		protected.POST("/file/upload/session/create", hs.fileHandler.UploadSessionCreate)
		protected.POST("/file/upload/confirm", hs.fileHandler.UploadConfirm)
		protected.POST("/file/download/presign", hs.fileHandler.DownloadPresign)
		protected.POST("/file/upload/cleanup-expired", hs.fileHandler.CleanupExpired, middleware.RequirePermission(permChecker, "platform.system.config"))
	}
}
