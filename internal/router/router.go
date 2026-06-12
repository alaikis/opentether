package router

import (
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/database"
	"github.com/alaikis/opentether/internal/handler"
	"github.com/alaikis/opentether/internal/middleware"
	"github.com/alaikis/opentether/internal/service"
	"github.com/alaikis/opentether/internal/storage"
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
func Setup(app *fiber.App, h *handler.Handler, cfg *config.Config, uiFS http.FileSystem, db *gorm.DB, useEmbedded bool, store storage.Driver) {
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

	// === 下载端点（本地存储） ===
	if cfg.Storage.Type == "local" {
		rootPath, _ := filepath.Abs(cfg.Storage.Local.Path)
		app.Static("/downloads", rootPath)
		log.Printf("Download endpoint: /downloads -> %s", rootPath)
	}

	// === 前端路由 ===
	if useEmbedded {
		registerEmbeddedFrontend(app, uiFS)
	} else {
		registerFilesystemFrontend(app)
	}
}

// serveEmbeddedFile 从嵌入 FS 输出文件，并按扩展名设置正确 MIME
func serveEmbeddedFile(c *fiber.Ctx, uiFS http.FileSystem, filePath string) error {
	filePath = strings.TrimPrefix(filePath, "/")
	if filePath == "" {
		filePath = "index.html"
	}

	f, err := uiFS.Open(filePath)
	if err == nil {
		defer f.Close()

		stat, statErr := f.Stat()
		if statErr != nil {
			return c.Status(500).SendString("Failed to read embedded file metadata")
		}
		if stat.IsDir() {
			return serveEmbeddedFile(c, uiFS, strings.TrimSuffix(filePath, "/")+"/index.html")
		}
		setEmbeddedContentType(c, filePath)
		return c.SendStream(f, int(stat.Size()))
	}

	if !shouldFallbackToSPA(filePath) {
		return c.Status(404).SendString("Not Found")
	}

	index, err := uiFS.Open("index.html")
	if err != nil {
		return c.Status(404).SendString("Not Found")
	}
	defer index.Close()
	stat, statErr := index.Stat()
	if statErr != nil {
		return c.Status(500).SendString("Failed to read embedded index metadata")
	}
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendStream(index, int(stat.Size()))
}

func shouldFallbackToSPA(filePath string) bool {
	filePath = strings.TrimPrefix(filePath, "/")
	if filePath == "" || filePath == "index.html" {
		return true
	}
	if strings.HasPrefix(filePath, "_app/") {
		return false
	}
	return filepath.Ext(filePath) == ""
}

func setEmbeddedContentType(c *fiber.Ctx, filePath string) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".js" || ext == ".mjs" {
		c.Set("Content-Type", "application/javascript; charset=utf-8")
		return
	}
	if ext == ".wasm" {
		c.Set("Content-Type", "application/wasm")
		return
	}
	if ext == ".html" || ext == ".htm" {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return
	}
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		c.Set("Content-Type", contentType)
	}
}

// registerEmbeddedFrontend 注册嵌入模式前端路由
func registerEmbeddedFrontend(app *fiber.App, uiFS http.FileSystem) {
	// /_app 静态资源在根路径下也需要提供（base 为空时 SvelteKit 引用根路径资源）
	app.Use("/_app", func(c *fiber.Ctx) error {
		return serveEmbeddedFile(c, uiFS, c.Path())
	})

	// SPA fallback: 非文件路径返回 index.html
	app.Use("/admin", func(c *fiber.Ctx) error {
		filePath := strings.TrimPrefix(c.Path(), "/admin")
		return serveEmbeddedFile(c, uiFS, filePath)
	})

	// /docs 和 /im 重定向到 SPA 对应路由
	app.Get("/docs", func(c *fiber.Ctx) error { return c.Redirect("/admin/docs") })
	app.Get("/docs/*", func(c *fiber.Ctx) error { return c.Redirect("/admin/docs") })
	app.Get("/im", func(c *fiber.Ctx) error { return c.Redirect("/admin/im") })
	app.Get("/im/*", func(c *fiber.Ctx) error { return c.Redirect("/admin/im") })

	// /setup 初始化向导
	app.Get("/setup", func(c *fiber.Ctx) error {
		return serveEmbeddedFile(c, uiFS, "setup/index.html")
	})
	app.Get("/setup/*", func(c *fiber.Ctx) error {
		return c.Redirect("/setup")
	})

	// /open/u/login 登录页面 fallback 到前端 index.html
	app.Get("/open/u/login", func(c *fiber.Ctx) error {
		return serveEmbeddedFile(c, uiFS, "index.html")
	})
	// /admin/login 兼容旧链接，重定向到正确的登录页
	app.Get("/admin/login", func(c *fiber.Ctx) error {
		return c.Redirect("/open/u/login")
	})
}

// registerFilesystemFrontend 注册文件系统模式前端路由（开发用）
func registerFilesystemFrontend(app *fiber.App) {
	staticRoot := findStaticRoot()
	log.Printf("静态文件根目录: %s", staticRoot)

	// /_app 静态资源在根路径下也需要提供
	app.Static("/_app", staticRoot+"/_app")

	// SPA
	app.Static("/admin", staticRoot)
	app.Get("/admin/*", func(c *fiber.Ctx) error {
		filePath := strings.TrimPrefix(c.Path(), "/admin")
		if !shouldFallbackToSPA(filePath) {
			return c.Status(404).SendString("Not Found")
		}
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

	// /open/u/login 登录页面
	app.Get("/open/u/login", func(c *fiber.Ctx) error {
		return c.SendFile(filepath.Join(staticRoot, "index.html"))
	})

	// 重定向
	app.Get("/docs", func(c *fiber.Ctx) error { return c.Redirect("/admin/docs") })
	app.Get("/docs/*", func(c *fiber.Ctx) error { return c.Redirect("/admin/docs") })
	app.Get("/im", func(c *fiber.Ctx) error { return c.Redirect("/admin/im") })
	app.Get("/im/*", func(c *fiber.Ctx) error { return c.Redirect("/admin/im") })
}
