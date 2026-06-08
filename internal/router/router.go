package router

import (
	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/handler"
	"github.com/alaikis/opentether/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App, h *handler.Handler, cfg *config.Config, _ interface{}) {
	// API routes
	api := app.Group("/api/v1")

	// Health check
	app.Get("/health", h.HealthCheck)

	// Public routes
	public := api.Group("")
	public.Post("/auth/login", h.AuthLogin)
	public.Post("/auth/refresh", h.AuthRefresh)

	// Admin routes (require authentication)
	admin := api.Group("/admin", middleware.Auth(cfg.Security.JWT.Secret))
	admin.Get("/users", h.ListUsers)
	admin.Post("/users", h.CreateUser)
	admin.Put("/users/:id", h.UpdateUser)
	admin.Delete("/users/:id", h.DeleteUser)
	admin.Post("/users/batch", h.BatchCreateUsers)

	admin.Get("/groups", h.ListUserGroups)
	admin.Post("/groups", h.CreateUserGroup)
	admin.Put("/groups/:id", h.UpdateUserGroup)
	admin.Delete("/groups/:id", h.DeleteUserGroup)
	admin.Post("/groups/:id/members", h.AddGroupMember)

	admin.Get("/providers", h.ListProviders)
	admin.Post("/providers", h.CreateProvider)
	admin.Put("/providers/:id", h.UpdateProvider)
	admin.Delete("/providers/:id", h.DeleteProvider)
	admin.Post("/providers/:id/test", h.TestProvider)

	admin.Get("/datasources", h.ListDataSources)
	admin.Post("/datasources", h.CreateDataSource)
	admin.Put("/datasources/:id", h.UpdateDataSource)
	admin.Delete("/datasources/:id", h.DeleteDataSource)
	admin.Post("/datasources/:id/test", h.TestDataSource)
	admin.Post("/datasources/:id/analyze", h.AnalyzeDataSource)

	admin.Get("/skills", h.ListSkills)
	admin.Post("/skills", h.CreateSkill)
	admin.Put("/skills/:id", h.UpdateSkill)
	admin.Delete("/skills/:id", h.DeleteSkill)
	admin.Post("/skills/:id/test", h.TestSkill)
	admin.Post("/skills/:id/sync", h.SyncSkillVector)

	admin.Get("/tasks", h.ListTasks)
	admin.Post("/tasks", h.CreateTask)
	admin.Put("/tasks/:id", h.UpdateTask)
	admin.Delete("/tasks/:id", h.DeleteTask)
	admin.Post("/tasks/:id/run", h.RunTask)
	admin.Get("/tasks/:id/logs", h.GetTaskLogs)

	admin.Get("/im/configs", h.ListIMConfigs)
	admin.Post("/im/configs", h.CreateIMConfig)
	admin.Put("/im/configs/:id", h.UpdateIMConfig)
	admin.Delete("/im/configs/:id", h.DeleteIMConfig)
	admin.Post("/im/configs/:id/test", h.TestIMConfig)
	admin.Get("/im/pairings", h.ListIMPairings)
	admin.Delete("/im/pairings/:id", h.UnbindIM)

	admin.Get("/logs/audit", h.ListAuditLogs)
	admin.Get("/logs/request", h.ListRequestLogs)
	admin.Get("/logs/export", h.ExportLogs)

	// User routes (require authentication)
	user := api.Group("/user", middleware.Auth(cfg.Security.JWT.Secret))
	user.Post("/chat", h.Chat)
	user.Post("/chat/stream", h.ChatStream)
	user.Get("/conversations", h.ListConversations)
	user.Get("/conversations/:id", h.GetConversation)

	// IM callback routes
	imCallback := api.Group("/im/callback")
	imCallback.Post("/wecom", h.WeComCallback)
	imCallback.Post("/feishu", h.FeishuCallback)
	imCallback.Post("/dingtalk", h.DingTalkCallback)
	imCallback.Post("/whatsapp", h.WhatsAppCallback)

	// Setup routes (public, no auth required)
	// 用于系统首次初始化
	setup := api.Group("/setup")
	setup.Get("/status", h.SetupStatus)
	setup.Post("", h.Setup)

	// Static files - Admin UI
	// Serve from local filesystem
	app.Static("/admin", "./admin-ui/build")

	// Catch-all for SPA routing (Admin UI)
	app.Get("/admin/*", func(c *fiber.Ctx) error {
		return c.SendFile("./admin-ui/build/index.html")
	})

	// Static files - Setup Wizard
	app.Static("/setup", "./admin-ui/build/setup")

	// Catch-all for Setup - 如果系统未初始化，跳转到安装向导
	app.Get("/setup", func(c *fiber.Ctx) error {
		return c.SendFile("./admin-ui/build/setup/index.html")
	})
	app.Get("/setup/*", func(c *fiber.Ctx) error {
		return c.SendFile("./admin-ui/build/setup/index.html")
	})

	// Static files - Docs
	app.Static("/docs", "./admin-ui/build/docs")

	// Catch-all for Docs SPA routing
	app.Get("/docs/*", func(c *fiber.Ctx) error {
		return c.SendFile("./admin-ui/build/docs/index.html")
	})
}
