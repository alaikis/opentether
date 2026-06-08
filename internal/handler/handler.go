package handler

import (
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

func (h *Handler) FeishuCallback(c *fiber.Ctx) error {
	return h.handleIMCallback(c, "feishu")
}

func (h *Handler) DingTalkCallback(c *fiber.Ctx) error {
	return h.handleIMCallback(c, "dingtalk")
}

func (h *Handler) WhatsAppCallback(c *fiber.Ctx) error {
	return h.handleIMCallback(c, "whatsapp")
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
		"success":   true,
		"skill_id":  skill.ID,
		"skill_name": skill.Name,
		"parsed":    parsed,
	})
}

// ParseMarkdownPreview 预览解析 MD 文件内容
func (h *Handler) ParseMarkdownPreview(c *fiber.Ctx) error {
	type Request struct {
		Content string `json:"content"`
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
		ToolName   string                 `json:"tool_name"`
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
		Markdown string              `json:"markdown"`
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
