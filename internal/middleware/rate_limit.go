package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/alaikis/opentether/internal/config"
	"github.com/gofiber/fiber/v2"
)

type rateLimitBucket struct {
	count       int
	windowStart time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateLimitBucket
	now     func() time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{buckets: make(map[string]*rateLimitBucket), now: time.Now}
}

func RateLimit(cfg config.RateLimitConfig) fiber.Handler {
	return NewRateLimiter().Middleware(cfg)
}

func (l *RateLimiter) Middleware(cfg config.RateLimitConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !cfg.Enabled || !isRateLimitedPath(c.Path()) {
			return c.Next()
		}
		limit := cfg.RequestsPerMinute
		if limit <= 0 {
			limit = 1
		}
		if !l.allow(rateLimitKey(c), limit) {
			c.Set("Retry-After", "60")
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":               "rate_limited",
				"requests_per_minute": limit,
				"retry_after_seconds": 60,
				"rate_limit_identity": rateLimitKey(c),
			})
		}
		return c.Next()
	}
}

func (l *RateLimiter) allow(key string, limit int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	bucket := l.buckets[key]
	if bucket == nil || now.Sub(bucket.windowStart) >= time.Minute {
		l.buckets[key] = &rateLimitBucket{count: 1, windowStart: now}
		return true
	}
	if bucket.count >= limit {
		return false
	}
	bucket.count++
	return true
}

func isRateLimitedPath(path string) bool {
	return len(path) >= 7 && path[:7] == "/api/v1"
}

func rateLimitKey(c *fiber.Ctx) string {
	if apiKeyID, ok := c.Locals("api_key_id").(string); ok && apiKeyID != "" {
		return "api_key:" + apiKeyID
	}
	if userID, ok := c.Locals("user_id").(string); ok && userID != "" {
		return "user:" + userID
	}
	return fmt.Sprintf("ip:%s", c.IP())
}
