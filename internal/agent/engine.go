package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/llm"
	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

type AgentEngine struct {
	db        *gorm.DB
	config    *config.Config
	skills    *SkillManager
	providers *ProviderManager
	memory    *MemoryManager
}

type ChatRequest struct {
	UserID         string                 `json:"user_id"`
	Message        string                 `json:"message"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
}

type ChatResponse struct {
	Message        string                 `json:"message"`
	ConversationID string                 `json:"conversation_id"`
	SkillUsed      string                 `json:"skill_used,omitempty"`
	Data           map[string]interface{} `json:"data,omitempty"`
	TokensUsed     int                    `json:"tokens_used,omitempty"`
}

type IntentResult struct {
	Intent     string                 `json:"intent"`
	Confidence float64                `json:"confidence"`
	Entities   map[string]string      `json:"entities"`
	Parameters map[string]interface{} `json:"parameters"`
}

type PlanningResult struct {
	Steps      []PlanStep `json:"steps"`
	SkillName  string     `json:"skill_name"`
	CanExecute bool       `json:"can_execute"`
	Reason     string     `json:"reason,omitempty"`
}

type PlanStep struct {
	StepID     string                 `json:"step_id"`
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters"`
	DependsOn  []string               `json:"depends_on,omitempty"`
}

func NewAgentEngine(db *gorm.DB, cfg *config.Config) *AgentEngine {
	return &AgentEngine{
		db:        db,
		config:    cfg,
		skills:    NewSkillManager(db),
		providers: NewProviderManager(db),
		memory:    NewMemoryManager(db),
	}
}

// ProcessUserMessage 处理用户消息的主入口
func (e *AgentEngine) ProcessUserMessage(req *ChatRequest) (*ChatResponse, error) {
	// 1. 获取用户上下文和权限
	user, err := e.getUserContext(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("获取用户上下文失败: %w", err)
	}

	// 2. 意图识别
	intent, err := e.recognizeIntent(req.Message, user)
	if err != nil {
		return nil, fmt.Errorf("意图识别失败: %w", err)
	}

	// 3. 规划执行步骤
	plan, err := e.planExecution(intent, user)
	if err != nil {
		return nil, fmt.Errorf("规划失败: %w", err)
	}

	// 4. 检查权限边界
	if !plan.CanExecute {
		return &ChatResponse{
			Message:        fmt.Sprintf("抱歉，您没有权限执行此操作: %s", plan.Reason),
			ConversationID: req.ConversationID,
		}, nil
	}

	// 5. 执行计划
	result, err := e.executePlan(plan, req, user)
	if err != nil {
		return nil, fmt.Errorf("执行失败: %w", err)
	}

	// 6. 记忆更新
	e.memory.SaveConversation(req.UserID, req.ConversationID, req.Message, result.Message)

	return result, nil
}

// getUserContext 获取用户上下文信息
func (e *AgentEngine) getUserContext(userID string) (*UserContext, error) {
	var user models.User
	if err := e.db.Preload("Groups").First(&user, userID).Error; err != nil {
		return nil, err
	}

	ctx := &UserContext{
		UserID:       user.ID,
		GlobalUserID: user.GlobalUserID,
		Name:         user.Name,
		Department:   user.Department,
		Status:       user.Status,
	}

	// 获取用户组信息
	for _, group := range user.Groups {
		ctx.Groups = append(ctx.Groups, GroupContext{
			ID:              group.ID,
			Name:            group.GroupName,
			Code:            group.GroupCode,
			DataAccessScope: group.DataAccessScope,
		})
	}

	// 获取用户可用的 Skills
	var skillAccess []models.SkillAccess
	e.db.Where("user_id = ?", userID).Or("group_id IN (?)", ctx.getGroupIDs()).Find(&skillAccess)
	for _, s := range skillAccess {
		ctx.AvailableSkills = append(ctx.AvailableSkills, s.SkillID)
	}

	return ctx, nil
}

// recognizeIntent 意图识别
func (e *AgentEngine) recognizeIntent(message string, user *UserContext) (*IntentResult, error) {
	// 简单的关键词匹配 + 向量匹配
	// 在实际实现中应该使用 LLM 或向量数据库

	lowerMsg := toLower(message)

	// 预定义意图匹配
	intentPatterns := map[string][]string{
		"text2sql":     {"查询", "查询数据", "统计", "sql", "query", "搜索", "找"},
		"chat":         {"你好", "hello", "在吗", "问下", "聊聊"},
		"file_process": {"处理文件", "上传文件", "解析文件", "read file"},
		"report":       {"生成报告", "报表", "导出", "report", "统计报表"},
		"api_caller":   {"调用接口", "请求API", "http", "api call"},
		"system":       {"系统", "设置", "配置", "system"},
	}

	for intent, keywords := range intentPatterns {
		for _, kw := range keywords {
			if contains(lowerMsg, kw) {
				return &IntentResult{
					Intent:     intent,
					Confidence: 0.9,
					Entities:   extractEntities(message),
					Parameters: map[string]interface{}{},
				}, nil
			}
		}
	}

	// 默认 chat 意图
	return &IntentResult{
		Intent:     "chat",
		Confidence: 0.5,
		Entities:   map[string]string{},
		Parameters: map[string]interface{}{},
	}, nil
}

