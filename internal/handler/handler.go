package handler

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/im"
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

// audit logs an operation to the audit log
func (h *Handler) audit(c *fiber.Ctx, action, resourceType, resourceID, details string) {
	userID := ""
	userName := ""
	if id, ok := c.Locals("user_id").(string); ok {
		userID = id
	}
	if name, ok := c.Locals("name").(string); ok {
		userName = name
	}
	ipAddress := c.IP()
	userAgent := c.Get("User-Agent")

	// Fire and forget - don't block the request if audit logging fails
	go h.services.Log.Audit(userID, userName, action, resourceType, resourceID, details, ipAddress, userAgent)
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

	h.audit(c, "create", "user", result.ID, "Created user: "+user.Name)
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
	h.audit(c, "update", "user", id, "Updated user")
	return c.JSON(result)
}

func (h *Handler) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.services.User.Delete(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	h.audit(c, "delete", "user", id, "Deleted user")
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

// BindIM 员工绑定自己的 IM 账号
func (h *Handler) BindIM(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	var input service.BindIMInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	binding, err := h.services.IM.BindUser(userID, input)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(binding)
}

// ListMyIMBindings 员工查看自己的 IM 绑定
func (h *Handler) ListMyIMBindings(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	bindings, err := h.services.IM.ListUserBindings(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(bindings)
}

func (h *Handler) UnbindIM(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.services.IM.Unbind(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) handleIMCallback(c *fiber.Ctx, platform string) error {
	body := c.Body()

	// 解析回调，识别用户
	imCtx, err := h.services.IM.HandleCallback(platform, body)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	// 通过 Agent 处理消息
	result, err := h.services.Agent.Chat(imCtx.UserID, imCtx.Message, "")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// 向 IM 用户发送回复
	if reply, ok := result["message"].(string); ok && reply != "" {
		if sendErr := h.services.IM.SendIMReply(platform, imCtx.ReplyTo, reply); sendErr != nil {
			// 发送失败但 AI 处理成功，仍返回成功
			return c.JSON(fiber.Map{"success": true, "message": reply, "send_error": sendErr.Error()})
		}
	}

	return c.JSON(result)
}

func (h *Handler) WeComCallback(c *fiber.Ctx) error {
	return h.handleIMCallback(c, "wecom")
}

func (h *Handler) PersonalWeChatCallback(c *fiber.Ctx) error {
	return h.handleIMCallback(c, "personal_wechat")
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

func (h *Handler) PersonalWhatsAppCallback(c *fiber.Ctx) error {
	return h.handleIMCallback(c, "whatsapp_personal")
}

func (h *Handler) BusinessWhatsAppCallback(c *fiber.Ctx) error {
	return h.handleIMCallback(c, "whatsapp_business")
}

// ILinkCallback 处理 iLink AI Webhook 回调（同步回复模式）
func (h *Handler) ILinkCallback(c *fiber.Ctx) error {
	body := c.Body()

	// 解析回调，识别用户
	imCtx, err := h.services.IM.HandleCallback("ilink", body)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	// 通过 Agent 处理消息
	result, err := h.services.Agent.Chat(imCtx.UserID, imCtx.Message, "")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// iLink 同步回复：直接在 HTTP Response 中返回消息
	reply, _ := result["message"].(string)
	if reply == "" {
		reply = "已收到您的消息"
	}

	return c.JSON(im.FormatILinkReply(imCtx.ReplyTo, reply))
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

// ============================================
// Skill Markdown Handlers - 从 MD 文件创建 Skill
// ============================================

// UploadMarkdownAndCreateSkill 上传 MD 文件并创建 Skill
func (h *Handler) UploadMarkdownAndCreateSkill(c *fiber.Ctx) error {
	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "请上传 MD 文件"})
	}

	// 检查文件类型
	if len(file.Filename) < 3 || file.Filename[len(file.Filename)-3:] != ".md" {
		return c.Status(400).JSON(fiber.Map{"error": "只支持 MD 文件"})
	}

	// 读取文件内容
	fileContent, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "无法读取文件"})
	}
	defer fileContent.Close()

	buffer := make([]byte, file.Size)
	if _, err := fileContent.Read(buffer); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "无法读取文件内容"})
	}

	markdownContent := string(buffer)

	// 解析 MD 文件
	parsed, err := h.services.SkillMarkdown.ParseMarkdownToSkill(markdownContent, file.Filename)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// 创建 Skill
	skill, err := h.services.SkillMarkdown.CreateSkillFromParsed(parsed)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	h.audit(c, "create", "skill", skill.ID, "从 MD 文件创建 Skill: "+skill.Name)

	return c.JSON(fiber.Map{
		"success":    true,
		"skill_id":   skill.ID,
		"skill_name": skill.Name,
		"parsed":     parsed,
	})
}

