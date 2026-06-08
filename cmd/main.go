// Package main provides the OpenTether Enterprise AI Agent entry point
// This version serves admin UI from the filesystem (admin-ui/build must exist)
package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/database"
	"github.com/alaikis/opentether/internal/handler"
	"github.com/alaikis/opentether/internal/middleware"
	"github.com/alaikis/opentether/internal/router"
	"github.com/alaikis/opentether/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// adminUI is not embedded in cmd build - uses filesystem instead
// The router will automatically use filesystem mode when adminUI is empty
var adminUI embed.FS

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

	// Initialize services
	services := service.NewServices(db, cfg)

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
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// CORS
	app.Use(middleware.CORS(cfg.Security.CORS))

	// Setup routes (adminUI is empty, router will use filesystem mode)
	router.Setup(app, handlers, cfg, adminUI, db)

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
	}()

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting OpenTether server on %s", addr)
	log.Printf("Admin UI available at /admin (served from filesystem)")

	if err := app.Listen(addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