// planExecution 规划执行
func (e *AgentEngine) planExecution(intent *IntentResult, user *UserContext) (*PlanningResult, error) {
	// 根据意图选择 Skill
	skillMap := map[string]string{
		"text2sql":     "skill_text2sql",
		"chat":         "skill_chat",
		"file_process": "skill_file_process",
		"report":       "skill_report",
		"api_caller":   "skill_api_caller",
	}

	skillName := skillMap[intent.Intent]
	if skillName == "" {
		skillName = "skill_chat"
	}

	// 检查权限
	canExecute, reason := e.checkPermission(user, skillName)

	plan := &PlanningResult{
		Steps: []PlanStep{
			{
				StepID: "step_1",
				Action: "execute_skill",
				Parameters: map[string]interface{}{
					"skill_name": skillName,
					"intent":     intent,
				},
			},
		},
		SkillName:  skillName,
		CanExecute: canExecute,
		Reason:     reason,
	}

	return plan, nil
}

// checkPermission 检查权限边界
func (e *AgentEngine) checkPermission(user *UserContext, skillName string) (bool, string) {
	// 检查用户是否激活
	if user.Status != "active" {
		return false, "用户账户未激活"
	}

	// 检查 Skill 是否在允许列表中
	for _, skill := range user.AvailableSkills {
		if skill == skillName {
			return true, ""
		}
	}

	// 允许管理员使用所有技能
	for _, group := range user.Groups {
		if group.Code == "admin" || group.Code == "Administrators" {
			return true, ""
		}
	}

	return false, fmt.Sprintf("您没有权限使用技能: %s", skillName)
}

// executePlan 执行计划
func (e *AgentEngine) executePlan(plan *PlanningResult, req *ChatRequest, user *UserContext) (*ChatResponse, error) {
	response := &ChatResponse{
		ConversationID: req.ConversationID,
		SkillUsed:      plan.SkillName,
	}

	switch plan.SkillName {
	case "skill_text2sql":
		result, err := e.executeText2SQL(req.Message, user)
		if err != nil {
			response.Message = fmt.Sprintf("SQL查询失败: %v", err)
		} else {
			response.Message = result.Message
			response.Data = result.Data
		}

	case "skill_chat":
		result, err := e.executeChat(req.Message, user)
		if err != nil {
			response.Message = fmt.Sprintf("对话处理失败: %v", err)
		} else {
			response.Message = result.Message
		}

	case "skill_report":
		result, err := e.executeReport(req.Message, user)
		if err != nil {
			response.Message = fmt.Sprintf("报告生成失败: %v", err)
		} else {
			response.Message = result.Message
			response.Data = result.Data
		}

	default:
		// 默认 chat
		result, _ := e.executeChat(req.Message, user)
		response.Message = result.Message
	}

	return response, nil
}

func (e *AgentEngine) executeText2SQL(message string, user *UserContext) (*ChatResponse, error) {
	return &ChatResponse{
		Message: "Text2SQL 功能需要配置数据源后使用",
		Data: map[string]interface{}{
			"type":    "text2sql",
			"message": "请在管理后台配置数据源后再使用此功能",
		},
	}, nil
}

func (e *AgentEngine) executeChat(message string, user *UserContext) (*ChatResponse, error) {
	// 获取对话历史
	messages, _ := e.memory.GetMessages(user.UserID, 10)

	// 构建 prompt
	_ = buildChatPrompt(message, messages, user)

	// 调用 LLM - 如果配置了 Provider 则调用真实 LLM
	provider, err := e.providers.GetActiveProvider()
	if err != nil || provider == nil {
		// 如果没有配置 Provider，返回模拟响应
		response := &ChatResponse{
			Message: fmt.Sprintf("[模拟回复] 您说: %s", message),
		}
		return response, nil
	}

	// 使用真实的 LLM 调用
	result, err := e.providers.CallLLM(nil, provider, message)
	if err != nil {
		return &ChatResponse{
			Message: fmt.Sprintf("调用 LLM 失败: %v", err),
		}, nil
	}

	response := &ChatResponse{
		Message: result,
	}

	return response, nil
}

func (e *AgentEngine) executeReport(message string, user *UserContext) (*ChatResponse, error) {
	return &ChatResponse{
		Message: "报告生成功能需要配置数据源后使用",
		Data: map[string]interface{}{
			"type": "report",
		},
	}, nil
}

