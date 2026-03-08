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
}

func newBootstrapHandlers(
	repos *bootstrapRepos,
	permChecker *casbinrepo.PermissionChecker,
	idGen func() string,
	now func() time.Time,
	jwtSecret string,
) *bootstrapHandlers {
	d := handlerDeps{
		repos:     repos,
		perm:      permChecker,
		idGen:     idGen,
		now:       now,
		jwtSecret: jwtSecret,
	}
	h := &bootstrapHandlers{}
	wireAuthHandlers(h, d)
	wirePlatformHandlers(h, d)
	wireFileHandlers(h, d)
	return h
}
