package router

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/database"
	"github.com/alaikis/opentether/internal/handler"
	"github.com/alaikis/opentether/internal/middleware"
	"github.com/alaikis/opentether/internal/service"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// findStaticRoot 查找静态文件的根目录
func findStaticRoot() string {
	candidates := []string{
		"./admin-ui/build",
		"../admin-ui/build",
	}
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		candidates = append(candidates,
			filepath.Join(execDir, "admin-ui", "build"),
			filepath.Join(execDir, "..", "admin-ui", "build"),
		)
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil {
			absPath, _ := filepath.Abs(candidate)
			return absPath
		}
	}
	return "./admin-ui/build"
}

// Setup 注册所有路由
// uiFS: 前端静态文件系统，嵌入模式为 http.FileSystem，文件系统模式传 nil
func Setup(app *fiber.App, h *handler.Handler, cfg *config.Config, uiFS http.FileSystem, db *gorm.DB, useEmbedded bool) {
	// === 根路径重定向 ===
	app.Get("/", func(c *fiber.Ctx) error {
		currentDB := db
		if currentDB == nil {
			if _, err := os.Stat("data/opentether.db"); err == nil {
				defaultDBCfg := config.DatabaseConfig{Type: "sqlite", Name: "data/opentether.db"}
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
		if err != nil || !initialized {
			return c.Redirect("/setup")
		}
		return c.Redirect("/admin")
	})

	// === API 路由 ===
	registerAPIRoutes(app, h, cfg.Security.JWT.Secret)

	// === 中间件 ===
	app.Use(middleware.CORS(cfg.Security.CORS))

	// === 前端路由 ===
	if useEmbedded {
		registerEmbeddedFrontend(app, uiFS)
	} else {
		registerFilesystemFrontend(app)
	}
}

// registerEmbeddedFrontend 注册嵌入模式前端路由
func registerEmbeddedFrontend(app *fiber.App, uiFS http.FileSystem) {
	// SPA fallback: 非文件路径返回 index.html
	app.Use("/admin", func(c *fiber.Ctx) error {
		// c.Path() = "/admin/xxx" → 去掉前缀得到 "xxx"
		// c.Path() = "/admin" → 返回 ""
		filePath := strings.TrimPrefix(c.Path(), "/admin")
		filePath = strings.TrimPrefix(filePath, "/")
		if filePath == "" {
			filePath = "index.html"
		}

		f, err := uiFS.Open(filePath)
		if err != nil {
			// SPA fallback
			index, err := uiFS.Open("index.html")
			if err != nil {
				return c.Status(404).SendString("Not Found")
			}
			defer index.Close()
			stat, _ := index.Stat()
			c.Set("Content-Type", "text/html")
			return c.SendStream(index, int(stat.Size()))
		}
		defer f.Close()
		stat, _ := f.Stat()
		return c.SendStream(f, int(stat.Size()))
	})

	// /docs 和 /im 重定向到 SPA 对应路由
	app.Get("/docs", func(c *fiber.Ctx) error { return c.Redirect("/admin/docs") })
	app.Get("/docs/*", func(c *fiber.Ctx) error { return c.Redirect("/admin/docs") })
	app.Get("/im", func(c *fiber.Ctx) error { return c.Redirect("/admin/im") })
	app.Get("/im/*", func(c *fiber.Ctx) error { return c.Redirect("/admin/im") })

	// /setup 初始化向导
	app.Get("/setup", func(c *fiber.Ctx) error {
		f, err := uiFS.Open("setup/index.html")
		if err != nil {
			return c.Status(404).SendString("Setup page not found")
		}
		defer f.Close()
		stat, _ := f.Stat()
		c.Set("Content-Type", "text/html")
		return c.SendStream(f, int(stat.Size()))
	})
	app.Get("/setup/*", func(c *fiber.Ctx) error {
		return c.Redirect("/setup")
	})
}

// registerFilesystemFrontend 注册文件系统模式前端路由（开发用）
func registerFilesystemFrontend(app *fiber.App) {
	staticRoot := findStaticRoot()
	log.Printf("静态文件根目录: %s", staticRoot)

	// SPA
	app.Static("/admin", staticRoot)
	app.Get("/admin/*", func(c *fiber.Ctx) error {
		return c.SendFile(filepath.Join(staticRoot, "index.html"))
	})

	// Setup 向导
	setupRoot := filepath.Join(staticRoot, "setup")
	app.Static("/setup", setupRoot)
	app.Get("/setup", func(c *fiber.Ctx) error {
		return c.SendFile(filepath.Join(setupRoot, "index.html"))
	})
	app.Get("/setup/*", func(c *fiber.Ctx) error {
		return c.SendFile(filepath.Join(setupRoot, "index.html"))
	})

	// 重定向
	app.Get("/docs", func(c *fiber.Ctx) error { return c.Redirect("/admin/docs") })
	app.Get("/docs/*", func(c *fiber.Ctx) error { return c.Redirect("/admin/docs") })
	app.Get("/im", func(c *fiber.Ctx) error { return c.Redirect("/admin/im") })
	app.Get("/im/*", func(c *fiber.Ctx) error { return c.Redirect("/admin/im") })
}
