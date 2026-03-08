package bootstrap

import (
	"time"

	"service/internal/delivery/http/handler"
	casbinrepo "service/internal/repo/casbin"
)

type bootstrapHandlers struct {
	authHandler           *handler.AuthHandler
	platformTenantHandler *handler.PlatformTenantHandler
	fileHandler           *handler.FileHandler
	healthHandler         *handler.HealthHandler
}

type handlerDeps struct {
	repos     *bootstrapRepos
	perm      *casbinrepo.PermissionChecker
	idGen     func() string
	now       func() time.Time
	jwtSecret string
	auth      AuthConfig
}

func newBootstrapHandlers(
	repos *bootstrapRepos,
	permChecker *casbinrepo.PermissionChecker,
	idGen func() string,
	now func() time.Time,
	jwtSecret string,
	authCfg AuthConfig,
) *bootstrapHandlers {
	d := handlerDeps{
		repos:     repos,
		perm:      permChecker,
		idGen:     idGen,
		now:       now,
		jwtSecret: jwtSecret,
		auth:      authCfg,
	}
	h := &bootstrapHandlers{}
	wireAuthHandlers(h, d)
	wirePlatformHandlers(h, d)
	if repos != nil && repos.objectStorage != nil {
		wireFileHandlers(h, d)
	}
	return h
}
