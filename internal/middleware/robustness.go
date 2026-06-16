package middleware

import (
	"log"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
)

func PanicRecovery() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC recovered: %v\n%s", r, string(debug.Stack()))
				_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "internal_error",
					"message": "服务器内部错误，已记录并自动恢复。请重试。",
				})
			}
		}()
		return c.Next()
	}
}

func QuerySizeLimits(maxBodySize int) fiber.Handler {
	if maxBodySize <= 0 {
		maxBodySize = 32 * 1024
	}
	return func(c *fiber.Ctx) error {
		if len(c.Body()) > maxBodySize {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"error":   "request_too_large",
				"message": "请求体超过大小限制",
			})
		}
		return c.Next()
	}
}

func QueryMaxLen(maxLen int) fiber.Handler {
	if maxLen <= 0 {
		maxLen = 5000
	}
	return func(c *fiber.Ctx) error {
		if len(c.Body()) > maxLen {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"error":   "request_too_large",
				"message": "查询内容过长",
			})
		}
		return c.Next()
	}
}
