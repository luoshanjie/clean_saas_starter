package bootstrap

import (
	"context"
	"errors"
	"time"

	"github.com/labstack/echo/v4"

	"service/internal/app/usecase"
	"service/pkg/logger"
)

type App struct {
	Echo    *echo.Echo
	cleanup func()
}

func (a *App) Close() {
	if a == nil || a.cleanup == nil {
		return
	}
	a.cleanup()
}

// 手动依赖注入在 composition root 里显式组装各层依赖。
// wire 已归档为只读状态，当前项目使用手动组装依赖。
func Build(ctx context.Context, e *echo.Echo, db *DBRuntime, idGen func() string, now func() time.Time, cfg Config, appLogger logger.Logger) (*App, error) {
	if db == nil {
		return nil, errors.New("nil database runtime")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if idGen == nil {
		return nil, errors.New("nil id generator")
	}
	if cfg.JWTSecret == "" {
		return nil, errors.New("empty jwt secret")
	}
	if now == nil {
		now = time.Now
	}

	repos, err := newBootstrapRepos(db, now, cfg.OSS, cfg.JWTSecret)
	if err != nil {
		return nil, err
	}
	permChecker, err := newPermissionChecker()
	if err != nil {
		return nil, err
	}
	handlers := newBootstrapHandlers(repos, permChecker, idGen, now, cfg.JWTSecret, cfg.Auth)
	registerRoutes(e, handlers, repos, permChecker)

	var cleanupUC *usecase.CleanupExpiredUploadSessionsUsecase
	if repos.objectStorage != nil {
		cleanupUC = &usecase.CleanupExpiredUploadSessionsUsecase{
			Storage: repos.objectStorage,
			Repo:    repos.fileUploadSessionRepo,
			Now:     now,
		}
	}
	stopCleanupJob := startUploadCleanupJob(ctx, appLogger, cleanupUC, cfg.UploadCleanup)

	return &App{
		Echo:    e,
		cleanup: stopCleanupJob,
	}, nil
}
