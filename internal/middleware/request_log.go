package middleware

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RequestLogWriter 请求日志写入接口（避免循环引用）
type RequestLogWriter interface {
	WriteRequestLog(method, path string, status int, latencyMs float64, userID, userName, ip string)
}

// RequestLogger 创建请求日志中间件，捕获 method/path/status/latency 并异步写入
func RequestLogger(writer RequestLogWriter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// 执行请求（认证中间件在此期间设置 c.Locals）
		err := c.Next()

		// 读取认证后的用户信息
		userID := ""
		userName := ""
		if uid, ok := c.Locals("user_id").(string); ok {
			userID = uid
		}
		if name, ok := c.Locals("name").(string); ok {
			userName = name
		}

		// 计算耗时
		elapsed := time.Since(start)
		method := c.Method()
		path := c.Path()
		status := c.Response().StatusCode()
		ip := c.IP()
		latencyMs := float64(elapsed.Microseconds()) / 1000.0

		// 异步写入不阻塞请求
		go writer.WriteRequestLog(method, path, status, latencyMs, userID, userName, ip)

		return err
	}
}

// RequestLogEntry 前端需要的请求日志结构
type RequestLogEntry struct {
	ID        string  `json:"id"`
	Timestamp string  `json:"timestamp"`
	Method    string  `json:"method"`
	Path      string  `json:"path"`
	Status    int     `json:"status"`
	LatencyMs float64 `json:"latency_ms"`
	UserName  string  `json:"user_name"`
	IPAddress string  `json:"ip_address"`
}

// ParseRequestLogs 从 audit_log 的 details JSON 中解析出请求日志
func ParseRequestLogs(rawLogs []map[string]interface{}) []RequestLogEntry {
	result := make([]RequestLogEntry, 0, len(rawLogs))
	for _, raw := range rawLogs {
		entry := RequestLogEntry{
			ID:        fmt.Sprint(raw["id"]),
			UserName:  fmt.Sprint(raw["user_name"]),
			IPAddress: fmt.Sprint(raw["ip_address"]),
		}
		if ts, ok := raw["timestamp"]; ok {
			entry.Timestamp = fmt.Sprint(ts)
		}
		// 从 details JSON 解析 method/path/status/latency
		if details, ok := raw["details"].(string); ok && details != "" {
			var req struct {
				Method    string  `json:"method"`
				Path      string  `json:"path"`
				Status    int     `json:"status"`
				LatencyMs float64 `json:"latency_ms"`
			}
			if json.Unmarshal([]byte(details), &req) == nil {
				entry.Method = req.Method
				entry.Path = req.Path
				entry.Status = req.Status
				entry.LatencyMs = req.LatencyMs
			}
		}
		result = append(result, entry)
	}
	return result
}
