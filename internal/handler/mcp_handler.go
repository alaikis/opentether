package handler

import (
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) ListMCPServers(c *fiber.Ctx) error {
	servers, err := h.services.MCPRegistry.ListServers()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(servers)
}

func (h *Handler) RegisterMCPServer(c *fiber.Ctx) error {
	var cfg struct {
		ID       string            `json:"id"`
		Name     string            `json:"name"`
		Command  string            `json:"command"`
		Args     []string          `json:"args"`
		URL      string            `json:"url"`
		Headers  map[string]string `json:"headers"`
	}
	if err := c.BodyParser(&cfg); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if err := h.services.MCPRegistry.RegisterServer(cfg.ID, cfg.Name, cfg.Command, cfg.Args, cfg.URL, cfg.Headers); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(cfg)
}
