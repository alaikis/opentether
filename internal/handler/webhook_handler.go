package handler

import (
	"github.com/alaikis/opentether/internal/models"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) ListWebhookConfigs(c *fiber.Ctx) error {
	rows, err := h.services.Webhook.ListConfigs()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rows)
}

func (h *Handler) SaveWebhookConfig(c *fiber.Ctx) error {
	var row models.WebhookConfig
	if err := c.BodyParser(&row); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if id := c.Params("id"); id != "" {
		row.ID = id
	}
	if err := h.services.Webhook.SaveConfig(&row); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(row)
}

func (h *Handler) DeleteWebhookConfig(c *fiber.Ctx) error {
	if err := h.services.Webhook.DeleteConfig(c.Params("id")); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}
