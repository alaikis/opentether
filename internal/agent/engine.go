package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/database"
	"github.com/alaikis/opentether/internal/embedding"
	"github.com/alaikis/opentether/internal/llm"
	"github.com/alaikis/opentether/internal/models"
	"github.com/alaikis/opentether/internal/storage"
	"github.com/alaikis/opentether/internal/templating"
	"github.com/alaikis/opentether/internal/text2sql"
	"github.com/alaikis/opentether/internal/vectorstore"
	"gorm.io/gorm"
)

type AgentEngine struct {
	db             *gorm.DB
	config         *config.Config
	skills         *SkillManager
	providers      *ProviderManager
	memory         *MemoryManager
	longTermMemory *LongTermMemory
	experience     *ExperienceManager
	env            *EnvManager
	scripts        *ScriptManager
	store          storage.Driver
	sqlAuditor     text2sql.AuditRecorder
	externalDBPool *database.ExternalDBPoolManager
	mcpProvider    MCPToolProvider
	fastClassifier *FastPathClassifier
}

// MCPToolProvider exposes running MCP tools to the Agent without importing the service package.
type MCPToolProvider interface {
	ListAvailableTools() []MCPRuntimeTool
	CallTool(serverID, toolName string, arguments map[string]interface{}) (json.RawMessage, error)
}

type MCPRuntimeTool struct {
	ServerID    string
	ServerName  string
	Name        string
	Description string
	InputSchema json.RawMessage
}

// SetSQLAuditor 设置 SQL 审计器
func (e *AgentEngine) SetSQLAuditor(auditor text2sql.AuditRecorder) {
	e.sqlAuditor = auditor
}

