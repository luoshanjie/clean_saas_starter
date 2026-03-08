package port

import (
	"context"
	"time"
)

// Cache hides concrete cache backend details from business usecases.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
