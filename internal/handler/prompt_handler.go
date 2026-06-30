package handler

import (
	"github.com/alaikis/opentether/internal/agent"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) ListPromptVersions(c *fiber.Ctx) error {
	engine := h.services.Agent.GetEngine()
	if engine == nil {
		return c.Status(500).JSON(fiber.Map{"error": "agent engine not initialized"})
	}
	evolution := engine.GetPromptEvolution()
	if evolution == nil {
		return c.JSON(map[string]map[string]agent.PromptVersion{})
	}
	return c.JSON(evolution.GetAllVersionsGlobal())
}
