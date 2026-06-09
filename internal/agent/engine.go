package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/embedding"
	"github.com/alaikis/opentether/internal/llm"
	"github.com/alaikis/opentether/internal/models"
	"github.com/alaikis/opentether/internal/text2sql"
	"github.com/alaikis/opentether/internal/vectorstore"
	"gorm.io/gorm"
)

type AgentEngine struct {
	db         *gorm.DB
	config     *config.Config
	skills     *SkillManager
	providers  *ProviderManager
	memory     *MemoryManager
	experience *ExperienceManager
	env        *EnvManager
	scripts    *ScriptManager
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
		db:         db,
		config:     cfg,
		skills:     NewSkillManager(db),
		providers:  NewProviderManager(db),
		memory:     NewMemoryManager(db),
		experience: NewExperienceManager(db),
		env:        NewEnvManager(),
		scripts:    NewScriptManager(db),
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

// recognizeIntent 意图识别：向量语义匹配优先 → 关键词匹配兜底
// 不做开放式 fallback，无匹配时返回 boundary_reject
func (e *AgentEngine) recognizeIntent(message string, user *UserContext) (*IntentResult, error) {
	lowerMsg := toLower(message)

	// === 第一层：向量语义匹配（置信度阈值 0.15）===
	vectorSkill, score, err := e.skills.MatchByVector(message, 0.15)
	if err == nil && vectorSkill != nil && score > 0 {
		return &IntentResult{
			Intent:     vectorSkill.SkillType,
			Confidence: score,
			Entities:   extractEntities(message),
			Parameters: map[string]interface{}{
				"skill_name": vectorSkill.Name,
				"skill_id":   vectorSkill.ID,
				"match_type": "vector",
				"score":      score,
			},
		}, nil
	}

	// === 第二层：关键词匹配兜底 ===
	skills, err := e.skills.ListEnabledSkills()
	if err != nil {
		return nil, fmt.Errorf("获取 Skill 列表失败: %w", err)
	}

	for _, skill := range skills {
		keywords := parseKeywords(skill.Keywords)
		keywords = append(keywords, toLower(skill.Name), toLower(skill.SkillType))

		for _, kw := range keywords {
			if kw != "" && contains(lowerMsg, toLower(kw)) {
				return &IntentResult{
					Intent:     skill.SkillType,
					Confidence: 0.7,
					Entities:   extractEntities(message),
					Parameters: map[string]interface{}{
						"skill_name": skill.Name,
						"skill_id":   skill.ID,
						"match_type": "keyword",
					},
				}, nil
			}
		}
	}

	// 无匹配 → 边界拒绝
	return &IntentResult{
		Intent:     "boundary_reject",
		Confidence: 0,
		Entities:   map[string]string{},
		Parameters: map[string]interface{}{
			"reason":       "no_matching_skill",
			"user_message": message,
		},
	}, nil
}

// parseKeywords 解析 Skill 的关键词 JSON 数组
func parseKeywords(keywordsJSON string) []string {
	if keywordsJSON == "" {
		return nil
	}
	var keywords []string
	if err := json.Unmarshal([]byte(keywordsJSON), &keywords); err != nil {
		return strings.Split(keywordsJSON, ",")
	}
	return keywords
}

// planExecution 规划执行
func (e *AgentEngine) planExecution(intent *IntentResult, user *UserContext) (*PlanningResult, error) {
	// 边界拒绝：未匹配到任何注册 Skill
	if intent.Intent == "boundary_reject" {
		return &PlanningResult{
			SkillName:  "",
			CanExecute: false,
			Reason:     "抱歉，您的问题超出了当前系统支持的范围。请使用已注册的 Skill 进行操作。如需帮助，请联系管理员配置相关 Skill。",
		}, nil
	}

	// 根据意图选择 Skill
	skillMap := map[string]string{
		"text2sql":     "skill_text2sql",
		"chat":         "skill_chat",
		"file_process": "skill_file_process",
		"report":       "skill_report",
		"api_caller":   "skill_api_caller",
		"employee":     "skill_employee",
		"pdf":          "skill_pdf",
		"md2pdf":       "skill_md2pdf",
	}

	skillName := skillMap[intent.Intent]
	if skillName == "" {
		// 直接从 intent 提取 skill_name
		if name, ok := intent.Parameters["skill_name"].(string); ok {
			skillName = name
		}
	}

	if skillName == "" {
		return &PlanningResult{
			SkillName:  "",
			CanExecute: false,
			Reason:     "无法识别对应的 Skill，请联系管理员。",
		}, nil
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

	case "skill_employee":
		result, err := e.executeEmployee(req.Message, user)
		if err != nil {
			response.Message = fmt.Sprintf("员工查询失败: %v", err)
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

	case "skill_pdf":
		result, err := e.executePDFReport(req.Message, user)
		if err != nil {
			response.Message = fmt.Sprintf("PDF生成失败: %v", err)
		} else {
			response.Message = result.Message
			response.Data = result.Data
		}

	case "skill_md2pdf":
		result, err := e.executeMD2PDF(req.Message, user)
		if err != nil {
			response.Message = fmt.Sprintf("Markdown转PDF失败: %v", err)
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
	// 从用户上下文获取数据源配置
	dataSourceID := ""
	if user.Context != nil {
		if ds, ok := user.Context["data_source_id"].(string); ok {
			dataSourceID = ds
		}
	}

	// 如果没有指定数据源，尝试获取第一个启用的数据源
	if dataSourceID == "" {
		var ds models.DataSource
		err := e.db.Where("enabled = ?", true).First(&ds).Error
		if err != nil {
			return &ChatResponse{
				Message: "请先在管理后台配置数据源后再使用查询功能",
				Data: map[string]interface{}{
					"type":    "text2sql",
					"action":  "configure_datasource",
					"message": "未配置数据源，请先在数据源管理中添加数据库连接",
				},
			}, nil
		}
		dataSourceID = ds.ID
	}

	// 获取活跃的 LLM Provider
	provider, err := e.providers.GetActiveProvider()
	if err != nil || provider == nil {
		return &ChatResponse{
			Message: "请先在管理后台配置 LLM Provider 后再使用查询功能",
			Data: map[string]interface{}{
				"type":    "text2sql",
				"action":  "configure_provider",
				"message": "未配置 LLM Provider，请先在 Provider 管理中添加配置",
			},
		}, nil
	}

	// 创建 LLM Client
	client, err := llm.NewClient(provider)
	if err != nil {
		return &ChatResponse{
			Message: fmt.Sprintf("创建 LLM 客户端失败: %v", err),
			Data: map[string]interface{}{
				"type":  "text2sql",
				"error": err.Error(),
			},
		}, nil
	}

	// 创建 Text2SQL 实例
	t2s, err := text2sql.NewWithDataSource(e.db, client, dataSourceID)
	if err != nil {
		return &ChatResponse{
			Message: fmt.Sprintf("连接数据源失败: %v", err),
			Data: map[string]interface{}{
				"type":  "text2sql",
				"error": err.Error(),
			},
		}, nil
	}
	defer t2s.Close()

	// 执行查询
	req := &text2sql.QueryRequest{
		Question:     message,
		DataSourceID: dataSourceID,
	}

	result, err := t2s.ExecuteSQL(context.Background(), req)
	if err != nil {
		return &ChatResponse{
			Message: fmt.Sprintf("查询失败: %v", err),
		}, nil
	}

	if result.Error != "" {
		return &ChatResponse{
			Message: result.Error,
			Data: map[string]interface{}{
				"type":   "text2sql",
				"error":  result.Error,
				"sql":    result.SQL,
				"status": "failed",
			},
		}, nil
	}

	// 格式化结果
	responseMessage := formatQueryResult(result)

	return &ChatResponse{
		Message: responseMessage,
		Data: map[string]interface{}{
			"type":           "text2sql",
			"sql":            result.SQL,
			"columns":        result.Columns,
			"rows":           result.Rows,
			"row_count":      result.RowCount,
			"execution_time": result.ExecutionTime,
			"data_source_id": dataSourceID,
		},
		SkillUsed: "skill_text2sql",
	}, nil
}

// executeEmployee 处理员工查询
func (e *AgentEngine) executeEmployee(message string, user *UserContext) (*ChatResponse, error) {
	// 员工查询使用特殊的 prompt 和数据源
	// 首先尝试找到 HR/员工数据相关的数据源
	dataSourceID := ""

	// 从用户上下文获取数据源，或者查找名为 "hr" 或 "employee" 的数据源
	if user.Context != nil {
		if ds, ok := user.Context["employee_data_source_id"].(string); ok {
			dataSourceID = ds
		}
	}

	// 如果没有指定，尝试查找相关数据源
	if dataSourceID == "" {
		var ds models.DataSource
		err := e.db.Where("enabled = ? AND (name LIKE ? OR name LIKE ?)", true, "%hr%", "%员工%").First(&ds).Error
		if err != nil {
			// 尝试获取第一个启用的数据源
			err = e.db.Where("enabled = ?", true).First(&ds).Error
			if err != nil {
				return &ChatResponse{
					Message: "请先在管理后台配置员工数据源后再使用员工查询功能",
					Data: map[string]interface{}{
						"type":    "employee",
						"action":  "configure_datasource",
						"message": "未配置员工数据源，请先在数据源管理中添加数据库连接",
					},
				}, nil
			}
		}
		dataSourceID = ds.ID
	}

	// 获取活跃的 LLM Provider
	provider, err := e.providers.GetActiveProvider()
	if err != nil || provider == nil {
		return &ChatResponse{
			Message: "请先在管理后台配置 LLM Provider 后再使用查询功能",
			Data: map[string]interface{}{
				"type":    "text2sql",
				"action":  "configure_provider",
				"message": "未配置 LLM Provider，请先在 Provider 管理中添加配置",
			},
		}, nil
	}

	// 创建 LLM Client
	client, err := llm.NewClient(provider)
	if err != nil {
		return &ChatResponse{
			Message: fmt.Sprintf("创建 LLM 客户端失败: %v", err),
			Data: map[string]interface{}{
				"type":  "text2sql",
				"error": err.Error(),
			},
		}, nil
	}

	// 创建 Text2SQL 实例
	t2s, err := text2sql.NewWithDataSource(e.db, client, dataSourceID)
	if err != nil {
		return &ChatResponse{
			Message: fmt.Sprintf("连接数据源失败: %v", err),
			Data: map[string]interface{}{
				"type":  "employee",
				"error": err.Error(),
			},
		}, nil
	}
	defer t2s.Close()

	// 为员工查询构建特殊的 prompt
	req := &text2sql.QueryRequest{
		Question:     message,
		DataSourceID: dataSourceID,
	}

	result, err := t2s.ExecuteSQL(context.Background(), req)
	if err != nil {
		return &ChatResponse{
			Message: fmt.Sprintf("员工查询失败: %v", err),
		}, nil
	}

	if result.Error != "" {
		return &ChatResponse{
			Message: result.Error,
			Data: map[string]interface{}{
				"type":   "employee",
				"error":  result.Error,
				"sql":    result.SQL,
				"status": "failed",
			},
		}, nil
	}

	// 格式化结果，优先显示员工相关的列
	responseMessage := formatEmployeeResult(result)

	return &ChatResponse{
		Message: responseMessage,
		Data: map[string]interface{}{
			"type":           "employee",
			"sql":            result.SQL,
			"columns":        result.Columns,
			"rows":           result.Rows,
			"row_count":      result.RowCount,
			"execution_time": result.ExecutionTime,
			"data_source_id": dataSourceID,
		},
		SkillUsed: "skill_employee",
	}, nil
}

// formatQueryResult 格式化查询结果为可读文本
func formatQueryResult(result *text2sql.QueryResult) string {
	if result.Error != "" {
		return result.Error
	}

	if result.RowCount == 0 {
		return "查询完成，未找到相关数据"
	}

	var output string
	output += fmt.Sprintf("查询完成，共找到 %d 条记录\n\n", result.RowCount)

	// 显示前10条
	maxRows := result.RowCount
	if maxRows > 10 {
		maxRows = 10
		output += "(仅显示前10条)\n\n"
	}

	for i := 0; i < maxRows; i++ {
		var rowStr []string
		for j, col := range result.Columns {
			rowStr = append(rowStr, fmt.Sprintf("%s: %v", col, result.Rows[i][j]))
		}
		output += fmt.Sprintf("%d. %s\n", i+1, strings.Join(rowStr, ", "))
	}

	return output
}

// formatEmployeeResult 格式化员工查询结果
func formatEmployeeResult(result *text2sql.QueryResult) string {
	if result.Error != "" {
		return result.Error
	}

	if result.RowCount == 0 {
		return "查询完成，未找到相关员工数据"
	}

	var output string
	output += fmt.Sprintf("查询完成，共找到 %d 条员工记录\n\n", result.RowCount)

	// 查找员工姓名列（常见的列名）
	nameColIdx := -1
	for i, col := range result.Columns {
		lowerCol := strings.ToLower(col)
		if strings.Contains(lowerCol, "name") || strings.Contains(lowerCol, "姓名") || strings.Contains(lowerCol, "员工") {
			nameColIdx = i
			break
		}
	}

	// 显示前10条
	maxRows := result.RowCount
	if maxRows > 10 {
		maxRows = 10
		output += "(仅显示前10条)\n\n"
	}

	for i := 0; i < maxRows; i++ {
		if nameColIdx >= 0 && nameColIdx < len(result.Rows[i]) {
			output += fmt.Sprintf("%d. %v\n", i+1, result.Rows[i][nameColIdx])
		} else {
			var rowStr []string
			for j, col := range result.Columns {
				if j < len(result.Rows[i]) {
					rowStr = append(rowStr, fmt.Sprintf("%s: %v", col, result.Rows[i][j]))
				}
			}
			output += fmt.Sprintf("%d. %s\n", i+1, strings.Join(rowStr, ", "))
		}
	}

	output += fmt.Sprintf("\n执行时间: %s", result.ExecutionTime)

	return output
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

// executePDFReport 执行 PDF 报表生成
func (e *AgentEngine) executePDFReport(message string, user *UserContext) (*ChatResponse, error) {
	// 从用户上下文获取查询数据
	dataSourceID := ""
	if user.Context != nil {
		if ds, ok := user.Context["data_source_id"].(string); ok {
			dataSourceID = ds
		}
	}

	// 如果没有指定数据源，尝试获取第一个启用的数据源
	if dataSourceID == "" {
		var ds models.DataSource
		err := e.db.Where("enabled = ?", true).First(&ds).Error
		if err != nil {
			return &ChatResponse{
				Message: "请先在管理后台配置数据源后再使用 PDF 报表功能",
				Data: map[string]interface{}{
					"type":   "pdf",
					"action": "configure_datasource",
				},
			}, nil
		}
		dataSourceID = ds.ID
	}

	// 获取活跃的 LLM Provider
	provider, err := e.providers.GetActiveProvider()
	if err != nil || provider == nil {
		return &ChatResponse{
			Message: "请先在管理后台配置 LLM Provider 后再使用报表功能",
			Data: map[string]interface{}{
				"type":   "pdf",
				"action": "configure_provider",
			},
		}, nil
	}

	// 创建 LLM Client
	client, err := llm.NewClient(provider)
	if err != nil {
		return &ChatResponse{
			Message: fmt.Sprintf("创建 LLM 客户端失败: %v", err),
			Data: map[string]interface{}{
				"type":  "pdf",
				"error": err.Error(),
			},
		}, nil
	}

	// 使用 text2sql 获取数据
	t2s, err := text2sql.NewWithDataSource(e.db, client, dataSourceID)
	if err != nil {
		return &ChatResponse{
			Message: fmt.Sprintf("连接数据源失败: %v", err),
		}, nil
	}
	defer t2s.Close()

	// 构建查询 - 尝试从 message 中提取查询意图
	queryMessage := message
	// 如果用户没有明确指定查询，构建一个默认查询
	if !contains(toLower(message), "查询") && !contains(toLower(message), "select") {
		queryMessage = message + " 相关数据"
	}

	req := &text2sql.QueryRequest{
		Question:     queryMessage,
		DataSourceID: dataSourceID,
	}

	result, err := t2s.ExecuteSQL(context.Background(), req)
	if err != nil {
		return &ChatResponse{
			Message: fmt.Sprintf("查询失败: %v", err),
		}, nil
	}

	if result.Error != "" {
		return &ChatResponse{
			Message: result.Error,
		}, nil
	}

	// 生成 PDF (需要使用 PDFService，但这需要单独的实例)
	// 这里我们返回数据，让用户可以下载为 PDF
	message = fmt.Sprintf("已查询到 %d 条数据，正在生成 PDF 报表...", result.RowCount)

	return &ChatResponse{
		Message: message,
		Data: map[string]interface{}{
			"type":           "pdf",
			"sql":            result.SQL,
			"columns":        result.Columns,
			"rows":           result.Rows,
			"row_count":      result.RowCount,
			"data_source_id": dataSourceID,
			"action":         "generate_pdf",
			"api_endpoint":   "/api/v1/admin/reports/query-pdf",
		},
		SkillUsed: "skill_pdf",
	}, nil
}

// executeMD2PDF 执行 Markdown 转 PDF
func (e *AgentEngine) executeMD2PDF(message string, user *UserContext) (*ChatResponse, error) {
	// Markdown 转 PDF 需要用户提供 Markdown 内容
	// 这个功能主要用于转换用户提供的文档内容

	// 尝试从消息中提取 markdown 内容的标记
	var markdownContent string

	// 检查是否有 markdown 代码块
	markdownBlock := extractMarkdownBlock(message)
	if markdownBlock != "" {
		markdownContent = markdownBlock
	} else {
		// 如果没有代码块，假设整个消息就是 markdown 内容
		markdownContent = message
	}

	// 生成 PDF (需要使用 MarkdownPDFService)
	// 由于 AgentEngine 现在没有 MarkdownPDFService，我们需要返回数据让前端处理
	message = "正在将 Markdown 内容转换为 PDF..."

	return &ChatResponse{
		Message: message,
		Data: map[string]interface{}{
			"type":         "md2pdf",
			"markdown":     markdownContent,
			"action":       "convert_to_pdf",
			"api_endpoint": "/api/v1/admin/docs/md2pdf",
		},
		SkillUsed: "skill_md2pdf",
	}, nil
}

// extractMarkdownBlock 从消息中提取 markdown 代码块内容
func extractMarkdownBlock(message string) string {
	// 匹配 ```markdown 或 ``` 包裹的内容
	patterns := []string{
		"(?s)```markdown\\s*(.+?)```,",
		"(?s)```md\\s*(.+?)```,",
		"(?s)```\\s*(.+?)```,",
	}

	for _, pattern := range patterns {
		if match := regexp.MustCompile(pattern).FindStringSubmatch(message); len(match) > 1 {
			return strings.TrimSpace(match[1])
		}
	}

	return ""
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
	Context         map[string]interface{} // 额外的上下文数据
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
	db       *gorm.DB
	embedder embedding.Embedder
	store    vectorstore.Store
}

func NewSkillManager(db *gorm.DB) *SkillManager {
	sm := &SkillManager{db: db}
	// 延迟初始化，由外部调用 InitVector
	return sm
}

// InitVector 初始化向量层（加载配置 → 创建 Embedder → 构建索引）
func (m *SkillManager) InitVector(vc VectorConfig) error {
	skills, err := m.ListEnabledSkills()
	if err != nil || len(skills) == 0 {
		return err
	}

	// 构建语料库
	docs := make([]string, len(skills))
	for i, s := range skills {
		docs[i] = s.Name + " " + s.Description + " " + s.Keywords
	}

	// 创建 Embedder
	m.embedder, err = NewEmbedder(vc, docs)
	if err != nil {
		return fmt.Errorf("创建 Embedder 失败: %w", err)
	}

	// 创建 VectorStore
	m.store, err = NewVectorStore(vc)
	if err != nil {
		return fmt.Errorf("创建 VectorStore 失败: %w", err)
	}

	// 加载已有向量到 Store
	loaded := 0
	for _, skill := range skills {
		if skill.VectorEnabled && len(skill.VectorIndex) > 0 {
			if vec, err := DecodeVector(skill.VectorIndex); err == nil {
				m.store.Index(skill.ID, skill.Name, vec)
				loaded++
			}
		}
	}

	if loaded == 0 {
		// 索引为空，触发首次全量同步
		return m.SyncVector()
	}

	return nil
}

func (m *SkillManager) GetSkill(name string) (*models.Skill, error) {
	var skill models.Skill
	err := m.db.Where("name = ? AND enabled = ?", name, true).First(&skill).Error
	return &skill, err
}

// ListEnabledSkills 获取所有启用的 Skill
func (m *SkillManager) ListEnabledSkills() ([]models.Skill, error) {
	var skills []models.Skill
	err := m.db.Where("enabled = ?", true).Find(&skills).Error
	return skills, err
}

// SyncVector 同步所有 Skill 的向量索引（全量重建）
func (m *SkillManager) SyncVector() error {
	skills, err := m.ListEnabledSkills()
	if err != nil {
		return fmt.Errorf("获取 Skill 列表失败: %w", err)
	}

	if m.embedder == nil || m.store == nil {
		return fmt.Errorf("向量层未初始化，请先调用 InitVector")
	}

	// 清空旧索引
	m.store.Clear()

	// 重建词表（TF-IDF 需要）
	docs := make([]string, len(skills))
	for i, s := range skills {
		docs[i] = s.Name + " " + s.Description + " " + s.Keywords
	}
	if tfidf, ok := m.embedder.(*embedding.TFIDFEmbedder); ok {
		tfidf.BuildVocabulary(docs)
	}

	// 计算向量并存储
	for _, skill := range skills {
		doc := skill.Name + " " + skill.Description + " " + skill.Keywords
		vec, err := m.embedder.Embed(doc)
		if err != nil {
			return fmt.Errorf("向量化失败 [%s]: %w", skill.Name, err)
		}

		encoded := EncodeVector(vec)
		if err := m.db.Model(&skill).Updates(map[string]interface{}{
			"vector_index":   encoded,
			"vector_enabled": true,
		}).Error; err != nil {
			return fmt.Errorf("存储向量失败 [%s]: %w", skill.Name, err)
		}

		m.store.Index(skill.ID, skill.Name, vec)
	}

	return nil
}

// MatchByVector 向量匹配：使用 Embedder + VectorStore 搜索最佳匹配
func (m *SkillManager) MatchByVector(message string, threshold float64) (*models.Skill, float64, error) {
	if m.embedder == nil || m.store == nil || m.store.Count() == 0 {
		return nil, 0, nil
	}

	// 向量化用户消息
	queryVec, err := m.embedder.Embed(message)
	if err != nil {
		return nil, 0, fmt.Errorf("向量化消息失败: %w", err)
	}

	// 搜索最佳匹配
	matches, err := m.store.Search(queryVec, 1, threshold)
	if err != nil || len(matches) == 0 {
		return nil, 0, nil
	}

	// 从数据库加载 Skill 详情
	var skill models.Skill
	if err := m.db.Where("id = ?", matches[0].SkillID).First(&skill).Error; err != nil {
		return nil, 0, nil
	}

	return &skill, matches[0].Score, nil
}

// VectorStore 返回底层 VectorStore（供外部使用）
func (m *SkillManager) VectorStore() vectorstore.Store {
	return m.store
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
