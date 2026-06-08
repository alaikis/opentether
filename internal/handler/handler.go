package handler

import (
	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/service"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Handler struct {
	services *service.Services
	config   *config.Config
	db       *gorm.DB
}

func NewHandlers(services *service.Services, cfg *config.Config, db *gorm.DB) *Handler {
	return &Handler{
		services: services,
		config:   cfg,
		db:       db,
	}
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
		"name":   "OpenTether",
	})
}

// SetupStatus 检查系统初始化状态
func (h *Handler) SetupStatus(c *fiber.Ctx) error {
	// 创建临时的 SetupService 来检查状态
	setupSvc := service.NewSetupService(h.db, h.config)
	initialized, err := setupSvc.IsInitialized()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"initialized": initialized,
	})
}

// Setup 执行系统初始化
func (h *Handler) Setup(c *fiber.Ctx) error {
	var req service.SetupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request",
		})
	}

	// 创建临时的 SetupService 来执行初始化
	setupSvc := service.NewSetupService(h.db, h.config)
	result, err := setupSvc.Setup(&req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(result)
}

// AuthLogin handles login requests
func (h *Handler) AuthLogin(c *fiber.Ctx) error {
	type LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	token, err := h.services.Auth.Login(req.Username, req.Password)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"token": token})
}

// AuthRefresh handles token refresh requests
func (h *Handler) AuthRefresh(c *fiber.Ctx) error {
	type RefreshRequest struct {
		Token string `json:"token"`
	}

	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	token, err := h.services.Auth.RefreshToken(req.Token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"token": token})
}

// User handlers
func (h *Handler) ListUsers(c *fiber.Ctx) error {
	users, err := h.services.User.List()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(users)
}

