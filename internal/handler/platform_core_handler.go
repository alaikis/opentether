package handler

import (
	"github.com/alaikis/opentether/internal/models"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) ValidateSkill(c *fiber.Ctx) error {
	result, err := h.services.PlatformCore.ValidateSkill(c.Params("id"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) StartSkillBootstrap(c *fiber.Ctx) error {
	result, err := h.services.PlatformCore.StartSkillBootstrap(c.Params("id"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) ListEvalCases(c *fiber.Ctx) error {
	rows, err := h.services.PlatformCore.ListEvalCases()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rows)
}

func (h *Handler) SaveEvalCase(c *fiber.Ctx) error {
	var row models.EvalCase
	if err := c.BodyParser(&row); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if id := c.Params("id"); id != "" {
		row.ID = id
	}
	if err := h.services.PlatformCore.SaveEvalCase(&row); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(row)
}

func (h *Handler) DeleteEvalCase(c *fiber.Ctx) error {
	if err := h.services.PlatformCore.DeleteEvalCase(c.Params("id")); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) RunEvalCase(c *fiber.Ctx) error {
	run, err := h.services.PlatformCore.RunEvalCase(c.Params("id"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(run)
}

func (h *Handler) ListEvalRuns(c *fiber.Ctx) error {
	rows, err := h.services.PlatformCore.ListEvalRuns(c.Query("case_id"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rows)
}
