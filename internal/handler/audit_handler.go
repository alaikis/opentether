package handler

import (
	"time"

	"github.com/alaikis/opentether/internal/audit"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) QueryAuditLogs(c *fiber.Ctx) error {
	startStr := c.Query("start", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	endStr := c.Query("end", time.Now().Format(time.RFC3339))
	start, _ := time.Parse(time.RFC3339, startStr)
	end, _ := time.Parse(time.RFC3339, endStr)
	filters := map[string]interface{}{
		"actor_id": c.Query("actor_id", ""),
		"operation": c.Query("operation", ""),
		"resource": c.Query("resource", ""),
	}
	entries, err := h.services.Audit.Query(filters, start, end)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(entries)
}

func (h *Handler) ExportAuditLogs(c *fiber.Ctx) error {
	var req struct {
		Format string `json:"format"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	format := audit.ExportFormat(req.Format)
	if format == "" {
		format = audit.ExportFormatJSON
	}
	job, err := h.services.Audit.ExportLogs(format)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(job)
}

func (h *Handler) ExportAuditLogsToS3(c *fiber.Ctx) error {
	var req struct {
		Bucket string `json:"bucket"`
		Key    string `json:"key"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Bucket == "" {
		req.Bucket = "audit-logs"
	}
	if req.Key == "" {
		req.Key = "audit_" + time.Now().Format("20060102_150405") + ".json"
	}
	jobID := "s3_" + time.Now().Format("20060102_150405")
	if err := h.services.Audit.ExportToS3(jobID, req.Bucket, req.Key); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{
		"job_id": jobID,
		"bucket": req.Bucket,
		"key":    req.Key,
		"status": "completed",
	})
}

func (h *Handler) ListComplianceReports(c *fiber.Ctx) error {
	reports, err := h.services.Audit.ListReports()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(reports)
}

func (h *Handler) GenerateComplianceReport(c *fiber.Ctx) error {
	var req struct {
		Format string `json:"format"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	format := audit.ExportFormat(req.Format)
	if format == "" {
		format = audit.ExportFormatJSON
	}
	job, err := h.services.Audit.GenerateComplianceReport(format)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(job)
}
