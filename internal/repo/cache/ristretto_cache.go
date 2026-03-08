package cache

import (
	"context"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

type RistrettoCache struct {
	c *ristretto.Cache[string, []byte]
}

type RistrettoConfig struct {
	NumCounters int64
	MaxCost     int64
	BufferItems int64
}

func NewRistrettoCache(cfg RistrettoConfig) (*RistrettoCache, error) {
	if cfg.NumCounters <= 0 {
		cfg.NumCounters = 10000
	}
	if cfg.MaxCost <= 0 {
		cfg.MaxCost = 64 << 20 // 64MB
	}
	if cfg.BufferItems <= 0 {
		cfg.BufferItems = 64
	}
	c, err := ristretto.NewCache(&ristretto.Config[string, []byte]{
		NumCounters: cfg.NumCounters,
		MaxCost:     cfg.MaxCost,
		BufferItems: cfg.BufferItems,
		Cost: func(v []byte) int64 {
			return int64(len(v))
		},
	})
	if err != nil {
		return nil, err
	}
	return &RistrettoCache{c: c}, nil
}

func (r *RistrettoCache) Get(_ context.Context, key string) ([]byte, bool) {
	if r == nil || r.c == nil {
		return nil, false
	}
	v, ok := r.c.Get(key)
	return v, ok
}

func (r *RistrettoCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	if r == nil || r.c == nil {
		return nil
	}
	r.c.SetWithTTL(key, value, int64(len(value)), ttl)
	return nil
}

func (r *RistrettoCache) Delete(_ context.Context, key string) error {
	if r == nil || r.c == nil {
		return nil
	}
	r.c.Del(key)
	return nil
}