// ParseMarkdownPreview 预览解析 MD 文件内容
func (h *Handler) ParseMarkdownPreview(c *fiber.Ctx) error {
	type Request struct {
		Content  string `json:"content"`
		Filename string `json:"filename"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.Content == "" {
		return c.Status(400).JSON(fiber.Map{"error": "请提供 markdown 内容"})
	}

	parsed, err := h.services.SkillMarkdown.ParseMarkdownToSkill(req.Content, req.Filename)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(parsed)
}

// ============================================
// MCP Handlers - MCP 协议集成
// ============================================

// ListMCPConfigs 列出 MCP 配置
func (h *Handler) ListMCPConfigs(c *fiber.Ctx) error {
	configs, err := h.services.MCP.GetConfigs()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(configs)
}

// CreateMCPConfig 创建 MCP 配置
func (h *Handler) CreateMCPConfig(c *fiber.Ctx) error {
	type MCPConfigInput struct {
		Name    string `json:"name"`
		Command string `json:"command"`
		Args    string `json:"args"`
		Env     string `json:"env"`
		Enabled bool   `json:"enabled"`
	}

	var input MCPConfigInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	config := &service.MCPConfig{
		Name:    input.Name,
		Command: input.Command,
		Args:    input.Args,
		Env:     input.Env,
		Enabled: input.Enabled,
	}

	if err := h.services.MCP.SaveToDB(config); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "config": config})
}

// StartMCPServer 启动 MCP 服务器
func (h *Handler) StartMCPServer(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.services.MCP.StartServer(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// 获取服务器状态
	server, _ := h.services.MCP.GetServerStatus(id)

	h.audit(c, "start", "mcp_server", id, "启动 MCP 服务器")

	return c.JSON(fiber.Map{
		"success": true,
		"status":  server,
	})
}

// StopMCPServer 停止 MCP 服务器
func (h *Handler) StopMCPServer(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.services.MCP.StopServer(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	h.audit(c, "stop", "mcp_server", id, "停止 MCP 服务器")

	return c.JSON(fiber.Map{"success": true})
}

// GetMCPStatus 获取 MCP 服务器状态
func (h *Handler) GetMCPStatus(c *fiber.Ctx) error {
	id := c.Params("id")

	server, err := h.services.MCP.GetServerStatus(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(server)
}

// ListMCPTools 列出 MCP 工具
func (h *Handler) ListMCPTools(c *fiber.Ctx) error {
	id := c.Params("id")

	tools, err := h.services.MCP.ListTools(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"server_id": id,
		"tools":     tools,
	})
}

// CallMCPTool 调用 MCP 工具
func (h *Handler) CallMCPTool(c *fiber.Ctx) error {
	type CallToolInput struct {
		ToolName  string                 `json:"tool_name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	var input CallToolInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	serverID := c.Params("id")

	result, err := h.services.MCP.CallTool(serverID, input.ToolName, input.Arguments)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"result":  string(result),
	})
}

// ============================================
// PDF Handlers - PDF 报表生成
// ============================================

// GeneratePDFReport 生成 PDF 报表
func (h *Handler) GeneratePDFReport(c *fiber.Ctx) error {
	type ReportInput struct {
		Title    string          `json:"title"`
		Subtitle string          `json:"subtitle"`
		Columns  []string        `json:"columns"`
		Rows     [][]interface{} `json:"rows"`
		Summary  string          `json:"summary"`
	}

	var input ReportInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	reportData := &service.ReportData{
		Title:    input.Title,
		Subtitle: input.Subtitle,
		Columns:  input.Columns,
		Rows:     input.Rows,
		Summary:  input.Summary,
	}

	pdfBytes, err := h.services.PDF.GenerateReport(reportData)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// 设置响应头
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename=report.pdf")

	return c.Send(pdfBytes)
}

