package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestRequireApiKeyScopeRequiresApiKey(t *testing.T) {
	app := fiber.New()
	app.Get("/external/users", RequireApiKeyScope("external:users:read"), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/external/users", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestRequireApiKeyScopeRejectsMissingScope(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("auth_method", "api_key")
		c.Locals("api_key_id", "key-1")
		c.Locals("scopes", "external:im:bind")
		return c.Next()
	})
	app.Get("/external/users", RequireApiKeyScope("external:users:read"), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/external/users", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestRequireApiKeyScopeAllowsMatchingScope(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("auth_method", "api_key")
		c.Locals("api_key_id", "key-1")
		c.Locals("scopes", "external:users:read")
		return c.Next()
	})
	app.Get("/external/users", RequireApiKeyScope("external:users:read"), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/external/users", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRequireApiKeyScopeAllowsWildcardScope(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("auth_method", "api_key")
		c.Locals("api_key_id", "key-1")
		c.Locals("scopes", "*")
		return c.Next()
	})
	app.Get("/external/users", RequireApiKeyScope("external:users:read"), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/external/users", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
