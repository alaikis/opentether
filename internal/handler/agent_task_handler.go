package handler

import (
	"bufio"
	"fmt"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) CreateAgentTaskGraph(c *fiber.Ctx) error {
	var input struct {
		Goal           string `json:"goal"`
		ConversationID string `json:"conversation_id"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	userID, _ := c.Locals("user_id").(string)
	graph, err := h.services.AgentTasks.CreateGraph(userID, input.ConversationID, input.Goal)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(graph)
}

func (h *Handler) GetAgentTaskGraph(c *fiber.Ctx) error {
	graph, nodes, outputs, err := h.services.AgentTasks.GetGraph(c.Params("id"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "task graph not found"})
	}
	return c.JSON(fiber.Map{"graph": graph, "nodes": nodes, "outputs": outputs})
}

func (h *Handler) GetAgentTaskGraphVisualization(c *fiber.Ctx) error {
	vis, err := h.services.AgentTasks.GetGraphVisualization(c.Params("id"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "task graph not found"})
	}
	return c.JSON(vis)
}

func (h *Handler) InsertAgentTaskNode(c *fiber.Ctx) error {
	var node models.AgentTaskNode
	if err := c.BodyParser(&node); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	result, err := h.services.AgentTasks.InsertNode(c.Params("id"), &node)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) ReviewAgentTaskNode(c *fiber.Ctx) error {
	var input struct {
		Approved bool   `json:"approved"`
		Comment  string `json:"comment"`
	}
	_ = c.BodyParser(&input)
	node, err := h.services.AgentTasks.ReviewNode(c.Params("node_id"), input.Approved, input.Comment)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "node not found"})
	}
	return c.JSON(node)
}

func (h *Handler) ResumeAgentTaskNodeFromCheckpoint(c *fiber.Ctx) error {
	if err := h.services.AgentTasks.ResumeFromCheckpoint(c.Params("id"), c.Params("node_id")); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"resumed": true})
}

func (h *Handler) RetryAgentTaskNode(c *fiber.Ctx) error {
	node, err := h.services.AgentTasks.RetryNode(c.Params("node_id"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "node not found"})
	}
	return c.JSON(node)
}

func (h *Handler) SkipAgentTaskNode(c *fiber.Ctx) error {
	node, err := h.services.AgentTasks.SkipNode(c.Params("node_id"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "node not found"})
	}
	return c.JSON(node)
}

func (h *Handler) ResumeAgentTaskGraph(c *fiber.Ctx) error {
	if err := h.services.AgentTasks.ResumeGraph(c.Params("id")); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	go h.services.AgentTasks.RunGraph(c.Params("id"))
	return c.JSON(fiber.Map{"resumed": true})
}

func (h *Handler) CancelAgentTaskGraph(c *fiber.Ctx) error {
	if err := h.services.AgentTasks.CancelGraph(c.Params("id")); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"cancelled": true})
}

func (h *Handler) StreamAgentTaskGraph(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	id := c.Params("id")
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		for i := 0; i < 120; i++ {
			graph, nodes, outputs, err := h.services.AgentTasks.GetGraph(id)
			if err != nil {
				return
			}
			fmt.Fprintf(w, "data: {\"status\":\"%s\",\"nodes\":%d,\"outputs\":%d,\"progress\":%d}\n\n", graph.Status, len(nodes), len(outputs), graph.Progress)
			_ = w.Flush()
			if graph.Status == "completed" || graph.Status == "failed" || graph.Status == "cancelled" {
				break
			}
			time.Sleep(1 * time.Second)
		}
	})
	return nil
}

func (h *Handler) GetAgentTaskHistory(c *fiber.Ctx) error {
	id := c.Params("id")
	history, err := h.services.AgentTasks.GetNodeHistory(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "task graph not found"})
	}
	return c.JSON(history)
}
