package handler

import (
	"encoding/json"
	"time"

	"github.com/alaikis/opentether/internal/agent"
	"github.com/alaikis/opentether/internal/models"
	"github.com/gofiber/fiber/v2"
)

type PlanMultiTaskInput struct {
	Message string `json:"message"`
}

type ExecuteMultiTaskInput struct {
	PlanID string `json:"plan_id"`
}

func (h *Handler) PlanMultiTask(c *fiber.Ctx) error {
	var input PlanMultiTaskInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if input.Message == "" {
		return c.Status(400).JSON(fiber.Map{"error": "message cannot be empty"})
	}
	userID := c.Locals("user_id").(string)
	plan := h.services.Agent.GetEngine().BuildMultiTaskPlanWithLLM(c.Context(), input.Message)
	if plan == nil {
		return c.JSON(fiber.Map{"plan": nil, "message": "single task, no plan needed"})
	}
	h.services.Agent.GetEngine().GetSkillPlanner().PlanSkills(plan)
	planJSON, _ := json.Marshal(plan.SubTasks)
	planModel := models.MultiTaskPlanModel{
		UserID:     userID,
		Original:   input.Message,
		SubTasks:   string(planJSON),
		TotalSteps: plan.TotalSteps,
		IsTree:     plan.IsTree,
		Status:     "planned",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := h.db.Create(&planModel).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"plan_id":    planModel.ID,
		"plan":       plan,
		"totalSteps": plan.TotalSteps,
	})
}

func (h *Handler) ExecuteMultiTask(c *fiber.Ctx) error {
	var input ExecuteMultiTaskInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if input.PlanID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "plan_id is required"})
	}
	userID := c.Locals("user_id").(string)
	var planModel models.MultiTaskPlanModel
	if err := h.db.Where("id = ? AND user_id = ?", input.PlanID, userID).First(&planModel).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "plan not found"})
	}
	var subTasks []agent.SubTask
	json.Unmarshal([]byte(planModel.SubTasks), &subTasks)
	plan := &agent.MultiTaskPlan{
		Original:   planModel.Original,
		SubTasks:   subTasks,
		TotalSteps: planModel.TotalSteps,
		IsTree:     planModel.IsTree,
	}
	conv, err := h.services.Agent.GetOrCreateConversation(userID, "")
	result, err := h.services.Agent.GetEngine().ExecuteMultiTaskPlan(c.Context(), nil, plan, conv, func(current, total int, label, status string) {})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	execution := models.MultiTaskExecutionModel{
		PlanID:     planModel.ID,
		UserID:     userID,
		Summary:    result.Summary,
		Data:       mapToJSON(result.Data),
		Status:     "completed",
		CreatedAt:  time.Now(),
		FinishedAt: time.Now(),
	}
	h.db.Create(&execution)
	h.db.Model(&planModel).Update("status", "completed")
	return c.JSON(fiber.Map{
		"execution_id": execution.ID,
		"plan_id":      planModel.ID,
		"summary":      result.Summary,
		"data":         result.Data,
	})
}

func (h *Handler) ListMultiTaskPlans(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	limit := c.QueryInt("limit", 50)
	var plans []models.MultiTaskPlanModel
	h.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&plans)
	return c.JSON(plans)
}

func mapToJSON(m map[string]interface{}) string {
	b, _ := json.Marshal(m)
	return string(b)
}
