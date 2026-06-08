package router

import (
	"github.com/alaikis/opentether/internal/handler"
	"github.com/alaikis/opentether/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

// registerAPIRoutes 注册所有 /api/v1 路由
func registerAPIRoutes(app *fiber.App, h *handler.Handler, jwtSecret string) {
	api := app.Group("/api/v1")

	// Health check
	app.Get("/health", h.HealthCheck)

	// === 公开接口 ===
	public := api.Group("")
	public.Post("/auth/login", h.AuthLogin)
	public.Post("/auth/refresh", h.AuthRefresh)

	// 初始化向导
	setup := api.Group("/setup")
	setup.Get("/status", h.SetupStatus)
	setup.Post("", h.Setup)

	// IM 回调（由 IM 平台调用）
	imCallback := api.Group("/im/callback")
	imCallback.Post("/wecom", h.WeComCallback)
	imCallback.Post("/feishu", h.FeishuCallback)
	imCallback.Post("/dingtalk", h.DingTalkCallback)
	imCallback.Post("/whatsapp", h.WhatsAppCallback)
	imCallback.Post("/ilink", h.ILinkCallback)

	// === 需认证接口 ===
	auth := middleware.Auth(jwtSecret)
	adminOnly := middleware.RequireAdmin()

	// 管理员接口
	admin := api.Group("/admin", auth)
	adminAdmin := admin.Group("", adminOnly)

	// 用户管理
	admin.Get("/users", h.ListUsers)
	adminAdmin.Post("/users", h.CreateUser)
	adminAdmin.Put("/users/:id", h.UpdateUser)
	adminAdmin.Delete("/users/:id", h.DeleteUser)
	adminAdmin.Post("/users/batch", h.BatchCreateUsers)

	// 用户组
	admin.Get("/groups", h.ListUserGroups)
	adminAdmin.Post("/groups", h.CreateUserGroup)
	adminAdmin.Put("/groups/:id", h.UpdateUserGroup)
	adminAdmin.Delete("/groups/:id", h.DeleteUserGroup)
	adminAdmin.Post("/groups/:id/members", h.AddGroupMember)

	// LLM 提供商
	admin.Get("/providers", h.ListProviders)
	adminAdmin.Post("/providers", h.CreateProvider)
	adminAdmin.Put("/providers/:id", h.UpdateProvider)
	adminAdmin.Delete("/providers/:id", h.DeleteProvider)
	adminAdmin.Post("/providers/:id/test", h.TestProvider)

	// 数据源
	admin.Get("/datasources", h.ListDataSources)
	adminAdmin.Post("/datasources", h.CreateDataSource)
	adminAdmin.Put("/datasources/:id", h.UpdateDataSource)
	adminAdmin.Delete("/datasources/:id", h.DeleteDataSource)
	adminAdmin.Post("/datasources/:id/test", h.TestDataSource)
	adminAdmin.Post("/datasources/:id/analyze", h.AnalyzeDataSource)

	// Skills
	adminAdmin.Get("/skills", h.ListSkills)
	adminAdmin.Post("/skills", h.CreateSkill)
	adminAdmin.Put("/skills/:id", h.UpdateSkill)
	adminAdmin.Delete("/skills/:id", h.DeleteSkill)
	adminAdmin.Post("/skills/:id/test", h.TestSkill)
	adminAdmin.Post("/skills/:id/sync", h.SyncSkillVector)
	adminAdmin.Post("/skills/from-markdown", h.UploadMarkdownAndCreateSkill)
	adminAdmin.Post("/skills/preview", h.ParseMarkdownPreview)

	// MCP
	admin.Get("/mcp/configs", h.ListMCPConfigs)
	adminAdmin.Post("/mcp/configs", h.CreateMCPConfig)
	adminAdmin.Post("/mcp/configs/:id/start", h.StartMCPServer)
	adminAdmin.Post("/mcp/configs/:id/stop", h.StopMCPServer)
	admin.Get("/mcp/configs/:id/status", h.GetMCPStatus)
	admin.Get("/mcp/configs/:id/tools", h.ListMCPTools)
	adminAdmin.Post("/mcp/configs/:id/call", h.CallMCPTool)

	// 报表
	adminAdmin.Post("/reports/pdf", h.GeneratePDFReport)
	adminAdmin.Post("/reports/employee-pdf", h.GenerateEmployeePDF)
	adminAdmin.Post("/reports/query-pdf", h.QueryToPDF)
	admin.Post("/docs/md2pdf", h.ConvertMarkdownToPDF)
	admin.Post("/docs/md2pdf/template", h.ConvertMarkdownToPDFWithTemplate)

	// 定时任务
	admin.Get("/tasks", h.ListTasks)
	adminAdmin.Post("/tasks", h.CreateTask)
	adminAdmin.Put("/tasks/:id", h.UpdateTask)
	adminAdmin.Delete("/tasks/:id", h.DeleteTask)
	admin.Post("/tasks/:id/run", h.RunTask)
	adminAdmin.Get("/tasks/:id/logs", h.GetTaskLogs)

	// IM 平台管理
	admin.Get("/im/configs", h.ListIMConfigs)
	adminAdmin.Post("/im/configs", h.CreateIMConfig)
	adminAdmin.Put("/im/configs/:id", h.UpdateIMConfig)
	adminAdmin.Delete("/im/configs/:id", h.DeleteIMConfig)
	adminAdmin.Post("/im/configs/:id/test", h.TestIMConfig)
	admin.Get("/im/pairings", h.ListIMPairings)
	adminAdmin.Delete("/im/pairings/:id", h.UnbindIM)

	// 日志
	adminAdmin.Get("/logs/audit", h.ListAuditLogs)
	adminAdmin.Get("/logs/request", h.ListRequestLogs)
	admin.Get("/logs/export", h.ExportLogs)

	// === 用户接口 ===
	user := api.Group("/user", auth)
	user.Post("/chat", h.Chat)
	user.Post("/chat/stream", h.ChatStream)
	user.Get("/conversations", h.ListConversations)
	user.Get("/conversations/:id", h.GetConversation)

	// IM 绑定（员工个人操作）
	user.Get("/im/bindings", h.ListMyIMBindings)
	user.Post("/im/bindings", h.BindIM)
	user.Delete("/im/bindings/:id", h.UnbindIM)
}
