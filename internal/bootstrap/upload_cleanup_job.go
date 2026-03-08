package bootstrap

import (
	"context"
	"sync"
	"time"

	"service/internal/app/usecase"
	"service/pkg/logger"
)

func startUploadCleanupJob(parent context.Context, l logger.Logger, uc *usecase.CleanupExpiredUploadSessionsUsecase, cfg UploadCleanupConfig) func() {
	if l == nil {
		l = logger.NewNopLogger()
	}
	if uc == nil || !cfg.Enabled {
		l.Info("upload_cleanup_job_disabled")
		return func() {}
	}
	interval := cfg.Interval
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 200
	}

	ctx, cancel := context.WithCancel(parent)
	var once sync.Once
	stop := func() {
		once.Do(cancel)
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		l.Info("upload_cleanup_job_started", "interval_sec", int(interval.Seconds()), "batch_size", batchSize)
		runOnce := func() {
			out, err := uc.Execute(ctx, usecase.CleanupExpiredUploadSessionsInput{Limit: batchSize})
			if err != nil {
				l.Error("upload_cleanup_job_failed", "error", err.Error())
				return
			}
			l.Info("upload_cleanup_job_tick",
				"scanned", out.Scanned,
				"cleaned", out.Cleaned,
				"failed", out.Failed,
			)
		}
		runOnce()
		for {
			select {
			case <-ctx.Done():
				l.Info("upload_cleanup_job_stopped")
				return
			case <-ticker.C:
				runOnce()
			}
		}
	}()

	return stop
}
