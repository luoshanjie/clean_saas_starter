package logger

import "sync"

type LogFactory struct {
	cfg   Config
	mu    sync.Mutex
	cache map[string]Logger
}

func NewLogFactory(cfg Config) Factory {
	return &LogFactory{
		cfg:   cfg,
		cache: make(map[string]Logger),
	}
}

func (f *LogFactory) Get(category string) Logger {
	f.mu.Lock()
	defer f.mu.Unlock()

	if logger, ok := f.cache[category]; ok {
		return logger
	}

	newLogger := NewSlogLogger(f.cfg, category)
	f.cache[category] = newLogger
	return newLogger
}
