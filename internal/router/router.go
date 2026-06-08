package router

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/database"
	"github.com/alaikis/opentether/internal/handler"
	"github.com/alaikis/opentether/internal/middleware"
	"github.com/alaikis/opentether/internal/service"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// findStaticRoot 查找静态文件的根目录，支持从不同工作目录运行
func findStaticRoot() string {
	// 候选路径列表
	candidates := []string{
		"./admin-ui/build",   // 从项目根目录运行
		"../admin-ui/build",   // 从 output 目录运行
	}

	// 获取可执行文件所在目录
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		candidates = append(candidates,
			filepath.Join(execDir, "admin-ui", "build"),
			filepath.Join(execDir, "..", "admin-ui", "build"),
		)
	}

	// 返回第一个存在的路径
	for _, candidate := range candidates {
		indexPath := filepath.Join(candidate, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			absPath, _ := filepath.Abs(candidate)
			return absPath
		}
	}

	// 默认返回相对于当前工作目录的路径
	return "./admin-ui/build"
}

// Setup - adminUI can be nil for filesystem mode, or populated for embedded mode
func Setup(app *fiber.App, h *handler.Handler, cfg *config.Config, adminUI embed.FS, db *gorm.DB) {
	// 根路径 / - 根据系统初始化状态重定向
	app.Get("/", func(c *fiber.Ctx) error {
		// 尝试获取可用的数据库连接
		currentDB := db
		if currentDB == nil {
			// 主应用 db 为空，尝试用默认 SQLite 配置连接
			defaultDBCfg := config.DatabaseConfig{
				Type: "sqlite",
				Name: "data/opentether.db",
			}
			// 检查数据库文件是否存在
			if _, err := os.Stat(defaultDBCfg.Name); err == nil {
				var initErr error
				currentDB, initErr = database.Initialize(defaultDBCfg)
				if initErr != nil {
					log.Printf("尝试连接默认数据库失败: %v", initErr)
					return c.Redirect("/setup")
				}
			} else {
				return c.Redirect("/setup")
			}
		}

		if currentDB == nil {
			return c.Redirect("/setup")
		}

		setupSvc := service.NewSetupService(currentDB, cfg)
		initialized, err := setupSvc.IsInitialized()
		if err != nil {
			// 如果检查失败，默认跳转到 setup
			return c.Redirect("/setup")
		}
		if !initialized {
			// 系统未初始化，跳转到设置页面
			return c.Redirect("/setup")
		}
		// 系统已初始化，跳转到管理界面的登录页
		return c.Redirect("/admin")
	})

	// API routes
	api := app.Group("/api/v1")

	// Health check
	app.Get("/health", h.HealthCheck)

	// Public routes
	public := api.Group("")
	public.Post("/auth/login", h.AuthLogin)
	public.Post("/auth/refresh", h.AuthRefresh)

	// Admin routes (require authentication + admin role for sensitive operations)
	admin := api.Group("/admin", middleware.Auth(cfg.Security.JWT.Secret))

	// 用户管理 - 需要管理员权限
	adminAdmin := admin.Group("", middleware.RequireAdmin())

	// 用户管理 - 敏感操作需要管理员
	admin.Get("/users", h.ListUsers)
	adminAdmin.Post("/users", h.CreateUser)
	adminAdmin.Put("/users/:id", h.UpdateUser)
	adminAdmin.Delete("/users/:id", h.DeleteUser)
	adminAdmin.Post("/users/batch", h.BatchCreateUsers)

	// 用户组管理 - 需要管理员权限
	admin.Get("/groups", h.ListUserGroups)
	adminAdmin.Post("/groups", h.CreateUserGroup)
	adminAdmin.Put("/groups/:id", h.UpdateUserGroup)
	adminAdmin.Delete("/groups/:id", h.DeleteUserGroup)
	adminAdmin.Post("/groups/:id/members", h.AddGroupMember)

	// LLM Provider 管理 - 需要管理员权限
	admin.Get("/providers", h.ListProviders)
	adminAdmin.Post("/providers", h.CreateProvider)
	adminAdmin.Put("/providers/:id", h.UpdateProvider)
	adminAdmin.Delete("/providers/:id", h.DeleteProvider)
	adminAdmin.Post("/providers/:id/test", h.TestProvider)

	// 数据源管理 - 需要管理员权限
	admin.Get("/datasources", h.ListDataSources)
	adminAdmin.Post("/datasources", h.CreateDataSource)
	adminAdmin.Put("/datasources/:id", h.UpdateDataSource)
	adminAdmin.Delete("/datasources/:id", h.DeleteDataSource)
	adminAdmin.Post("/datasources/:id/test", h.TestDataSource)
	adminAdmin.Post("/datasources/:id/analyze", h.AnalyzeDataSource)

	// Skill 管理
	adminAdmin.Get("/skills", h.ListSkills)
	adminAdmin.Post("/skills", h.CreateSkill)
	adminAdmin.Put("/skills/:id", h.UpdateSkill)
	adminAdmin.Delete("/skills/:id", h.DeleteSkill)
	adminAdmin.Post("/skills/:id/test", h.TestSkill)
	adminAdmin.Post("/skills/:id/sync", h.SyncSkillVector)

	// Skill Markdown - 从 MD 文件创建 Skill
	adminAdmin.Post("/skills/from-markdown", h.UploadMarkdownAndCreateSkill)
	adminAdmin.Post("/skills/preview", h.ParseMarkdownPreview)

	// MCP 管理
	admin.Get("/mcp/configs", h.ListMCPConfigs)
	adminAdmin.Post("/mcp/configs", h.CreateMCPConfig)
	adminAdmin.Post("/mcp/configs/:id/start", h.StartMCPServer)
	adminAdmin.Post("/mcp/configs/:id/stop", h.StopMCPServer)
	admin.Get("/mcp/configs/:id/status", h.GetMCPStatus)
	admin.Get("/mcp/configs/:id/tools", h.ListMCPTools)
	adminAdmin.Post("/mcp/configs/:id/call", h.CallMCPTool)

	// PDF 报表生成
	adminAdmin.Post("/reports/pdf", h.GeneratePDFReport)
	adminAdmin.Post("/reports/employee-pdf", h.GenerateEmployeePDF)
	adminAdmin.Post("/reports/query-pdf", h.QueryToPDF)

	// Markdown 转 PDF
	admin.Post("/docs/md2pdf", h.ConvertMarkdownToPDF)
	admin.Post("/docs/md2pdf/template", h.ConvertMarkdownToPDFWithTemplate)

	// 任务调度 - 需要管理员权限
	admin.Get("/tasks", h.ListTasks)
	adminAdmin.Post("/tasks", h.CreateTask)
	adminAdmin.Put("/tasks/:id", h.UpdateTask)
	adminAdmin.Delete("/tasks/:id", h.DeleteTask)
	admin.Post("/tasks/:id/run", h.RunTask)
	adminAdmin.Get("/tasks/:id/logs", h.GetTaskLogs)

	// IM 配置 - 需要管理员权限
	admin.Get("/im/configs", h.ListIMConfigs)
	adminAdmin.Post("/im/configs", h.CreateIMConfig)
	adminAdmin.Put("/im/configs/:id", h.UpdateIMConfig)
	adminAdmin.Delete("/im/configs/:id", h.DeleteIMConfig)
	adminAdmin.Post("/im/configs/:id/test", h.TestIMConfig)
	admin.Get("/im/pairings", h.ListIMPairings)
	adminAdmin.Delete("/im/pairings/:id", h.UnbindIM)

	// 日志 - 需要管理员权限
	adminAdmin.Get("/logs/audit", h.ListAuditLogs)
	adminAdmin.Get("/logs/request", h.ListRequestLogs)
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
	setup := api.Group("/setup")
	setup.Get("/status", h.SetupStatus)
	setup.Post("", h.Setup)

	// Static files - Admin UI
	// 根据是否提供 embedded FS 决定使用哪种模式
	if adminUI != (embed.FS{}) {
		// 使用嵌入的 FS

		// Admin UI - 从 admin-ui/build 目录
		app.Get("/admin/*", func(c *fiber.Ctx) error {
			path := c.Params("*")
			if path == "" || path == "/" {
				path = "admin-ui/build/index.html"
			} else {
				path = "admin-ui/build/" + path
			}
			file, err := adminUI.Open(path)
			if err != nil {
				// 返回 index.html 用于 SPA 路由
				indexFile, err := adminUI.ReadFile("admin-ui/build/index.html")
				if err != nil {
					return c.Status(404).SendString("Not Found")
				}
				return c.Type("html").Send(indexFile)
			}
			defer file.Close()
			return c.SendFile(path)
		})

		// Admin UI 根路径
		app.Get("/admin", func(c *fiber.Ctx) error {
			indexFile, err := adminUI.ReadFile("admin-ui/build/index.html")
			if err != nil {
				return c.Status(404).SendString("Not Found")
			}
			return c.Type("html").Send(indexFile)
		})

		// Setup Wizard
		app.Get("/setup/*", func(c *fiber.Ctx) error {
			indexFile, err := adminUI.ReadFile("admin-ui/build/setup/index.html")
			if err != nil {
				return c.Status(404).SendString("Not Found")
			}
			return c.Type("html").Send(indexFile)
		})
		app.Get("/setup", func(c *fiber.Ctx) error {
			indexFile, err := adminUI.ReadFile("admin-ui/build/setup/index.html")
			if err != nil {
				return c.Status(404).SendString("Not Found")
			}
			return c.Type("html").Send(indexFile)
		})

		// Docs
		app.Get("/docs/*", func(c *fiber.Ctx) error {
			indexFile, err := adminUI.ReadFile("admin-ui/build/docs/index.html")
			if err != nil {
				return c.Status(404).SendString("Not Found")
			}
			return c.Type("html").Send(indexFile)
		})

		// IM
		app.Get("/im/*", func(c *fiber.Ctx) error {
			path := c.Params("*")
			if path == "" || path == "/" {
				path = "admin-ui/build/im/index.html"
			} else {
				path = "admin-ui/build/im/" + path
			}
			indexFile, err := adminUI.ReadFile(path)
			if err != nil {
				return c.Status(404).SendString("Not Found")
			}
			return c.Type("html").Send(indexFile)
		})
	} else {
		// 使用本地文件系统 (开发模式)
		staticRoot := findStaticRoot()
		log.Printf("静态文件根目录: %s", staticRoot)

		adminRoot := filepath.Join(staticRoot)
		setupRoot := filepath.Join(staticRoot, "setup")
		docsRoot := filepath.Join(staticRoot, "docs")
		imRoot := filepath.Join(staticRoot, "im")

		app.Static("/admin", adminRoot)
		app.Get("/admin/*", func(c *fiber.Ctx) error {
			return c.SendFile(filepath.Join(adminRoot, "index.html"))
		})
		app.Static("/setup", setupRoot)
		app.Get("/setup", func(c *fiber.Ctx) error {
			return c.SendFile(filepath.Join(setupRoot, "index.html"))
		})
		app.Get("/setup/*", func(c *fiber.Ctx) error {
			return c.SendFile(filepath.Join(setupRoot, "index.html"))
		})
		app.Static("/docs", docsRoot)
		app.Get("/docs/*", func(c *fiber.Ctx) error {
			return c.SendFile(filepath.Join(docsRoot, "index.html"))
		})
		app.Static("/im", imRoot)
	}
}
