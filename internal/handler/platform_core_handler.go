package handler

import (
	"strings"

	"github.com/alaikis/opentether/internal/database"
	"github.com/alaikis/opentether/internal/models"
	"github.com/alaikis/opentether/internal/text2sql"
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

func (h *Handler) TestSQLTemplate(c *fiber.Ctx) error {
	var input struct {
		SQLTemplate  string            `json:"sql_template"`
		Variables    map[string]string `json:"variables"`
		DataSourceID string            `json:"data_source_id"`
		Execute      bool              `json:"execute"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	rendered := input.SQLTemplate
	for k, v := range input.Variables {
		rendered = strings.ReplaceAll(rendered, "{{"+k+"}}", "'"+strings.ReplaceAll(v, "'", "''")+"'")
		rendered = strings.ReplaceAll(rendered, "{{ "+k+" }}", "'"+strings.ReplaceAll(v, "'", "''")+"'")
	}
	if strings.Contains(rendered, "{{") {
		return c.Status(400).JSON(fiber.Map{"error": "unresolved variables", "rendered_sql": rendered})
	}
	if !input.Execute {
		return c.JSON(fiber.Map{"rendered_sql": rendered})
	}
	pool := database.NewExternalDBPoolManager(h.db, nil)
	db, err := pool.Get(c.UserContext(), input.DataSourceID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error(), "rendered_sql": rendered})
	}
	t := text2sql.NewWithExternalDB(h.db, nil, input.DataSourceID, db)
	res, err := t.ExecuteSQL(c.UserContext(), &text2sql.QueryRequest{Question: "template test", RawSQL: rendered, DataSourceID: input.DataSourceID})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error(), "rendered_sql": rendered})
	}
	return c.JSON(fiber.Map{"rendered_sql": rendered, "result": res})
}

func (h *Handler) ListPolicies(c *fiber.Ctx) error {
	rows, err := h.services.PlatformCore.ListPolicies()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rows)
}
func (h *Handler) SavePolicy(c *fiber.Ctx) error {
	var row models.AccessPolicy
	if err := c.BodyParser(&row); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if id := c.Params("id"); id != "" {
		row.ID = id
	}
	if err := h.services.PlatformCore.SavePolicy(&row); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(row)
}
func (h *Handler) DeletePolicy(c *fiber.Ctx) error {
	if err := h.services.PlatformCore.DeletePolicy(c.Params("id")); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}
func (h *Handler) EvaluatePolicy(c *fiber.Ctx) error {
	var body map[string]interface{}
	_ = c.BodyParser(&body)
	res, err := h.services.PlatformCore.EvaluatePolicy(c.Query("scope"), c.Query("resource", "*"), body)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}

func (h *Handler) ListPrecomputeJobs(c *fiber.Ctx) error {
	rows, err := h.services.PlatformCore.ListPrecomputeJobs()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rows)
}
func (h *Handler) SavePrecomputeJob(c *fiber.Ctx) error {
	var row models.PrecomputeJob
	if err := c.BodyParser(&row); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if id := c.Params("id"); id != "" {
		row.ID = id
	}
	if err := h.services.PlatformCore.SavePrecomputeJob(&row); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(row)
}
func (h *Handler) DeletePrecomputeJob(c *fiber.Ctx) error {
	if err := h.services.PlatformCore.DeletePrecomputeJob(c.Params("id")); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}
func (h *Handler) RunPrecomputeJob(c *fiber.Ctx) error {
	row, err := h.services.PlatformCore.RunPrecomputeJob(c.Params("id"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(row)
}

func (h *Handler) ListReportTemplates(c *fiber.Ctx) error {
	rows, err := h.services.ReportEngine.ListTemplates(c.UserContext(), c.Query("category"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rows)
}
func (h *Handler) SaveReportTemplate(c *fiber.Ctx) error {
	var row models.ReportTemplate
	if err := c.BodyParser(&row); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if id := c.Params("id"); id != "" {
		row.ID = id
		if err := h.services.ReportEngine.UpdateTemplate(c.UserContext(), &row); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(row)
	}
	if err := h.services.ReportEngine.CreateTemplate(c.UserContext(), &row); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(row)
}
func (h *Handler) DeleteReportTemplate(c *fiber.Ctx) error {
	if err := h.services.ReportEngine.DeleteTemplate(c.UserContext(), c.Params("id")); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
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

func (h *Handler) AgentBacktest(c *fiber.Ctx) error {
	query := c.Query("query", "")
	result := h.services.Agent.Backtest(query)
	return c.JSON(result)
}

func (h *Handler) AgentQualityMetrics(c *fiber.Ctx) error {
	snapshot := h.services.Agent.MetricsSnapshot()
	return c.JSON(snapshot)
}