// SetMCPProvider 设置 MCP 工具提供器。
func (e *AgentEngine) SetMCPProvider(provider MCPToolProvider) {
	e.mcpProvider = provider
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

func (r *ChatResponse) SkillUsedToRoute() string {
	if r == nil {
		return "agent_loop"
	}
	switch r.SkillUsed {
	case "fast_local":
		return "fast_local"
	case "fast_chat":
		return "fast_chat"
	case "fast_text2sql_template", "fast_text2sql_approved_template":
		return "fast_text2sql"
	default:
		return "agent_loop"
	}
}

type IntentResult struct {
	Intent     string                 `json:"intent"`
	Confidence float64                `json:"confidence"`
	Entities   map[string]string      `json:"entities"`
	Parameters map[string]interface{} `json:"parameters"`
	Candidates []SkillCandidate       `json:"candidates,omitempty"`
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

func NewAgentEngine(db *gorm.DB, cfg *config.Config, store storage.Driver) *AgentEngine {
	return NewAgentEngineWithExternalDBPool(db, cfg, store, database.NewExternalDBPoolManager(db, nil))
}

func NewAgentEngineWithExternalDBPool(db *gorm.DB, cfg *config.Config, store storage.Driver, externalDBPool *database.ExternalDBPoolManager) *AgentEngine {
	if externalDBPool == nil {
		externalDBPool = database.NewExternalDBPoolManager(db, nil)
	}
	return &AgentEngine{
		db:             db,
		config:         cfg,
		skills:         NewSkillManager(db),
		providers:      NewProviderManager(db),
		memory:         NewMemoryManager(db),
		experience:     NewExperienceManager(db),
		env:            NewEnvManager(),
		scripts:        NewScriptManager(db),
		store:          store,
		externalDBPool: externalDBPool,
		fastClassifier: NewFastPathClassifier(db),
	}
}

// Close closes resources owned by the engine, including cached external datasource pools.
func (e *AgentEngine) Close() error {
	if e == nil || e.externalDBPool == nil {
		return nil
	}
	return e.externalDBPool.CloseAll()
}

// UpdateConversationMemory updates compressed conversation state and task working memory after a turn.
func (e *AgentEngine) UpdateConversationMemory(user *UserContext, conversationID, userQuery, assistantReply string) error {
	if e == nil || e.memory == nil {
		return nil
	}
	err := e.memory.UpdateConversationMemoryWithSummary(user, conversationID, userQuery, assistantReply, "")
	if err == nil {
		go e.compactConversationSummaryAsync(user, conversationID)
	}
	return err
}

func (e *AgentEngine) compactConversationSummaryAsync(user *UserContext, conversationID string) {
	compressed := e.compressConversationSummaryWithLLM(user, conversationID)
	if compressed == "" || e == nil || e.memory == nil || e.memory.db == nil {
		return
	}
	_ = e.memory.db.Model(&models.ConversationState{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, user.UserID).
		Update("summary", compressed).Error
}

func (e *AgentEngine) compressConversationSummaryWithLLM(user *UserContext, conversationID string) string {
	if e == nil || e.memory == nil || e.providers == nil || user == nil || conversationID == "" {
		return ""
	}
	state, err := e.memory.getConversationWorkingState(user.UserID, conversationID)
	if err != nil {
		return ""
	}
	if len([]rune(state.Summary)) < 700 {
		return ""
	}

	provider, err := e.providers.GetActiveProvider()
	if err != nil || provider == nil {
		return ""
	}
	client, err := llm.NewClient(provider)
	if err != nil {
		return ""
	}

	prompt := templating.SafeRender(conversationSummaryCompressJinja, map[string]interface{}{"summary": state.Summary}, fmt.Sprintf("请压缩以下对话摘要到 500 字以内：\n%s", state.Summary))

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	resp, err := client.ChatCompletion(ctx, llm.ChatRequest{
		Model:       provider.Model,
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   500,
		Temperature: 0.1,
	})
	if err != nil {
		return ""
	}
	return strings.TrimSpace(resp.Content)
}

func (e *AgentEngine) LearnRouteExampleCandidate(text, route, intent string, confidence float64) {
	e.learnRouteExampleCandidate(text, route, intent, confidence)
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

	// 注入命中的 Skill 配置（用户自定义 Skill 的 MD/数据源/关系等）
	if user.Context == nil {
		user.Context = make(map[string]interface{})
	}
	if skillID, ok := intent.Parameters["skill_id"].(string); ok && skillID != "" {
		var skill models.Skill
		if e.db.Where("id = ?", skillID).First(&skill).Error == nil {
			user.Context["selected_skill_id"] = skill.ID
			user.Context["selected_skill_name"] = skill.Name
			user.Context["selected_skill_type"] = skill.SkillType
			user.Context["selected_skill_config"] = skill.Config
			if skill.Config != "" {
				var cfg map[string]interface{}
				if json.Unmarshal([]byte(skill.Config), &cfg) == nil {
					if dsID, ok := cfg["data_source_id"].(string); ok && dsID != "" {
						user.Context["data_source_id"] = dsID
					}
				}
			}
		}
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
	if err := e.db.Where("id = ?", userID).Preload("Groups").First(&user).Error; err != nil {
		return nil, err
	}

	ctx := &UserContext{
		UserID:       user.ID,
		GlobalUserID: user.GlobalUserID,
		Name:         user.Name,
		Department:   user.Department,
		Role:         user.Role,
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
	e.db.Where("user_id = ?", userID).Or("group_id IN ?", ctx.getGroupIDs()).Find(&skillAccess)
	for _, s := range skillAccess {
		ctx.AvailableSkills = append(ctx.AvailableSkills, s.SkillID)
	}

	return ctx, nil
}

// recognizeIntent 意图识别：关键词匹配优先 → 向量语义匹配兜底
// 不做开放式 fallback，无匹配时返回 boundary_reject
func (e *AgentEngine) recognizeIntent(message string, user *UserContext) (*IntentResult, error) {
	lowerMsg := toLower(message)

	skills, err := e.skills.ListEnabledSkills()
	if err != nil {
		return nil, fmt.Errorf("获取 Skill 列表失败: %w", err)
	}

	// === 两阶段路由 ===
	// Stage 1: 对所有 Skill 打分
	candidates := scoreSkills(message, lowerMsg, skills)

	// 无任何匹配
	if len(candidates) == 0 {
		return &IntentResult{
			Intent:     "boundary_reject",
			Confidence: 0,
		}, nil
	}

	// 最佳候选
	best := candidates[0]

	// 高置信：直接路由
	if best.Score >= 0.7 {
		return &IntentResult{
			Intent:     best.SkillType,
			Confidence: best.Score,
			Parameters: map[string]interface{}{"skill_name": best.SkillName, "skill_id": best.SkillID, "match_type": best.MatchType},
		}, nil
	}

	// 中置信 + 与第二名差距大：自动路由
	if best.Score >= 0.4 && (len(candidates) < 2 || best.Score-candidates[1].Score > 0.25) {
		return &IntentResult{
			Intent:     best.SkillType,
			Confidence: best.Score,
			Parameters: map[string]interface{}{"skill_name": best.SkillName, "skill_id": best.SkillID, "match_type": best.MatchType},
		}, nil
	}

	// 低置信或接近：返回候选列表
	return &IntentResult{
		Intent:     "needs_disambiguation",
		Confidence: best.Score,
		Candidates: candidates,
		Parameters: map[string]interface{}{"reason": "ambiguous_query"},
	}, nil
}

// parseKeywords 解析 Skill 的关键词 JSON 数组
// 处理多层嵌套的 JSON 转义（前端多次保存导致的累积错误）
func parseKeywords(keywordsJSON string) []string {
	if keywordsJSON == "" {
		return nil
	}

	// 递归展开被多次 JSON.stringify 嵌套的值
	current := strings.TrimSpace(keywordsJSON)
	for depth := 0; depth < 10; depth++ {
		// 尝试直接解析为 []string
		var keywords []string
		if err := json.Unmarshal([]byte(current), &keywords); err == nil {
			// 检查元素是否仍包含可解析的 JSON 字符串（需要进一步展开）
			result := make([]string, 0, len(keywords))
			hasNested := false
			for _, k := range keywords {
				// 尝试将元素作为 JSON 字符串解析（检查是否有额外引号）
				var inner string
				if json.Unmarshal([]byte(k), &inner) == nil && inner != k {
					// 元素是嵌套的 JSON 字符串，展开后收集
					hasNested = true
					subResult := parseKeywords(inner)
					result = append(result, subResult...)
				} else {
					// 去除可能的残留外层引号
					cleaned := strings.Trim(k, `"`)
					if cleaned != "" {
						result = append(result, cleaned)
					}
				}
			}
			if !hasNested {
				return result
			}
			// 有嵌套，用展开后的结果继续
			if len(result) > 0 {
				return result
			}
			return keywords
		}

		// 尝试作为单个 JSON 字符串解析（整个值被额外引号包裹）
		var inner string
		if err := json.Unmarshal([]byte(current), &inner); err == nil {
			current = inner
			continue
		}
		break
	}

	// 兜底：逗号分隔
	return strings.Split(current, ",")
}

// SkillCandidate 技能候选（两阶段路由用）
type SkillCandidate struct {
	SkillID     string  `json:"skill_id"`
	SkillName   string  `json:"skill_name"`
	SkillType   string  `json:"skill_type"`
	Score       float64 `json:"score"`
	MatchType   string  `json:"match_type"`
	Description string  `json:"description"`
}

// scoreSkills 对所有启用的 Skill 匹配打分，返回按分数降序的候选列表
func scoreSkills(message, lowerMsg string, skills []models.Skill) []SkillCandidate {
	type resultItem struct {
		SkillCandidate
	}
	var list []resultItem

	for _, sk := range skills {
		keywords := parseKeywords(sk.Keywords)
		score := 0.0
		matchType := ""
		totalKW := len(keywords)

		// 关键词完全匹配（query contains keyword）: +1.0 each
		for _, kw := range keywords {
			if kw != "" && contains(lowerMsg, toLower(kw)) {
				score += 1.0
				matchType = "keyword"
			}
		}

		// Skill 名称匹配: +0.8
		if contains(lowerMsg, toLower(sk.Name)) {
			score += 0.8
			matchType = "keyword"
		}

		// Skill 类型匹配: +0.6
		if contains(lowerMsg, toLower(sk.SkillType)) {
			score += 0.6
			matchType = "keyword"
		}

		// 归一化：总数 / (总关键词数 + 2)，上限 1.0
		if score > 0 {
			normalized := score / float64(totalKW+2)
			if normalized > 1.0 {
				normalized = 1.0
			}
			list = append(list, resultItem{SkillCandidate: SkillCandidate{
				SkillID: sk.ID, SkillName: sk.Name, SkillType: sk.SkillType,
				Score: normalized, MatchType: matchType,
				Description: sk.Description,
			}})
		}
	}

	// 按分数降序
	sort.Slice(list, func(i, j int) bool { return list[i].Score > list[j].Score })

	// 最多返回 3 个候选
	result := make([]SkillCandidate, 0, min(3, len(list)))
	for i := 0; i < len(list) && i < 3; i++ {
		result = append(result, list[i].SkillCandidate)
	}
	return result
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
		"excel":        "skill_report",
		"word":         "skill_report",
		"ppt":          "skill_report",
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

	// 管理员角色直接放行
	if user.Role == "admin" {
		return true, ""
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

	externalDB, err := e.externalDBPool.Get(context.Background(), dataSourceID)
	if err != nil {
		return &ChatResponse{
			Message: "数据库暂时不可用，请稍后重试或联系管理员检查数据源配置",
			Data: map[string]interface{}{
				"type":  "text2sql",
				"error": "datasource_unavailable",
			},
		}, nil
	}
	t2s := text2sql.NewWithExternalDB(e.db, client, dataSourceID, externalDB)

	// 注入 SQL 审计服务
	if e.sqlAuditor != nil {
		t2s.SetAuditService(e.sqlAuditor)
	}

	// 确定是否为管理员
	isAdmin := user.Role == "admin"

	// 获取选中的 Skill ID
	skillID := ""
	if user.Context != nil {
		if sid, ok := user.Context["selected_skill_id"].(string); ok {
			skillID = sid
		}
	}

	schemaContext := buildText2SQLSkillSchemaContext(user)
	if runtimeContext := e.buildText2SQLRuntimeContext(skillID, dataSourceID); runtimeContext != "" {
		schemaContext += runtimeContext
	}

	boundaryRules := parseDataBoundaryRulesFromUserContext(user)
	// 执行查询
	req := &text2sql.QueryRequest{
		Question:          message,
		DataSourceID:      dataSourceID,
		SchemaContext:     schemaContext,
		UserID:            user.UserID,
		SkillID:           skillID,
		IsAdmin:           isAdmin,
		DataBoundaryRules: boundaryRules,
		UserContext:       buildBoundaryUserContext(user),
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

	go e.learnText2SQLRuntime(skillID, dataSourceID, message, result.SQL, result.Columns, result.RowCount)

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

	// 从共享连接池获取外部数据源连接，避免每次请求重复创建数据库连接池
	externalDB, err := e.externalDBPool.Get(context.Background(), dataSourceID)
	if err != nil {
		return &ChatResponse{
			Message: fmt.Sprintf("连接数据源失败: %v", err),
			Data: map[string]interface{}{
				"type":  "employee",
				"error": err.Error(),
			},
		}, nil
	}
	t2s := text2sql.NewWithExternalDB(e.db, client, dataSourceID, externalDB)

	// 注入 SQL 审计服务
	if e.sqlAuditor != nil {
		t2s.SetAuditService(e.sqlAuditor)
	}

	// 确定是否为管理员
	isAdmin := user.Role == "admin"

	// 获取选中的 Skill ID
	skillID := ""
	if user.Context != nil {
		if sid, ok := user.Context["selected_skill_id"].(string); ok {
			skillID = sid
		}
	}

	schemaContext := buildText2SQLSkillSchemaContext(user)
	if runtimeContext := e.buildText2SQLRuntimeContext(skillID, dataSourceID); runtimeContext != "" {
		schemaContext += runtimeContext
	}

	// 为员工查询构建特殊的 prompt
	req := &text2sql.QueryRequest{
		Question:      message,
		DataSourceID:  dataSourceID,
		SchemaContext: schemaContext,
		UserID:        user.UserID,
		SkillID:       skillID,
		IsAdmin:       isAdmin,
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

	go e.learnText2SQLRuntime(skillID, dataSourceID, message, result.SQL, result.Columns, result.RowCount)

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

func parseDataBoundaryRulesFromUserContext(user *UserContext) []text2sql.DataBoundaryRule {
	if user == nil || user.Context == nil {
		return nil
	}
	raw, _ := user.Context["selected_skill_config"].(string)
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var cfg struct {
		Rules []text2sql.DataBoundaryRule `json:"data_boundary_rules"`
	}
	if json.Unmarshal([]byte(raw), &cfg) != nil {
		return nil
	}
	return cfg.Rules
}

func buildBoundaryUserContext(user *UserContext) map[string]interface{} {
	ctx := map[string]interface{}{}
	if user == nil {
		return ctx
	}
	ctx["user_id"] = user.UserID
	ctx["global_user_id"] = user.GlobalUserID
	ctx["company_user_id"] = user.GlobalUserID
	ctx["name"] = user.Name
	ctx["department"] = user.Department
	ctx["role"] = user.Role
	var groupIDs, groupNames, groupCodes []string
	for _, g := range user.Groups {
		groupIDs = append(groupIDs, g.ID)
		groupNames = append(groupNames, g.Name)
		groupCodes = append(groupCodes, g.Code)
	}
	ctx["group_ids"] = groupIDs
	ctx["group_names"] = groupNames
	ctx["group_codes"] = groupCodes
	return ctx
}

// formatEmployeeResult 格式化员工查询结果
func buildText2SQLSkillSchemaContext(user *UserContext) string {
	if user == nil || user.Context == nil {
		return ""
	}
	raw, _ := user.Context["selected_skill_config"].(string)
	if raw == "" {
		return ""
	}
	var cfg map[string]interface{}
	if json.Unmarshal([]byte(raw), &cfg) != nil {
		return ""
	}

	if contextMD, ok := cfg["context_md"].(string); ok && strings.TrimSpace(contextMD) != "" {
		return contextMD
	}
	if contextURL, ok := cfg["context_md_url"].(string); ok && strings.TrimSpace(contextURL) != "" {
		if md := fetchSkillContextMD(contextURL); md != "" {
			return md
		}
	}

	var sb strings.Builder
	if entryTable, ok := cfg["entry_table"].(string); ok && strings.TrimSpace(entryTable) != "" {
		sb.WriteString("Text2SQL 入口表（主事实表，生成 SQL 时优先从此表开始）：\n")
		sb.WriteString("表: " + entryTable + "\n\n")
	}
	if metrics, ok := cfg["metric_rules"].([]interface{}); ok && len(metrics) > 0 {
		sb.WriteString("Text2SQL 指标规则：\n")
		for _, item := range metrics {
			if b, err := json.Marshal(item); err == nil {
				sb.WriteString("  - " + string(b) + "\n")
			}
		}
		sb.WriteString("\n")
	}
	if entities, ok := cfg["entity_rules"].([]interface{}); ok && len(entities) > 0 {
		sb.WriteString("Text2SQL 实体规则：\n")
		for _, item := range entities {
			if b, err := json.Marshal(item); err == nil {
				sb.WriteString("  - " + string(b) + "\n")
			}
		}
		sb.WriteString("\n")
	}
	if selected, ok := cfg["selected_tables"].([]interface{}); ok && len(selected) > 0 {
		sb.WriteString("Text2SQL Skill 设计期候选表（运行时优先使用）：\n")
		for _, item := range selected {
			switch table := item.(type) {
			case string:
				sb.WriteString("表: " + table + "\n")
			case map[string]interface{}:
				name, _ := table["name"].(string)
				if name == "" {
					continue
				}
				sb.WriteString("表: " + name + "\n")
				if cols, ok := table["columns"].([]interface{}); ok {
					for _, col := range cols {
						sb.WriteString(fmt.Sprintf("  - %v\n", col))
					}
				}
			}
			sb.WriteString("\n")
		}
	}
	if relations, ok := cfg["table_relations"].([]interface{}); ok && len(relations) > 0 {
		sb.WriteString("设计期确认的表关系：\n")
		for _, item := range relations {
			if rel, ok := item.(map[string]interface{}); ok {
				sb.WriteString(fmt.Sprintf("  %v.%v → %v.%v\n", rel["from_table"], rel["from_column"], rel["to_table"], rel["to_column"]))
			}
		}
		sb.WriteString("\n")
	}
	if rules, ok := cfg["business_rules"].(string); ok && strings.TrimSpace(rules) != "" {
		sb.WriteString("业务口径：\n" + rules + "\n")
	}
	return strings.TrimSpace(sb.String())
}

func fetchSkillContextMD(url string) string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return ""
	}
	return string(data)
}

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

	// 使用共享连接池获取数据，避免每次报表生成重复创建数据库连接池
	externalDB, err := e.externalDBPool.Get(context.Background(), dataSourceID)
	if err != nil {
		return &ChatResponse{
			Message: fmt.Sprintf("连接数据源失败: %v", err),
		}, nil
	}
	t2s := text2sql.NewWithExternalDB(e.db, client, dataSourceID, externalDB)

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

	// 生成 PDF (使用 gofpdf)
	ctx := context.Background()
	downloadURL, pdfErr := e.generateReportPDF(ctx, message, result.Columns, result.Rows)
	if pdfErr != nil {
		return &ChatResponse{
			Message: fmt.Sprintf("PDF 生成失败: %v", pdfErr),
			Data: map[string]interface{}{
				"type":           "pdf",
				"sql":            result.SQL,
				"columns":        result.Columns,
				"rows":           result.Rows,
				"row_count":      result.RowCount,
				"data_source_id": dataSourceID,
				"error":          pdfErr.Error(),
			},
			SkillUsed: "skill_pdf",
		}, nil
	}

	message = fmt.Sprintf("已查询到 %d 条数据，PDF 报表已生成\n\n[📄 下载 PDF 报表](%s)", result.RowCount, downloadURL)

	return &ChatResponse{
		Message: message,
		Data: map[string]interface{}{
			"type":           "pdf",
			"sql":            result.SQL,
			"columns":        result.Columns,
			"rows":           result.Rows,
			"row_count":      result.RowCount,
			"data_source_id": dataSourceID,
			"download_url":   downloadURL,
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
	Role            string
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
	if s[:len(substr)] == substr {
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

func (m *ProviderManager) GetProviderByRole(role string) (*models.Provider, error) {
	if role == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var providers []models.Provider
	if err := m.db.Where("enabled = ?", true).Order("priority ASC").Find(&providers).Error; err != nil {
		return nil, err
	}
	for _, provider := range providers {
		var cfg map[string]interface{}
		if json.Unmarshal([]byte(provider.Config), &cfg) == nil {
			if r, ok := cfg["role"].(string); ok && r == role {
				return &provider, nil
			}
		}
	}
	return nil, gorm.ErrRecordNotFound
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
	err := m.db.Where("enabled = ?", true).
		Order("CASE WHEN category = '系统内置' THEN 1 ELSE 0 END, created_at DESC").
		Find(&skills).Error
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
	db       *gorm.DB
	longTerm *LongTermMemory
}

func NewMemoryManager(db *gorm.DB) *MemoryManager {
	return &MemoryManager{db: db, longTerm: NewLongTermMemory(db)}
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
	memories, err := m.longTerm.GetUserMemory(userID, memoryType)
	if err != nil || len(memories) == 0 {
		return "", err
	}
	var sb strings.Builder
	for _, mem := range memories {
		sb.WriteString(mem.Key + ": " + mem.Content + "\n")
	}
	return sb.String(), nil
}

func (m *MemoryManager) SaveUserMemory(userID, memoryType, content string) error {
	key := "user_" + memoryType
	return m.longTerm.SaveUserMemory(userID, memoryType, key, content, 1)
}

func (m *MemoryManager) GetGroupMemory(groupID, memoryType string) (string, error) {
	memories, err := m.longTerm.GetGroupMemories([]string{groupID})
	if err != nil || len(memories) == 0 {
		return "", err
	}
	var sb strings.Builder
	for _, mem := range memories {
		sb.WriteString(mem.Key + ": " + mem.Content + "\n")
	}
	return sb.String(), nil
}

func (m *MemoryManager) SaveGroupMemory(groupID, memoryType, content string) error {
	key := "group_" + memoryType
	return m.longTerm.SaveGroupMemory(groupID, memoryType, key, content, 0)
}

// SaveConversationMemory 对话结束后自动沉淀记忆
func (m *MemoryManager) SaveConversationMemory(userID, userQuery, assistantReply string, groupIDs []string) {
	summary := fmt.Sprintf("Q: %s A: %s", truncateStr(userQuery, 200), truncateStr(assistantReply, 300))
	m.longTerm.SaveUserMemory(userID, "conversation", fmt.Sprintf("conv_%d", time.Now().Unix()), summary, 0)
	for _, gid := range groupIDs {
		topic := extractTopic(userQuery)
		if topic != "" {
			m.longTerm.SaveGroupMemory(gid, "topic", topic, userQuery, 5)
		}
	}
}

func extractTopic(query string) string {
	for _, kw := range []string{"订单", "销售", "库存", "员工", "成本", "客户", "采购", "报表", "产品", "利润", "业绩", "部门", "仓库", "物流", "广告"} {
		if strings.Contains(query, kw) {
			return kw
		}
	}
	return ""
}

var _ = json.Marshal // use json
