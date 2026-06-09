package middleware

import (
	"github.com/alaikis/opentether/internal/service"
	"github.com/gofiber/fiber/v2"
)

// ApiKeyAuth 创建 API Key 认证中间件
// 支持 X-API-Key header，用于外部系统（OA/ERP）集成
// 如果请求同时携带 Bearer Token，优先使用 Bearer Token
func ApiKeyAuth(apiKeyService *service.ApiKeyService) fiber.Handler {
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
