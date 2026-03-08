package logger

import (
	"fmt"
	"log/slog"
	"os"
)

type SlogLogger struct {
	logger *slog.Logger
}

func NewSlogLogger(cfg Config, category string) Logger {
	// rolling file handler
	fileHandler, err := NewRollingFileHandler(
		cfg.Dir,
		category,
		cfg.Format,
		parseLevel(cfg.Level),
	)
	if err != nil {
		panic(fmt.Sprintf("create rolling handler failed: %v", err))
	}

	handlers := []slog.Handler{fileHandler}

	if cfg.EnableConsole {
		opts := &slog.HandlerOptions{Level: parseLevel(cfg.Level)}
		if cfg.ConsoleFormat == "json" {
			handlers = append(handlers, slog.NewJSONHandler(os.Stdout, opts))
		} else {
			handlers = append(handlers, NewConsoleTextHandler(os.Stdout, parseLevel(cfg.Level)))
		}
	}

	multi := NewMultiHandler(handlers...)
	return &SlogLogger{logger: slog.New(multi)}
}

func (l *SlogLogger) Debug(msg string, args ...any) { l.logger.Debug(msg, args...) }
func (l *SlogLogger) Info(msg string, args ...any)  { l.logger.Info(msg, args...) }
func (l *SlogLogger) Warn(msg string, args ...any)  { l.logger.Warn(msg, args...) }
func (l *SlogLogger) Error(msg string, args ...any) { l.logger.Error(msg, args...) }
func (l *SlogLogger) Errorf(format string, args ...any) {
	l.logger.Error(fmt.Sprintf(format, args...))
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
