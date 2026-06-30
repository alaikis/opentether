package handler

import (
	"github.com/alaikis/opentether/internal/models"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) EnterpriseSettings(c *fiber.Ctx) error { rows, err := h.services.Enterprise.GetSettings(); if err != nil { return c.Status(500).JSON(fiber.Map{"error":err.Error()}) }; return c.JSON(rows) }
func (h *Handler) SaveEnterpriseSetting(c *fiber.Ctx) error { var row models.SystemSetting; if err:=c.BodyParser(&row); err!=nil { return c.Status(400).JSON(fiber.Map{"error":"invalid request"}) }; if err:=h.services.Enterprise.SaveSetting(&row); err!=nil { return c.Status(500).JSON(fiber.Map{"error":err.Error()}) }; return c.JSON(row) }
func (h *Handler) RequestSkillPublish(c *fiber.Ctx) error { var input struct{ Reason string `json:"reason"` }; _=c.BodyParser(&input); req, err := h.services.Enterprise.RequestSkillPublish(c.Params("id"), input.Reason); if err!=nil { return c.Status(500).JSON(fiber.Map{"error":err.Error()}) }; return c.JSON(req) }
func (h *Handler) ReviewSkillPublish(c *fiber.Ctx) error { var input struct{ Status string `json:"status"`; Comment string `json:"comment"` }; _=c.BodyParser(&input); reviewer,_:=c.Locals("user_id").(string); req, err := h.services.Enterprise.ReviewSkillPublish(c.Params("id"), reviewer, input.Status, input.Comment); if err!=nil { return c.Status(500).JSON(fiber.Map{"error":err.Error()}) }; return c.JSON(req) }
func (h *Handler) ListSkillPublishRequests(c *fiber.Ctx) error { rows, err := h.services.Enterprise.ListSkillPublishRequests(); if err!=nil { return c.Status(500).JSON(fiber.Map{"error":err.Error()}) }; return c.JSON(rows) }
func (h *Handler) CreateBackup(c *fiber.Ctx) error { rec, err := h.services.Enterprise.BackupData(); if err!=nil { return c.Status(500).JSON(fiber.Map{"error":err.Error()}) }; return c.JSON(rec) }
func (h *Handler) ListBackups(c *fiber.Ctx) error { rows, err := h.services.Enterprise.ListBackups(); if err!=nil { return c.Status(500).JSON(fiber.Map{"error":err.Error()}) }; return c.JSON(rows) }
