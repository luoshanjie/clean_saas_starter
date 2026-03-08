package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	"service/internal/delivery/http/resp"
)

type rateLimitBucket struct {
	count int
	reset time.Time
}

// RateLimitMiddleware provides a simple in-memory fixed window limiter.
// It is intended for M0 baseline, not for multi-instance production use.
func RateLimitMiddleware(max int, window time.Duration) echo.MiddlewareFunc {
	var mu sync.Mutex
	buckets := make(map[string]*rateLimitBucket)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := c.RealIP()
			if key == "" {
				key = "unknown"
			}

			now := time.Now()
			allowed := false

			mu.Lock()
			b, ok := buckets[key]
			if !ok || now.After(b.reset) {
				b = &rateLimitBucket{count: 0, reset: now.Add(window)}
				buckets[key] = b
			}
			if b.count < max {
				b.count++
				allowed = true
			}
			mu.Unlock()

			if !allowed {
				return c.JSON(http.StatusOK, resp.ErrorWithRequestID(GetRequestID(c), resp.CodeRateLimited, "rate limited"))
			}
			return next(c)
		}
	}
}