// GenerateEmployeePDF 生成员工 PDF 报表
func (h *Handler) GenerateEmployeePDF(c *fiber.Ctx) error {
	type EmployeeReportInput struct {
		Title   string          `json:"title"`
		Columns []string        `json:"columns"`
		Rows    [][]interface{} `json:"rows"`
	}

	var input EmployeeReportInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	pdfBytes, err := h.services.PDF.GenerateEmployeeReport(input.Title, input.Columns, input.Rows)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename=employee_report.pdf")

	return c.Send(pdfBytes)
}

// QueryToPDF 将查询结果转换为 PDF
func (h *Handler) QueryToPDF(c *fiber.Ctx) error {
	type QueryPDFInput struct {
		Query   string          `json:"query"`
		Columns []string        `json:"columns"`
		Rows    [][]interface{} `json:"rows"`
	}

	var input QueryPDFInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	pdfBytes, err := h.services.PDF.GenerateQueryReport(input.Query, input.Columns, input.Rows)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename=query_result.pdf")

	return c.Send(pdfBytes)
}

// ============================================
// Markdown PDF Handlers - Markdown 转 PDF
// ============================================

// ConvertMarkdownToPDF 将 Markdown 转换为 PDF
func (h *Handler) ConvertMarkdownToPDF(c *fiber.Ctx) error {
	type MD2PDFInput struct {
		Markdown string `json:"markdown"`
		Title    string `json:"title"`
		Author   string `json:"author"`
	}

	var input MD2PDFInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if input.Markdown == "" {
		return c.Status(400).JSON(fiber.Map{"error": "请提供 markdown 内容"})
	}

	var pdfBytes []byte
	var err error

	if input.Title != "" {
		tmpl := service.DefaultTemplate()
		tmpl.Title = input.Title
		tmpl.Author = input.Author
		pdfBytes, err = h.services.MarkdownPDF.ConvertWithTemplate(input.Markdown, tmpl)
	} else {
		pdfBytes, err = h.services.MarkdownPDF.Convert(input.Markdown)
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename=document.pdf")

	return c.Send(pdfBytes)
}

// ConvertMarkdownToPDFWithTemplate 使用模板转换
func (h *Handler) ConvertMarkdownToPDFWithTemplate(c *fiber.Ctx) error {
	type MD2PDFInput struct {
		Markdown string               `json:"markdown"`
		Template *service.PDFTemplate `json:"template"`
	}

	var input MD2PDFInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if input.Markdown == "" {
		return c.Status(400).JSON(fiber.Map{"error": "请提供 markdown 内容"})
	}

	tmpl := input.Template
	if tmpl == nil {
		tmpl = service.DefaultTemplate()
	}

	pdfBytes, err := h.services.MarkdownPDF.ConvertWithTemplate(input.Markdown, tmpl)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename=document.pdf")

	return c.Send(pdfBytes)
}

// ============================================
// ApiKey Handlers - API 密钥管理
// ============================================

// ListApiKeys 列出 API 密钥（管理员可查询所有，或按 user_id 过滤）
func (h *Handler) ListApiKeys(c *fiber.Ctx) error {
	userID := c.Query("user_id")
	keys, err := h.services.ApiKey.List(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(keys)
}

// CreateApiKey 创建新的 API 密钥（返回原始密钥仅此一次）
func (h *Handler) CreateApiKey(c *fiber.Ctx) error {
	type CreateApiKeyInput struct {
		UserID        string   `json:"user_id"`
		Name          string   `json:"name"`
		Scopes        []string `json:"scopes"`
		ExpiresInDays int      `json:"expires_in_days"`
	}
	var input CreateApiKeyInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	userID := input.UserID
	if userID == "" {
		userID = c.Locals("user_id").(string)
	}

	apiKey, rawKey, err := h.services.ApiKey.Create(userID, input.Name, input.Scopes, input.ExpiresInDays)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	h.audit(c, "create", "api_key", apiKey.ID, "创建 API 密钥: "+apiKey.Name)

	return c.JSON(fiber.Map{
		"success": true,
		"api_key": apiKey,
		"raw_key": rawKey,
	})
}

// DeleteApiKey 删除 API 密钥
func (h *Handler) DeleteApiKey(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.services.ApiKey.Delete(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

// RegenerateApiKey 重新生成 API 密钥
func (h *Handler) RegenerateApiKey(c *fiber.Ctx) error {
	id := c.Params("id")
	apiKey, rawKey, err := h.services.ApiKey.Regenerate(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"api_key": apiKey,
		"raw_key": rawKey,
	})
}

// ListMyApiKeys 列出当前用户的 API 密钥
func (h *Handler) ListMyApiKeys(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	keys, err := h.services.ApiKey.List(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(keys)
}

// ============================================
// IM 自助绑定 Handlers - 员工扫码绑定 IM 渠道
// ============================================

// ListIMPlatforms 列出可绑定的 IM 平台（含绑定说明和二维码信息）
func (h *Handler) ListIMPlatforms(c *fiber.Ctx) error {
	platforms, err := h.services.IM.ListAvailablePlatforms()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(platforms)
}

// RequestIMBinding 请求 IM 绑定（生成 token，返回二维码/验证信息）
func (h *Handler) RequestIMBinding(c *fiber.Ctx) error {
	type Request struct {
		ImConfigID string `json:"im_config_id"`
	}
	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	userID := c.Locals("user_id").(string)

	result, err := h.services.IM.GenerateBindingToken(userID, req.ImConfigID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

// ConfirmIMBinding IM 回调确认绑定（由 IM 平台回调，通过 token 匹配）
func (h *Handler) ConfirmIMBinding(c *fiber.Ctx) error {
	type ConfirmRequest struct {
		Token      string `json:"token"`
		ImUserID   string `json:"im_user_id"`
		ImUserName string `json:"im_user_name"`
	}
	var req ConfirmRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	binding, err := h.services.IM.ConfirmBinding(req.Token, req.ImUserID, req.ImUserName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"message":  "IM 绑定成功",
		"platform": binding.ImConfigID,
	})
}

// GetSystemConfig 获取系统配置
func (h *Handler) GetSystemConfig(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"server": fiber.Map{
			"port": h.config.Server.Port,
			"mode": h.config.Server.Mode,
		},
		"database": fiber.Map{
			"type":         h.config.Database.Type,
			"host":         h.config.Database.Host,
			"port":         h.config.Database.Port,
			"name":         h.config.Database.Name,
			"user":         h.config.Database.User,
			"password":     "", // 不返回密码
			"sslmode":      h.config.Database.SSLMode,
			"auto_migrate": h.config.Database.AutoMigrate,
		},
		"security": fiber.Map{
			"jwt": fiber.Map{
				"expire":         h.config.Security.JWT.Expire,
				"refresh_expire": h.config.Security.JWT.RefreshExpire,
				"secret":         "", // 不返回密钥
			},
			"rate_limit": fiber.Map{
				"enabled":             h.config.Security.RateLimit.Enabled,
				"requests_per_minute": h.config.Security.RateLimit.RequestsPerMinute,
			},
			"cors": fiber.Map{
				"allowed_origins": h.config.Security.CORS.AllowedOrigins,
				"allowed_methods": h.config.Security.CORS.AllowedMethods,
				"allowed_headers": h.config.Security.CORS.AllowedHeaders,
			},
			"https": fiber.Map{
				"enabled":   h.config.Security.HTTPS.Enabled,
				"cert_file": h.config.Security.HTTPS.CertFile,
				"key_file":  h.config.Security.HTTPS.KeyFile,
			},
		},
		"embedding": fiber.Map{
			"provider":  h.config.Embedding.Provider,
			"model":     h.config.Embedding.Model,
			"dimension": h.config.Embedding.Dimension,
			"store":     h.config.Embedding.StoreProvider,
		},
		"executor": fiber.Map{
			"mode": h.config.Executor.Mode,
			"embedded": fiber.Map{
				"max_concurrent": h.config.Executor.EmbeddedConfig.MaxConcurrent,
				"timeout":        h.config.Executor.EmbeddedConfig.Timeout,
			},
			"independent": fiber.Map{
				"queue": fiber.Map{
					"type":    h.config.Executor.IndependentConfig.Queue.Type,
					"address": h.config.Executor.IndependentConfig.Queue.Address,
				},
			},
		},
		"update": fiber.Map{
			"enabled":          h.config.Update.Enabled,
			"check_interval":   h.config.Update.CheckInterval,
			"github_repo":      h.config.Update.GithubRepo,
			"auto_backup":      h.config.Update.AutoBackup,
			"require_approval": h.config.Update.RequireApproval,
		},
		"smtp": fiber.Map{
			"enabled":    h.config.SMTP.Enabled,
			"host":       h.config.SMTP.Host,
			"port":       h.config.SMTP.Port,
			"username":   h.config.SMTP.Username,
			"password":   "",
			"encryption": h.config.SMTP.Encryption,
			"from_email": h.config.SMTP.FromEmail,
			"from_name":  h.config.SMTP.FromName,
			"to_email":   h.config.SMTP.ToEmail,
		},
	})
}

// ============================================
// 外部系统集成 - 通过 API Key 代用户操作
// ============================================

// ExternalBindIM 外部系统通过 API Key 代员工绑定 IM
func (h *Handler) ExternalBindIM(c *fiber.Ctx) error {
	type ExternalBindInput struct {
		GlobalUserID string `json:"global_user_id"`
		UserName     string `json:"user_name"`
		ImConfigID   string `json:"im_config_id"`
		ImUserID     string `json:"im_user_id"`
		ImUserName   string `json:"im_user_name"`
	}
	var input ExternalBindInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.services.IM.ExternalBindUser(input.GlobalUserID, input.UserName, input.ImConfigID, input.ImUserID, input.ImUserName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"result":  result,
	})
}

// ExternalListUsers 外部系统查询用户列表
func (h *Handler) ExternalListUsers(c *fiber.Ctx) error {
	users, err := h.services.User.List()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(users)
}

// UpdateSystemConfig 更新系统配置
func (h *Handler) UpdateSystemConfig(c *fiber.Ctx) error {
	type SystemConfigInput struct {
		Server *struct {
			Port int    `json:"port"`
			Mode string `json:"mode"`
		} `json:"server"`
		Database *struct {
			Type        string `json:"type"`
			Host        string `json:"host"`
			Port        int    `json:"port"`
			Name        string `json:"name"`
			User        string `json:"user"`
			Password    string `json:"password"`
			SSLMode     string `json:"sslmode"`
			AutoMigrate *bool  `json:"auto_migrate"`
		} `json:"database"`
		Security *struct {
			JWT *struct {
				Secret        string `json:"secret"`
				Expire        string `json:"expire"`
				RefreshExpire string `json:"refresh_expire"`
			} `json:"jwt"`
			RateLimit *struct {
				Enabled           bool `json:"enabled"`
				RequestsPerMinute int  `json:"requests_per_minute"`
			} `json:"rate_limit"`
			CORS *struct {
				AllowedOrigins []string `json:"allowed_origins"`
				AllowedMethods []string `json:"allowed_methods"`
				AllowedHeaders []string `json:"allowed_headers"`
			} `json:"cors"`
			HTTPS *struct {
				Enabled  bool   `json:"enabled"`
				CertFile string `json:"cert_file"`
				KeyFile  string `json:"key_file"`
			} `json:"https"`
		} `json:"security"`
		Embedding *struct {
			Provider  string `json:"provider"`
			Model     string `json:"model"`
			Dimension int    `json:"dimension"`
			Store     string `json:"store"`
		} `json:"embedding"`
		Executor *struct {
			Mode     string `json:"mode"`
			Embedded *struct {
				MaxConcurrent int    `json:"max_concurrent"`
				Timeout       string `json:"timeout"`
			} `json:"embedded"`
			Independent *struct {
				Queue *struct {
					Type    string `json:"type"`
					Address string `json:"address"`
				} `json:"queue"`
			} `json:"independent"`
		} `json:"executor"`
		Update *struct {
			Enabled         bool   `json:"enabled"`
			CheckInterval   string `json:"check_interval"`
			GithubRepo      string `json:"github_repo"`
			AutoBackup      bool   `json:"auto_backup"`
			RequireApproval bool   `json:"require_approval"`
		} `json:"update"`
		SMTP *struct {
			Enabled    bool   `json:"enabled"`
			Host       string `json:"host"`
			Port       int    `json:"port"`
			Username   string `json:"username"`
			Password   string `json:"password"`
			Encryption string `json:"encryption"`
			FromEmail  string `json:"from_email"`
			FromName   string `json:"from_name"`
			ToEmail    string `json:"to_email"`
		} `json:"smtp"`
	}

	var input SystemConfigInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	changed := false

	// 更新服务端配置
	if input.Server != nil {
		if input.Server.Port > 0 {
			h.config.Server.Port = input.Server.Port
			changed = true
		}
		if input.Server.Mode != "" {
			h.config.Server.Mode = input.Server.Mode
			changed = true
		}
	}

	// 更新数据库配置
	if input.Database != nil {
		d := input.Database
		if d.Type != "" {
			h.config.Database.Type = d.Type
			changed = true
		}
		if d.Host != "" {
			h.config.Database.Host = d.Host
			changed = true
		}
		if d.Port > 0 {
			h.config.Database.Port = d.Port
			changed = true
		}
		if d.Name != "" {
			h.config.Database.Name = d.Name
			changed = true
		}
		if d.User != "" {
			h.config.Database.User = d.User
			changed = true
		}
		if d.Password != "" {
			h.config.Database.Password = d.Password
			changed = true
		}
		if d.SSLMode != "" {
			h.config.Database.SSLMode = d.SSLMode
			changed = true
		}
		if d.AutoMigrate != nil {
			h.config.Database.AutoMigrate = *d.AutoMigrate
			changed = true
		}
	}

	// 更新安全配置
	if input.Security != nil {
		s := input.Security
		if s.JWT != nil {
			if s.JWT.Secret != "" {
				h.config.Security.JWT.Secret = s.JWT.Secret
				changed = true
			}
			if s.JWT.Expire != "" {
				h.config.Security.JWT.Expire = s.JWT.Expire
				changed = true
			}
			if s.JWT.RefreshExpire != "" {
				h.config.Security.JWT.RefreshExpire = s.JWT.RefreshExpire
				changed = true
			}
		}
		if s.RateLimit != nil {
			h.config.Security.RateLimit.Enabled = s.RateLimit.Enabled
			changed = true
			if s.RateLimit.RequestsPerMinute > 0 {
				h.config.Security.RateLimit.RequestsPerMinute = s.RateLimit.RequestsPerMinute
				changed = true
			}
		}
		if s.CORS != nil {
			if len(s.CORS.AllowedOrigins) > 0 {
				h.config.Security.CORS.AllowedOrigins = s.CORS.AllowedOrigins
				changed = true
			}
			if len(s.CORS.AllowedMethods) > 0 {
				h.config.Security.CORS.AllowedMethods = s.CORS.AllowedMethods
				changed = true
			}
			if len(s.CORS.AllowedHeaders) > 0 {
				h.config.Security.CORS.AllowedHeaders = s.CORS.AllowedHeaders
				changed = true
			}
		}
		if s.HTTPS != nil {
			h.config.Security.HTTPS.Enabled = s.HTTPS.Enabled
			changed = true
			if s.HTTPS.CertFile != "" {
				h.config.Security.HTTPS.CertFile = s.HTTPS.CertFile
				changed = true
			}
			if s.HTTPS.KeyFile != "" {
				h.config.Security.HTTPS.KeyFile = s.HTTPS.KeyFile
				changed = true
			}
		}
	}

	// 更新 Embedding 配置
	if input.Embedding != nil {
		e := input.Embedding
		if e.Provider != "" {
			h.config.Embedding.Provider = e.Provider
			changed = true
		}
		if e.Model != "" {
			h.config.Embedding.Model = e.Model
			changed = true
		}
		if e.Dimension > 0 {
			h.config.Embedding.Dimension = e.Dimension
			changed = true
		}
		if e.Store != "" {
			h.config.Embedding.StoreProvider = e.Store
			changed = true
		}
	}

	// 更新 Executor 配置
	if input.Executor != nil {
		e := input.Executor
		if e.Mode != "" {
			h.config.Executor.Mode = e.Mode
			changed = true
		}
		if e.Embedded != nil {
			if e.Embedded.MaxConcurrent > 0 {
				h.config.Executor.EmbeddedConfig.MaxConcurrent = e.Embedded.MaxConcurrent
				changed = true
			}
			if e.Embedded.Timeout != "" {
				h.config.Executor.EmbeddedConfig.Timeout = e.Embedded.Timeout
				changed = true
			}
		}
		if e.Independent != nil && e.Independent.Queue != nil {
			if e.Independent.Queue.Type != "" {
				h.config.Executor.IndependentConfig.Queue.Type = e.Independent.Queue.Type
				changed = true
			}
			if e.Independent.Queue.Address != "" {
				h.config.Executor.IndependentConfig.Queue.Address = e.Independent.Queue.Address
				changed = true
			}
		}
	}

	// 更新自动更新配置
	if input.Update != nil {
		u := input.Update
		h.config.Update.Enabled = u.Enabled
		changed = true
		if u.CheckInterval != "" {
			h.config.Update.CheckInterval = u.CheckInterval
			changed = true
		}
		if u.GithubRepo != "" {
			h.config.Update.GithubRepo = u.GithubRepo
			changed = true
		}
		h.config.Update.AutoBackup = u.AutoBackup
		changed = true
		h.config.Update.RequireApproval = u.RequireApproval
		changed = true
	}

	// 更新 SMTP 配置
	if input.SMTP != nil {
		s := input.SMTP
		h.config.SMTP.Enabled = s.Enabled
		changed = true
		if s.Host != "" {
			h.config.SMTP.Host = s.Host
			changed = true
		}
		if s.Port > 0 {
			h.config.SMTP.Port = s.Port
			changed = true
		}
		if s.Username != "" {
			h.config.SMTP.Username = s.Username
			changed = true
		}
		if s.Password != "" {
			h.config.SMTP.Password = s.Password
			changed = true
		}
		if s.Encryption != "" {
			h.config.SMTP.Encryption = s.Encryption
			changed = true
		}
		if s.FromEmail != "" {
			h.config.SMTP.FromEmail = s.FromEmail
			changed = true
		}
		if s.FromName != "" {
			h.config.SMTP.FromName = s.FromName
			changed = true
		}
		if s.ToEmail != "" {
			h.config.SMTP.ToEmail = s.ToEmail
			changed = true
		}
	}

	if changed {
		if err := config.SaveToFile(h.config, "config.yaml"); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "保存配置失败: " + err.Error()})
		}
		return c.JSON(fiber.Map{"message": "配置已更新，部分修改需重启服务后生效"})
	}

	return c.JSON(fiber.Map{"message": "没有需要更新的配置"})
}

