package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// Auth creates JWT authentication middleware
func Auth(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{"error": "missing authorization header"})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(401).JSON(fiber.Map{"error": "invalid authorization header format"})
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(401, "invalid signing method")
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "invalid or expired token"})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token claims"})
		}

		c.Locals("user_id", claims["user_id"])
		c.Locals("global_user_id", claims["global_user_id"])
		c.Locals("name", claims["name"])
		c.Locals("permissions", claims["permissions"])

		role, _ := claims["role"].(string)
		if role == "" {
			role = "user"
		}
		c.Locals("role", role)
		c.Locals("authenticated", true)

		return c.Next()
	}
}

// OptionalAuth creates optional JWT authentication middleware
func OptionalAuth(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Next()
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err == nil && token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				c.Locals("user_id", claims["user_id"])
				c.Locals("global_user_id", claims["global_user_id"])
				c.Locals("role", claims["role"])
			}
		}

		return c.Next()
	}
}

// RequireAdmin creates middleware that requires admin role
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(string)
		if !ok || role != "admin" {
			return c.Status(403).JSON(fiber.Map{
				"error":     "forbidden",
				"message":   "此操作需要管理员权限",
				"your_role": role,
			})
		}
		return c.Next()
	}
}

// RequireRole creates middleware that requires specific role
func RequireRole(roles ...string) fiber.Handler {
	roleMap := make(map[string]bool)
	for _, role := range roles {
		roleMap[role] = true
	}

	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(string)
		if !ok || !roleMap[role] {
			return c.Status(403).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "您没有权限执行此操作",
			})
		}
		return c.Next()
	}
}

// GetUserID retrieves user ID from context
func GetUserID(c *fiber.Ctx) string {
	if id, ok := c.Locals("user_id").(string); ok {
		return id
	}
	return ""
}

// GetUserRole retrieves user role from context
func GetUserRole(c *fiber.Ctx) string {
	if role, ok := c.Locals("role").(string); ok {
		return role
	}
	return ""
}

// GetUserName retrieves user name from context
func GetUserName(c *fiber.Ctx) string {
	if name, ok := c.Locals("name").(string); ok {
		return name
	}
	return ""
}

// IsAdmin checks if the current user is admin
func IsAdmin(c *fiber.Ctx) bool {
	role, _ := c.Locals("role").(string)
	return role == "admin"
}
