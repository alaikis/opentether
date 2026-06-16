package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alaikis/opentether/internal/config"
	"github.com/gofiber/fiber/v2"
)

func TestRateLimitDisabled(t *testing.T) {
	app := fiber.New()
	limiter := NewRateLimiter()
	app.Use(limiter.Middleware(config.RateLimitConfig{Enabled: false, RequestsPerMinute: 1}))
	app.Get("/api/v1/ping", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })

	for i := 0; i < 3; i++ {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	}
}

func TestRateLimitSkipsNonAPIPaths(t *testing.T) {
	app := fiber.New()
	limiter := NewRateLimiter()
	limiter.now = func() time.Time { return time.Unix(1000, 0) }
	app.Use(limiter.Middleware(config.RateLimitConfig{Enabled: true, RequestsPerMinute: 1}))
	app.Get("/admin", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })

	for i := 0; i < 3; i++ {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/admin", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	}
}

func TestRateLimitExceeded(t *testing.T) {
	app := fiber.New()
	limiter := NewRateLimiter()
	limiter.now = func() time.Time { return time.Unix(1000, 0) }
	app.Use(limiter.Middleware(config.RateLimitConfig{Enabled: true, RequestsPerMinute: 1}))
	app.Get("/api/v1/ping", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })

	first, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil))
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	first.Body.Close()
	if first.StatusCode != http.StatusOK {
		t.Fatalf("expected first status 200, got %d", first.StatusCode)
	}

	second, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil))
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	defer second.Body.Close()
	if second.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected second status 429, got %d", second.StatusCode)
	}
}

func TestRateLimitUsesApiKeyIdentity(t *testing.T) {
	app := fiber.New()
	limiter := NewRateLimiter()
	limiter.now = func() time.Time { return time.Unix(1000, 0) }
	app.Use(func(c *fiber.Ctx) error {
		if c.Get("X-Test-Key") != "" {
			c.Locals("api_key_id", c.Get("X-Test-Key"))
		}
		return c.Next()
	})
	app.Use(limiter.Middleware(config.RateLimitConfig{Enabled: true, RequestsPerMinute: 1}))
	app.Get("/api/v1/ping", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })

	requestA := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	requestA.Header.Set("X-Test-Key", "a")
	responseA, err := app.Test(requestA)
	if err != nil {
		t.Fatalf("request a failed: %v", err)
	}
	responseA.Body.Close()

	requestB := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	requestB.Header.Set("X-Test-Key", "b")
	responseB, err := app.Test(requestB)
	if err != nil {
		t.Fatalf("request b failed: %v", err)
	}
	defer responseB.Body.Close()

	if responseB.StatusCode != http.StatusOK {
		t.Fatalf("expected distinct api key identity to be allowed, got %d", responseB.StatusCode)
	}
}