// TestSMTP 测试 SMTP 配置
func (h *Handler) TestSMTP(c *fiber.Ctx) error {
	// 如果 SMTP 未启用，返回错误
	if !h.config.SMTP.Enabled {
		return c.Status(400).JSON(fiber.Map{"error": "SMTP 未启用"})
	}

	// 检查必要的配置
	if h.config.SMTP.Host == "" || h.config.SMTP.Port == 0 || h.config.SMTP.FromEmail == "" {
		return c.Status(400).JSON(fiber.Map{"error": "SMTP 配置不完整"})
	}

	// 创建测试邮件
	subject := "Wisehoof 系统 SMTP 测试"
	body := "这是一封测试邮件，用于验证 SMTP 配置是否正确。\n\n如果收到此邮件，说明配置成功。"

	// 发送测试邮件
	err := sendTestEmail(h.config.SMTP, subject, body)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "发送测试邮件失败: " + err.Error()})
	}

	return c.JSON(fiber.Map{"message": "测试邮件已发送"})
}

// sendTestEmail 发送测试邮件
func sendTestEmail(cfg config.SMTPConfig, subject, body string) error {
	// 收件人：如果配置了 to_email 则使用，否则使用 from_email 作为测试收件人
	toEmail := cfg.ToEmail
	if toEmail == "" {
		toEmail = cfg.FromEmail
	}

	if toEmail == "" {
		return fiber.NewError(400, "未配置收件人邮箱")
	}

	// 构建邮件内容
	message := "From: " + cfg.FromName + " <" + cfg.FromEmail + ">\r\n"
	message += "To: " + toEmail + "\r\n"
	message += "Subject: " + subject + "\r\n"
	message += "Content-Type: text/plain; charset=utf-8\r\n"
	message += "\r\n"
	message += body

	// 连接 SMTP 服务器并发送
	// 这里使用 Go 的 net/smtp 包
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	var auth smtp.Auth
	switch strings.ToLower(cfg.Encryption) {
	case "ssl":
		// SSL/TLS 直接连接
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	default:
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}

	err := smtp.SendMail(addr, auth, cfg.FromEmail, []string{toEmail}, []byte(message))
	if err != nil {
		return fmt.Errorf("SMTP 发送失败: %w", err)
	}

	return nil
}
