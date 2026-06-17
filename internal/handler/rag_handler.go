package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

type RAGIngestInput struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Source  string   `json:"source"`
	Tags    []string `json:"tags"`
}

func (h *Handler) RAGIngest(c *fiber.Ctx) error {
	var input RAGIngestInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if strings.TrimSpace(input.Content) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "content 不能为空"})
	}
	doc, err := h.services.RAG.Ingest(input.Title, input.Source, input.Content, input.Tags, 800)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(doc)
}

func (h *Handler) RAGListDocuments(c *fiber.Ctx) error {
	docs, err := h.services.RAG.ListDocuments()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(docs)
}

func (h *Handler) RAGDeleteDocument(c *fiber.Ctx) error {
	if err := h.services.RAG.DeleteDocument(c.Params("id")); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) RAGRetrieve(c *fiber.Ctx) error {
	query := c.Query("q")
	if strings.TrimSpace(query) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "q 参数不能为空"})
	}
	chunks, err := h.services.RAG.Retrieve(query, 5)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(chunks)
}
