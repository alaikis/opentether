package handler

import (
	"time"

	"github.com/alaikis/opentether/internal/tuning"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) ListTuningJobs(c *fiber.Ctx) error {
	jobs, err := h.services.Tuning.ListJobs()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(jobs)
}

func (h *Handler) CreateTuningJob(c *fiber.Ctx) error {
	var job tuning.TuningJob
	if err := c.BodyParser(&job); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	job.ID = generateID()
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
	if err := h.services.Tuning.CreateJob(&job); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(job)
}

func (h *Handler) StartTuningJob(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.services.Tuning.StartJob(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

func (h *Handler) ListTuningIterations(c *fiber.Ctx) error {
	jobID := c.Params("id")
	iterations, err := h.services.Tuning.ListIterations(jobID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(iterations)
}

func (h *Handler) RollbackTuningJob(c *fiber.Ctx) error {
	jobID := c.Params("id")
	iterationStr := c.Query("iteration", "")
	if iterationStr == "" {
		return c.Status(400).JSON(fiber.Map{"error": "iteration required"})
	}
	if err := h.services.Tuning.Rollback(jobID, iterationStr); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

func (h *Handler) GetTuningSuggestions(c *fiber.Ctx) error {
	suggestions, err := h.services.Tuning.GetSuggestions()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(suggestions)
}
