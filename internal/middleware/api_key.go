package middleware

import (
	"strings"

	"github.com/alaikis/opentether/internal/models"
	"github.com/gofiber/fiber/v2"
)

type ApiKeyValidator interface {
	Validate(rawKey string) (*models.ApiKey, error)
}

// ApiKeyAuth 创建 API Key 认证中间件
// 支持 X-API-Key header，用于外部系统（OA/ERP）集成
// 如果请求同时携带 Bearer Token，优先使用 Bearer Token
func ApiKeyAuth(apiKeyService ApiKeyValidator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 如果已经有 Bearer Token，跳过
		authHeader := c.Get("Authorization")
		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			return c.Next()
		}

		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			return c.Next()
		}

		keyRecord, err := apiKeyService.Validate(apiKey)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"error":   "invalid_api_key",
				"message": err.Error(),
			})
		}

		// 设置用户上下文
		c.Locals("user_id", keyRecord.UserID)
		c.Locals("auth_method", "api_key")
		c.Locals("api_key_id", keyRecord.ID)

		if keyRecord.User != nil {
			c.Locals("name", keyRecord.User.Name)
			c.Locals("role", keyRecord.User.Role)
			c.Locals("global_user_id", keyRecord.User.GlobalUserID)
			c.Locals("permissions", keyRecord.User.Permissions)
		}

		// 检查权限范围 (scopes)
		c.Locals("scopes", keyRecord.Scopes)

		return c.Next()
	}
}

// RequireApiKey 要求请求必须通过 API Key 认证（用于外部集成专用接口）
func RequireApiKey() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authMethod, ok := c.Locals("auth_method").(string)
		if !ok || authMethod != "api_key" {
			return c.Status(401).JSON(fiber.Map{
				"error":   "api_key_required",
				"message": "此接口需要使用 X-API-Key 认证",
			})
		}
		return c.Next()
	}
}

func RequireApiKeyScope(requiredScope string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authMethod, ok := c.Locals("auth_method").(string)
		if !ok || authMethod != "api_key" {
			return c.Status(401).JSON(fiber.Map{
				"error":   "api_key_required",
				"message": "此接口需要使用 X-API-Key 认证",
			})
		}

		scopes, _ := c.Locals("scopes").(string)
		if !hasApiKeyScope(scopes, requiredScope) {
			c.Locals("missing_scope", requiredScope)
			return c.Status(403).JSON(fiber.Map{
				"error":          "insufficient_scope",
				"required_scope": requiredScope,
				"message":        "API Key 权限范围不足",
			})
		}
		return c.Next()
	}
}

func hasApiKeyScope(scopes string, requiredScope string) bool {
	if requiredScope == "" {
		return true
	}
	for _, scope := range strings.Split(scopes, ",") {
		scope = strings.TrimSpace(scope)
		if scope == "*" || scope == requiredScope {
			return true
		}
	}
	return false
}
