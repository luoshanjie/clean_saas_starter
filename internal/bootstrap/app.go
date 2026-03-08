package bootstrap

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"service/pkg/logger"
)

// Run 负责启动应用：加载配置 -> 初始化依赖 -> 组装路由 -> 启动服务。
func Run(ctx context.Context) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	if cfg.JWTSecret == "" {
		return errors.New("JWT_SECRET is empty")
	}
	logFactory := logger.NewLogFactory(cfg.Log)
	appLogger := logFactory.Get("app")
	if appLogger == nil {
		appLogger = logger.NewNopLogger()
	}

	e := NewEcho(appLogger)
	if !cfg.SkipDB {
		if cfg.DBDriver == DBDriverPostgres && cfg.DBDSN == "" {
			return errors.New("DB_DSN is empty")
		}
		if cfg.DBDriver == DBDriverSQLite && cfg.SQLitePath == "" {
			return errors.New("SQLITE_PATH is empty")
		}
		db, err := InitDB(ctx, cfg)
		if err != nil {
			return err
		}
		defer db.Close()

		app, err := Build(ctx, e, db, newID, time.Now, cfg, appLogger)
		if err != nil {
			return err
		}
		defer app.Close()
	}
	RegisterSwagger(e)

	appLogger.Info("server starting", "addr", cfg.Addr)
	return e.Start(cfg.Addr)
}

func newID() string {
	return uuid.NewString()
}
