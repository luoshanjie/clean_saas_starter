package bootstrap

import (
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"service/internal/delivery/http/handler"
	"service/internal/delivery/http/middleware"
	storagerepo "service/internal/repo/storage"
)

func TestNewBootstrapHandlers_FileHandlerOptional(t *testing.T) {
	now := func() time.Time { return time.Unix(0, 0) }

	withoutOSS := newBootstrapHandlers(&bootstrapRepos{}, nil, func() string { return "id" }, now, "secret")
	if withoutOSS.fileHandler != nil {
		t.Fatalf("expected file handler to be nil when OSS is disabled")
	}

	withOSS := newBootstrapHandlers(&bootstrapRepos{
		objectStorage: &storagerepo.MockObjectStorage{Now: now},
	}, nil, func() string { return "id" }, now, "secret")
	if withOSS.fileHandler == nil {
		t.Fatalf("expected file handler to be wired when OSS is enabled")
	}
}

func TestRegisterRoutes_FileRoutesConditional(t *testing.T) {
	t.Run("without_oss", func(t *testing.T) {
		e := echo.New()
		registerRoutes(e, minimalBootstrapHandlers(nil), &bootstrapRepos{}, nil)

		if hasRoute(e, "/api/v1/file/upload/session/create") {
			t.Fatalf("expected upload session route to be absent when OSS is disabled")
		}
		if hasRoute(e, "/api/v1/file/download/presign") {
			t.Fatalf("expected download presign route to be absent when OSS is disabled")
		}
	})

	t.Run("with_oss", func(t *testing.T) {
		e := echo.New()
		registerRoutes(e, minimalBootstrapHandlers(&handler.FileHandler{}), &bootstrapRepos{}, nil)

		if !hasRoute(e, "/api/v1/file/upload/session/create") {
			t.Fatalf("expected upload session route to be present when OSS is enabled")
		}
		if !hasRoute(e, "/api/v1/file/download/presign") {
			t.Fatalf("expected download presign route to be present when OSS is enabled")
		}
	})
}

func minimalBootstrapHandlers(fileHandler *handler.FileHandler) *bootstrapHandlers {
	return &bootstrapHandlers{
		authHandler: &handler.AuthHandler{
			JWT: middleware.JWTMiddleware{Secret: []byte("secret")},
		},
		platformTenantHandler: &handler.PlatformTenantHandler{},
		fileHandler:           fileHandler,
		healthHandler:         &handler.HealthHandler{},
	}
}

func hasRoute(e *echo.Echo, path string) bool {
	for _, route := range e.Routes() {
		if route.Path == path {
			return true
		}
	}
	return false
}
