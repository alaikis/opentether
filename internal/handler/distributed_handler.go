package handler

import (
	"time"

	"github.com/alaikis/opentether/internal/distributed"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) ListHubNodes(c *fiber.Ctx) error {
	nodes, err := h.services.Distributed.ListNodes()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(nodes)
}

func (h *Handler) RegisterHubNode(c *fiber.Ctx) error {
	var node distributed.NodeInfo
	if err := c.BodyParser(&node); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	node.ID = generateID()
	node.Registered = time.Now()
	node.LastSeen = time.Now()
	if err := h.services.Distributed.RegisterNode(&node); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(node)
}

func (h *Handler) DeregisterHubNode(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.services.Distributed.DeregisterNode(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) ListHubTasks(c *fiber.Ctx) error {
	tasks, err := h.services.Distributed.ListTasks()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(tasks)
}

func (h *Handler) SubmitHubTask(c *fiber.Ctx) error {
	var assignment distributed.TaskAssignment
	if err := c.BodyParser(&assignment); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	assignment.CreatedAt = time.Now()
	if err := h.services.Distributed.SubmitTask(&assignment); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(assignment)
}

func (h *Handler) GetHubTaskResults(c *fiber.Ctx) error {
	taskID := c.Params("id")
	results, err := h.services.Distributed.GetTaskResults(taskID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(results)
}

func (h *Handler) CancelHubTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	if err := h.services.Distributed.CancelTask(taskID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}