func (h *Handler) CreateUser(c *fiber.Ctx) error {
	var user service.CreateUserInput
	if err := c.BodyParser(&user); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.User.Create(user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) UpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")
	var user service.UpdateUserInput
	if err := c.BodyParser(&user); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.User.Update(id, user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.services.User.Delete(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) BatchCreateUsers(c *fiber.Ctx) error {
	var users []service.CreateUserInput
	if err := c.BodyParser(&users); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.User.BatchCreate(users)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// UserGroup handlers
func (h *Handler) ListUserGroups(c *fiber.Ctx) error {
	groups, err := h.services.UserGroup.List()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(groups)
}

func (h *Handler) CreateUserGroup(c *fiber.Ctx) error {
	var group service.CreateUserGroupInput
	if err := c.BodyParser(&group); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.UserGroup.Create(group)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) UpdateUserGroup(c *fiber.Ctx) error {
	id := c.Params("id")
	var group service.UpdateUserGroupInput
	if err := c.BodyParser(&group); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.UserGroup.Update(id, group)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) DeleteUserGroup(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.services.UserGroup.Delete(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) AddGroupMember(c *fiber.Ctx) error {
	id := c.Params("id")
	type MemberInput struct {
		UserIDs []string `json:"user_ids"`
	}
	var input MemberInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if err := h.services.UserGroup.AddMembers(id, input.UserIDs); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

// Provider handlers
func (h *Handler) ListProviders(c *fiber.Ctx) error {
	providers, err := h.services.Provider.List()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(providers)
}

func (h *Handler) CreateProvider(c *fiber.Ctx) error {
	var provider service.CreateProviderInput
	if err := c.BodyParser(&provider); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.Provider.Create(provider)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) UpdateProvider(c *fiber.Ctx) error {
	id := c.Params("id")
	var provider service.UpdateProviderInput
	if err := c.BodyParser(&provider); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.Provider.Update(id, provider)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) DeleteProvider(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.services.Provider.Delete(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) TestProvider(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := h.services.Provider.Test(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// DataSource handlers
func (h *Handler) ListDataSources(c *fiber.Ctx) error {
	ds, err := h.services.DataSource.List()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(ds)
}

func (h *Handler) CreateDataSource(c *fiber.Ctx) error {
	var ds service.CreateDataSourceInput
	if err := c.BodyParser(&ds); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.DataSource.Create(ds)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) UpdateDataSource(c *fiber.Ctx) error {
	id := c.Params("id")
	var ds service.UpdateDataSourceInput
	if err := c.BodyParser(&ds); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.DataSource.Update(id, ds)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) DeleteDataSource(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.services.DataSource.Delete(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) TestDataSource(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := h.services.DataSource.Test(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) AnalyzeDataSource(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := h.services.DataSource.Analyze(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// Skill handlers
func (h *Handler) ListSkills(c *fiber.Ctx) error {
	skills, err := h.services.Skill.List()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(skills)
}

func (h *Handler) CreateSkill(c *fiber.Ctx) error {
	var skill service.CreateSkillInput
	if err := c.BodyParser(&skill); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.Skill.Create(skill)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) UpdateSkill(c *fiber.Ctx) error {
	id := c.Params("id")
	var skill service.UpdateSkillInput
	if err := c.BodyParser(&skill); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.Skill.Update(id, skill)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) DeleteSkill(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.services.Skill.Delete(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) TestSkill(c *fiber.Ctx) error {
	id := c.Params("id")
	type TestInput struct {
		Input string `json:"input"`
	}
	var input TestInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.Skill.Test(id, input.Input)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) SyncSkillVector(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.services.Skill.SyncVector(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

// Task handlers
func (h *Handler) ListTasks(c *fiber.Ctx) error {
	tasks, err := h.services.Task.List()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(tasks)
}

func (h *Handler) CreateTask(c *fiber.Ctx) error {
	var task service.CreateTaskInput
	if err := c.BodyParser(&task); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.Task.Create(task)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) UpdateTask(c *fiber.Ctx) error {
	id := c.Params("id")
	var task service.UpdateTaskInput
	if err := c.BodyParser(&task); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.Task.Update(id, task)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) DeleteTask(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.services.Task.Delete(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) RunTask(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := h.services.Task.Run(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) GetTaskLogs(c *fiber.Ctx) error {
	id := c.Params("id")
	logs, err := h.services.Task.GetLogs(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(logs)
}

// IM handlers
func (h *Handler) ListIMConfigs(c *fiber.Ctx) error {
	cfgs, err := h.services.IM.ListConfigs()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(cfgs)
}

func (h *Handler) CreateIMConfig(c *fiber.Ctx) error {
	var cfg service.CreateIMConfigInput
	if err := c.BodyParser(&cfg); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.IM.CreateConfig(cfg)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) UpdateIMConfig(c *fiber.Ctx) error {
	id := c.Params("id")
	var cfg service.UpdateIMConfigInput
	if err := c.BodyParser(&cfg); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.IM.UpdateConfig(id, cfg)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) DeleteIMConfig(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.services.IM.DeleteConfig(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) TestIMConfig(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := h.services.IM.TestConfig(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) ListIMPairings(c *fiber.Ctx) error {
	pairings, err := h.services.IM.ListPairings()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(pairings)
}

func (h *Handler) UnbindIM(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.services.IM.Unbind(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) WeComCallback(c *fiber.Ctx) error {
	return h.handleIMCallback(c, "wecom")
}

func (h *Handler) FeishuCallback(c *fiber.Ctx) error {
	return h.handleIMCallback(c, "feishu")
}

func (h *Handler) DingTalkCallback(c *fiber.Ctx) error {
	return h.handleIMCallback(c, "dingtalk")
}

func (h *Handler) WhatsAppCallback(c *fiber.Ctx) error {
	return h.handleIMCallback(c, "whatsapp")
}

func (h *Handler) handleIMCallback(c *fiber.Ctx, platform string) error {
	body := c.Body()
	result, err := h.services.IM.HandleCallback(platform, body)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// Log handlers
func (h *Handler) ListAuditLogs(c *fiber.Ctx) error {
	logs, err := h.services.Log.ListAudit(c.Query("page"), c.Query("limit"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(logs)
}

func (h *Handler) ListRequestLogs(c *fiber.Ctx) error {
	logs, err := h.services.Log.ListRequest(c.Query("page"), c.Query("limit"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(logs)
}

func (h *Handler) ExportLogs(c *fiber.Ctx) error {
	format := c.Query("format", "json")
	data, err := h.services.Log.Export(c.Query("type", "audit"), format)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	if format == "csv" {
		c.Set("Content-Type", "text/csv")
		c.Set("Content-Disposition", "attachment; filename=logs.csv")
		return c.Send(data)
	}

	return c.JSON(data)
}

// User chat handlers
func (h *Handler) Chat(c *fiber.Ctx) error {
	type ChatInput struct {
		Message        string `json:"message"`
		ConversationID string `json:"conversation_id"`
	}

	var input ChatInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	// Get user ID from context (set by auth middleware)
	userID := c.Locals("user_id").(string)

	result, err := h.services.Agent.Chat(userID, input.Message, input.ConversationID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func (h *Handler) ChatStream(c *fiber.Ctx) error {
	type ChatInput struct {
		Message        string `json:"message"`
		ConversationID string `json:"conversation_id"`
	}

	var input ChatInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	// Get user ID from context (set by auth middleware)
	userID := c.Locals("user_id").(string)

	// Get streaming response channel
	stream, err := h.services.Agent.ChatStream(userID, input.Message, input.ConversationID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	// Stream the response
	for chunk := range stream {
		// Write SSE format: "data: <message>\n\n"
		if _, err := c.Write([]byte("data: " + chunk + "\n\n")); err != nil {
			break
		}
	}

	// Send final message to indicate completion
	c.Write([]byte("data: [DONE]\n\n"))

	return nil
}

func (h *Handler) ListConversations(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	convs, err := h.services.Conversation.List(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(convs)
}

func (h *Handler) GetConversation(c *fiber.Ctx) error {
	id := c.Params("id")
	conv, err := h.services.Conversation.Get(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(conv)
}
