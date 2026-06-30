package handler

import (
	"time"

	"github.com/alaikis/opentether/internal/observability"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) ListMetricDefinitions(c *fiber.Ctx) error {
	defs, err := h.services.Observability.ListMetrics()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(defs)
}

func (h *Handler) CreateMetricDefinition(c *fiber.Ctx) error {
	var def observability.MetricDefinition
	if err := c.BodyParser(&def); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	def.ID = generateID()
	def.CreatedAt = time.Now()
	if err := h.services.Observability.RegisterMetric(&def); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(def)
}

func (h *Handler) QueryMetrics(c *fiber.Ctx) error {
	metricID := c.Params("id")
	startStr := c.Query("start", time.Now().Add(-1*time.Hour).Format(time.RFC3339))
	endStr := c.Query("end", time.Now().Format(time.RFC3339))
	start, _ := time.Parse(time.RFC3339, startStr)
	end, _ := time.Parse(time.RFC3339, endStr)
	values, err := h.services.Observability.QueryMetric(metricID, start, end)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(values)
}

func (h *Handler) ListAlertRules(c *fiber.Ctx) error {
	rules, err := h.services.Observability.ListAlertRules()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rules)
}

func (h *Handler) CreateAlertRule(c *fiber.Ctx) error {
	var rule observability.AlertRule
	if err := c.BodyParser(&rule); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	rule.ID = generateID()
	rule.CreatedAt = time.Now()
	if err := h.services.Observability.CreateAlertRule(&rule); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rule)
}

func (h *Handler) UpdateAlertRule(c *fiber.Ctx) error {
	id := c.Params("id")
	var rule observability.AlertRule
	if err := c.BodyParser(&rule); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	rule.ID = id
	if err := h.services.Observability.UpdateAlertRule(id, &rule); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rule)
}

func (h *Handler) DeleteAlertRule(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.services.Observability.DeleteAlertRule(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) ListAlertEvents(c *fiber.Ctx) error {
	events, err := h.services.Observability.ListAlertEvents()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(events)
}

func (h *Handler) AckAlert(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.services.Observability.AckAlert(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}
