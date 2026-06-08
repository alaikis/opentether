package middleware

import (
	"github.com/alaikis/opentether/internal/config"
	"github.com/gofiber/fiber/v2"
)

// CORS creates CORS middleware
func CORS(corsConfig config.CORSConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		origin := c.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range corsConfig.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Set("Access-Control-Allow-Origin", origin)
		}

		c.Set("Access-Control-Allow-Methods", joinStrings(corsConfig.AllowedMethods, ", "))
		c.Set("Access-Control-Allow-Headers", joinStrings(corsConfig.AllowedHeaders, ", "))
		c.Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight
		if c.Method() == "OPTIONS" {
			return c.SendStatus(204)
		}

		return c.Next()
	}
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

// ErrorHandler creates custom error handler middleware
func ErrorHandler() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		// Don't expose internal errors in production
		return c.Status(code).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
}