// UserContext 用户上下文
type UserContext struct {
	UserID          string
	GlobalUserID    string
	Name            string
	Department      string
	Status          string
	Groups          []GroupContext
	AvailableSkills []string
}

func (u *UserContext) getGroupIDs() []string {
	ids := make([]string, len(u.Groups))
	for i, g := range u.Groups {
		ids[i] = g.ID
	}
	return ids
}

// GroupContext 用户组上下文
type GroupContext struct {
	ID              string
	Name            string
	Code            string
	DataAccessScope string
}

// 辅助函数
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if s == substr {
		return true
	}
	return contains(s[1:], substr)
}

func extractEntities(message string) map[string]string {
	entities := make(map[string]string)
	// 简单实体提取示例
	if len(message) > 0 {
		entities["raw"] = message
	}
	return entities
}

func buildChatPrompt(message string, history []models.Message, user *UserContext) string {
	prompt := fmt.Sprintf("用户: %s\n", user.Name)

	for _, msg := range history {
		prompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	prompt += fmt.Sprintf("\n当前输入: %s\n\n请回复:", message)
	return prompt
}

// ProviderManager LLM Provider 管理
type ProviderManager struct {
	db *gorm.DB
}

func NewProviderManager(db *gorm.DB) *ProviderManager {
	return &ProviderManager{db: db}
}

func (m *ProviderManager) GetActiveProvider() (*models.Provider, error) {
	var provider models.Provider
	err := m.db.Where("enabled = ?", true).Order("priority ASC").First(&provider).Error
	return &provider, err
}

func (m *ProviderManager) CallLLM(ctx context.Context, provider *models.Provider, prompt string) (string, error) {
	// Use the new LLM client for actual API calls
	client, err := llm.NewClient(provider)
	if err != nil {
		return "", fmt.Errorf("failed to create LLM client: %w", err)
	}

	resp, err := client.ChatCompletion(ctx, llm.ChatRequest{
		Model: provider.Model,
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   2048,
		Temperature: 0.7,
	})

	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	return resp.Content, nil
}

// SkillManager Skills 管理
type SkillManager struct {
	db *gorm.DB
}

func NewSkillManager(db *gorm.DB) *SkillManager {
	return &SkillManager{db: db}
}

func (m *SkillManager) GetSkill(name string) (*models.Skill, error) {
	var skill models.Skill
	err := m.db.Where("name = ? AND enabled = ?", name, true).First(&skill).Error
	return &skill, err
}

// MemoryManager 记忆管理
type MemoryManager struct {
	db *gorm.DB
}

func NewMemoryManager(db *gorm.DB) *MemoryManager {
	return &MemoryManager{db: db}
}

func (m *MemoryManager) SaveConversation(userID, convID, userMsg, assistantMsg string) error {
	// 保存对话到数据库
	user := models.User{}
	if err := m.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}

	// 创建或获取会话
	conv := models.Conversation{}
	result := m.db.Where("id = ? AND user_id = ?", convID, userID).First(&conv)
	if result.Error == gorm.ErrRecordNotFound {
		conv = models.Conversation{
			ID:     convID,
			UserID: userID,
			Source: "api",
			Title:  "对话 " + time.Now().Format("2006-01-02 15:04"),
		}
		m.db.Create(&conv)
	}

	// 保存用户消息
	userMsgModel := models.Message{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        userMsg,
	}
	m.db.Create(&userMsgModel)

	// 保存助手消息
	assistantMsgModel := models.Message{
		ConversationID: conv.ID,
		Role:           "assistant",
		Content:        assistantMsg,
	}
	m.db.Create(&assistantMsgModel)

	return nil
}

func (m *MemoryManager) GetMessages(userID string, limit int) ([]models.Message, error) {
	var convs []models.Conversation
	m.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(1).Find(&convs)

	if len(convs) == 0 {
		return []models.Message{}, nil
	}

	var messages []models.Message
	m.db.Where("conversation_id = ?", convs[0].ID).Order("created_at ASC").Limit(limit).Find(&messages)

	return messages, nil
}

func (m *MemoryManager) GetUserMemory(userID, memoryType string) (string, error) {
	// 获取用户的长期记忆
	// 这里应该有实际的记忆表
	return "", nil
}

func (m *MemoryManager) SaveUserMemory(userID, memoryType, content string) error {
	// 保存用户的长期记忆
	return nil
}

func (m *MemoryManager) GetGroupMemory(groupID, memoryType string) (string, error) {
	// 获取用户组的共享记忆
	return "", nil
}

func (m *MemoryManager) SaveGroupMemory(groupID, memoryType, content string) error {
	// 保存用户组的共享记忆
	return nil
}

var _ = json.Marshal // use json
