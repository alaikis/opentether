package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/database"
	"github.com/alaikis/opentether/internal/handler"
	"github.com/alaikis/opentether/internal/middleware"
	"github.com/alaikis/opentether/internal/router"
	"github.com/alaikis/opentether/internal/service"
	"github.com/alaikis/opentether/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

//go:embed all:admin-ui/build
var adminUI embed.FS

func mustAdminUIFileSystem() http.FileSystem {
	subFS, err := fs.Sub(adminUI, "admin-ui/build")
	if err != nil {
		log.Fatalf("embedded admin UI root not found: %v", err)
	}

	requiredFiles := []string{
		"index.html",
		"setup/index.html",
		"_app/version.json",
	}
	for _, file := range requiredFiles {
		if _, err := fs.Stat(subFS, file); err != nil {
			log.Fatalf("embedded admin UI missing required file %q: %v", file, err)
		}
	}

	if ok, err := hasEmbeddedFile(subFS, "_app/immutable/entry", ".js"); err != nil {
		log.Fatalf("failed to inspect embedded admin UI entry assets: %v", err)
	} else if !ok {
		log.Fatalf("embedded admin UI missing SvelteKit entry JavaScript under _app/immutable/entry; ensure //go:embed uses all:admin-ui/build and rebuild the frontend before go build")
	}

	return http.FS(subFS)
}

func hasEmbeddedFile(root fs.FS, dir string, ext string) (bool, error) {
	entries, err := fs.ReadDir(root, dir)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ext) {
			return true, nil
		}
	}
	return false, nil
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Initialize(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Run migrations (only if database is configured)
	if db != nil {
		if err := database.Migrate(db); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
	} else {
		log.Printf("Database not configured, skipping migrations. Use /setup to configure.")
	}

	// Initialize object storage
	store, err := storage.New(storage.Config{
		Type: cfg.Storage.Type,
		Local: storage.LocalConfig{
			Path:    cfg.Storage.Local.Path,
			BaseURL: cfg.Storage.Local.BaseURL,
		},
		S3: storage.S3ConfigRaw{
			Endpoint:  cfg.Storage.S3.Endpoint,
			Region:    cfg.Storage.S3.Region,
			AccessKey: cfg.Storage.S3.AccessKey,
			SecretKey: cfg.Storage.S3.SecretKey,
			Bucket:    cfg.Storage.S3.Bucket,
			UseSSL:    cfg.Storage.S3.UseSSL,
			PathStyle: cfg.Storage.S3.PathStyle,
		},
	})
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	log.Printf("Storage initialized: %s", cfg.Storage.Type)

	// Initialize services
	services := service.NewServices(db, cfg, store)

	// 注入 SQL 审计器到 Agent Service
	services.Agent.SetSQLAuditor(services.SQLAudit)

	log.Printf("Services initialized — Agent Engine ready")

	// Initialize handlers
	handlers := handler.NewHandlers(services, cfg, db)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler:          middleware.ErrorHandler(),
		AppName:               "OpenTether",
		DisableStartupMessage: false,
		ReadTimeout:           60 * time.Second,
		WriteTimeout:          60 * time.Second,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${method} ${path} ${latency}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))
	// Enable compression (Fiber v2)
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed, // Use Levels: LevelDisabled(-1), LevelNoCompression(0), LevelBestSpeed(1), LevelBestCompression(2), LevelDefaultCompression(6)
	}))

	// CORS
	app.Use(middleware.CORS(cfg.Security.CORS))

	// API Key 认证中间件（在 JWT 之前，允许外部系统用 API Key 替代 Bearer Token）
	app.Use(middleware.ApiKeyAuth(services.ApiKey))

	// 请求日志中间件（捕获 method/path/status/latency 写入 audit_log）
	app.Use(middleware.RequestLogger(services.Log))

	// Setup routes (embedded mode - binary contains admin-ui/build)
	router.Setup(app, handlers, cfg, mustAdminUIFileSystem(), db, true, store)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nShutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := app.ShutdownWithContext(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		if services.Agent != nil {
			if err := services.Agent.Close(); err != nil {
				log.Printf("Agent resource shutdown error: %v", err)
			}
		}
		if services.MCP != nil {
			services.MCP.StopAll()
		}
	}()

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting OpenTether server on %s", addr)
	log.Printf("Admin UI available at /admin")

	if err := app.Listen(addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
