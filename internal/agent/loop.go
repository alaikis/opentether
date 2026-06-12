package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/llm"
	"github.com/alaikis/opentether/internal/models"
	"github.com/alaikis/opentether/internal/templating"
)

// ============================================
// Agentic Loop - ReAct 模式多步推理执行
// ============================================

const (
	LoopTimeout = time.Hour
)

// LoopContext 循环执行上下文
type LoopContext struct {
	UserID         string
	ConversationID string
	OriginalQuery  string
	Memory         *ConversationMemoryContext
	History        []LoopStep
	Observations   []string
	Iteration      int
	StartTime      time.Time
}

// LoopStep 循环中的每一步
type LoopStep struct {
	StepID     int                    `json:"step"`
	Action     string                 `json:"action"`  // think, tool_call, final_answer
	Thought    string                 `json:"thought"` // LLM 推理过程
	ToolName   string                 `json:"tool_name,omitempty"`
	ToolInput  map[string]interface{} `json:"tool_input,omitempty"`
	ToolOutput string                 `json:"tool_output,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// LoopEvent 循环执行过程中的实时事件（用于流式输出）
type LoopEvent struct {
	Type    string                 `json:"type"`    // "thinking", "tool_start", "tool_result", "final_token", "error"
	Content string                 `json:"content"` // 事件内容
	Data    map[string]interface{} `json:"data,omitempty"`
}

// sendEvent 非阻塞发送事件到 channel
func sendEvent(events chan<- LoopEvent, evt LoopEvent) {
	if events == nil {
		return
	}
	select {
	case events <- evt:
	default:
	}
}

// ExecuteLoop 执行 ReAct 循环（向后兼容，内部调用 ExecuteLoopWithEvents）
func (e *AgentEngine) ExecuteLoop(ctx context.Context, user *UserContext, query string, conversationID string) (*ChatResponse, error) {
	return e.ExecuteLoopWithEvents(ctx, user, query, conversationID, nil)
}

// ExecuteLoopWithEvents 执行 ReAct 循环并通过 events channel 发送实时进度事件
// events 若为 nil 则行为与 ExecuteLoop 一致
func (e *AgentEngine) ExecuteLoopWithEvents(ctx context.Context, user *UserContext, query string, conversationID string, events chan<- LoopEvent) (*ChatResponse, error) {
	// 函数退出时关闭事件 channel
	if events != nil {
		defer close(events)
	}

	var conversationMemory *ConversationMemoryContext
	if e.memory != nil {
		var memErr error
		conversationMemory, memErr = e.memory.LoadConversationMemory(user.UserID, conversationID, query, conversationWindowMessages)
		if memErr != nil {
			log.Printf("[Memory] 加载对话记忆失败: %v", memErr)
		}
	}

	if conversationMemory != nil && conversationMemory.Route.Action != "" && conversationMemory.Route.Action != "continue" {
		sendEvent(events, LoopEvent{
			Type:    "topic_route",
			Content: conversationMemory.Route.Reason,
			Data: map[string]interface{}{
				"action":       conversationMemory.Route.Action,
				"from_task_id": conversationMemory.Route.FromTaskID,
				"to_task_id":   conversationMemory.Route.ToTaskID,
			},
		})
	}

	loop := &LoopContext{
		UserID:         user.UserID,
		ConversationID: conversationID,
		OriginalQuery:  query,
		Memory:         conversationMemory,
		History:        make([]LoopStep, 0),
		Observations:   make([]string, 0),
		StartTime:      time.Now(),
	}

	// 获取活跃的 LLM provider
	provider, err := e.providers.GetActiveProvider()
	if err != nil {
		return e.fallbackResponse(conversationID, query, err)
	}

	// === 经验复用：检查是否有匹配的已激活经验 ===
	if e.experience != nil {
		exp, score, _ := e.experience.MatchExperience(user.UserID, query)
		if exp != nil && score >= 0.6 {
			log.Printf("[Loop] 命中经验 %s (分数: %.2f), 跳过 LLM 推理", exp.Name, score)
			resp, err := e.experience.ExecuteExperience(ctx, e, user, exp)
			if err == nil {
				resp.ConversationID = conversationID
				return resp, nil
			}
			log.Printf("[Loop] 经验执行失败: %v, 回退到正常推理", err)
		}
	}

	// 获取可用工具列表（受用户权限限制）
	availableTools := e.getAvailableTools(user)
	toolNames := make(map[string]bool)
	for _, t := range availableTools {
		toolNames[t.Name] = true
	}

	// 构建系统 prompt（含可用工具描述 + 边界约束 + 长期记忆 + 当前问题召回）
	systemPrompt := e.buildSystemPrompt(availableTools, user, query)

	// 创建 LLM client
	llmClient, err := llm.NewClient(provider)
	if err != nil {
		return e.fallbackResponse(conversationID, query, err)
	}

	totalTokens := 0

	maxIterations := e.config.Executor.EmbeddedConfig.MaxLoopIterations
	if maxIterations <= 0 {
		maxIterations = 1<<31 - 1 // 0 表示不限制，设为极大值
	}
	runtimeJob := e.startRuntimeJob(user, conversationID, query, maxIterations)

	for loop.Iteration = 0; loop.Iteration < maxIterations; loop.Iteration++ {
		e.heartbeatRuntimeJob(runtimeJob, loop.Iteration)
		// 超时检查
		if time.Since(loop.StartTime) > LoopTimeout {
			timeoutStep := LoopStep{
				StepID: loop.Iteration,
				Action: "error",
				Error:  "执行超时",
			}
			loop.History = append(loop.History, timeoutStep)
			e.saveRuntimeCheckpoint(runtimeJob, loop.Iteration, "timeout", timeoutStep)
			e.finishRuntimeJob(runtimeJob, "failed", "", "执行超时")
			sendEvent(events, LoopEvent{Type: "error", Content: "执行超时"})
			break
		}

		// 构建本轮消息
		messages := e.buildLoopMessages(systemPrompt, query, loop)

		// 调用 LLM 推理下一步
		resp, err := llmClient.ChatCompletion(ctx, llm.ChatRequest{
			Model:       provider.Model,
			Messages:    messages,
			MaxTokens:   512,
			Temperature: 0.3,
		})
		if err != nil {
			step := LoopStep{
				StepID: loop.Iteration,
				Action: "error",
				Error:  err.Error(),
			}
			loop.History = append(loop.History, step)
			e.saveRuntimeCheckpoint(runtimeJob, loop.Iteration, "llm_error", step)
			e.finishRuntimeJob(runtimeJob, "failed", "", err.Error())
			sendEvent(events, LoopEvent{Type: "error", Content: fmt.Sprintf("LLM 调用失败: %v", err)})
			break
		}

		totalTokens += resp.Usage.TotalTokens

		// 解析 LLM 决策
		decision, parseErr := parseLoopDecision(resp.Content)
		if parseErr != nil {
			log.Printf("[Loop] 解析决策失败: %v, raw=%.200s", parseErr, resp.Content)
			e.saveRuntimeCheckpoint(runtimeJob, loop.Iteration, "parse_error", map[string]interface{}{"error": parseErr.Error(), "raw": resp.Content})
			e.finishRuntimeJob(runtimeJob, "failed", resp.Content, parseErr.Error())
			sendEvent(events, LoopEvent{Type: "error", Content: fmt.Sprintf("解析决策失败: %v", parseErr)})
			return &ChatResponse{
				Message:        resp.Content,
				ConversationID: conversationID,
				TokensUsed:     totalTokens,
			}, nil
		}

		// 发送 thinking 事件
		sendEvent(events, LoopEvent{Type: "thinking", Content: decision.Thought})

		step := LoopStep{
			StepID:  loop.Iteration,
			Action:  decision.Action,
			Thought: decision.Thought,
		}

		switch decision.Action {
		case "final_answer":
			loop.History = append(loop.History, step)
			e.saveRuntimeCheckpoint(runtimeJob, loop.Iteration, "final_answer", map[string]interface{}{"step": step, "answer": decision.FinalAnswer})
			e.finishRuntimeJob(runtimeJob, "succeeded", decision.FinalAnswer, "")

			// 发送 final_token 事件
			sendEvent(events, LoopEvent{Type: "final_token", Content: decision.FinalAnswer})

			// === 经验积累：多步操作自动保存为待审核经验 ===
			if e.experience != nil {
				e.experience.TrySaveExperience(user.UserID, query, loop.History, totalTokens)
			}

			return &ChatResponse{
				Message:        decision.FinalAnswer,
				ConversationID: conversationID,
				SkillUsed:      "agent_loop",
				TokensUsed:     totalTokens,
				Data: map[string]interface{}{
					"iterations": loop.Iteration + 1,
					"history":    loop.History,
				},
			}, nil

		case "tool_call":
			step.ToolName = decision.ToolName
			step.ToolInput = decision.ToolInput

			e.saveRuntimeCheckpoint(runtimeJob, loop.Iteration, "tool_call", step)

			// 发送 tool_start 事件
			sendEvent(events, LoopEvent{
				Type:    "tool_start",
				Content: decision.ToolName,
				Data:    map[string]interface{}{"tool_name": decision.ToolName, "tool_input": decision.ToolInput},
			})

			// === 边界检查：工具是否在用户权限范围内 ===
			if !toolNames[decision.ToolName] {
				log.Printf("[Loop] 拒绝未授权工具: %s (user=%s)", decision.ToolName, user.UserID)
				step.Error = fmt.Sprintf("工具 %s 不在可用范围内", decision.ToolName)
				step.ToolOutput = fmt.Sprintf("[系统] 工具 %s 不可用，你只能使用: %v。请用已有工具或直接回答。", decision.ToolName, availableToolNames(availableTools))
				observation := fmt.Sprintf("[边界检查] 拒绝工具 %s: 不在用户权限范围内", decision.ToolName)
				loop.Observations = append(loop.Observations, observation)
				loop.History = append(loop.History, step)
				e.saveRuntimeCheckpoint(runtimeJob, loop.Iteration, "tool_rejected", step)
				continue
			}

			toolOutput, toolErr := e.executeTool(ctx, user, decision.ToolName, decision.ToolInput)
			if toolErr != nil {
				step.Error = toolErr.Error()
				step.ToolOutput = fmt.Sprintf("工具执行失败: %v", toolErr)
			} else {
				step.ToolOutput = toolOutput
			}

			// 发送 tool_result 事件
			sendEvent(events, LoopEvent{Type: "tool_result", Content: step.ToolOutput})

			observation := fmt.Sprintf("[工具 %s 执行结果]: %s", decision.ToolName, step.ToolOutput)
			loop.Observations = append(loop.Observations, observation)
			loop.History = append(loop.History, step)
			e.saveRuntimeCheckpoint(runtimeJob, loop.Iteration, "tool_result", step)

		case "parallel_calls":
			// === 并行执行：多个互不依赖的工具调用同时执行 ===
			if len(decision.ParallelCalls) == 0 {
				step.Error = "parallel_calls 需要提供 calls 列表"
				step.ToolOutput = "[系统] parallel_calls 缺少调用列表，请提供互不依赖的多个工具调用"
				loop.History = append(loop.History, step)
				continue
			}

			log.Printf("[Loop] 并行执行 %d 个工具调用", len(decision.ParallelCalls))

			// 发送 tool_start 事件（每个并行调用一个）
			for _, call := range decision.ParallelCalls {
				sendEvent(events, LoopEvent{
					Type:    "tool_start",
					Content: call.ToolName,
					Data:    map[string]interface{}{"tool_name": call.ToolName, "tool_input": call.ToolInput},
				})
			}

			e.saveRuntimeCheckpoint(runtimeJob, loop.Iteration, "parallel_calls", decision.ParallelCalls)
			parallelSteps := e.executeParallelCalls(ctx, user, decision.ParallelCalls, toolNames)

			// 发送 tool_result 事件（每个结果一个）
			for _, ps := range parallelSteps {
				sendEvent(events, LoopEvent{Type: "tool_result", Content: ps.ToolOutput})
			}

			observation := formatParallelResults(parallelSteps)
			loop.Observations = append(loop.Observations, observation)
			loop.History = append(loop.History, parallelSteps...)
			e.saveRuntimeCheckpoint(runtimeJob, loop.Iteration, "parallel_results", parallelSteps)

		case "clarify":
			// === 参数补全：向用户提问，等待下一轮对话补充参数 ===
			loop.History = append(loop.History, step)
			e.saveRuntimeCheckpoint(runtimeJob, loop.Iteration, "clarify", map[string]interface{}{"step": step, "question": decision.ClarifyQuestion})
			e.finishRuntimeJob(runtimeJob, "paused", decision.ClarifyQuestion, "等待用户补充参数")
			question := decision.ClarifyQuestion
			if question == "" {
				question = "请提供更多信息以完成任务"
			}
			return &ChatResponse{
				Message:        question,
				ConversationID: conversationID,
				SkillUsed:      "agent_loop",
				TokensUsed:     totalTokens,
				Data: map[string]interface{}{
					"needs_clarification": true,
					"iterations":          loop.Iteration + 1,
					"history":             loop.History,
				},
			}, nil

		case "confirm":
			// === 写入确认：生成确认文档，等待用户批准后再执行 ===
			loop.History = append(loop.History, step)
			e.saveRuntimeCheckpoint(runtimeJob, loop.Iteration, "confirm", map[string]interface{}{"step": step, "confirm_doc": decision.ConfirmDoc})
			doc := decision.ConfirmDoc
			if doc == nil {
				step.Error = "confirm 需要提供 confirm_doc"
				loop.History = append(loop.History, step)
				e.saveRuntimeCheckpoint(runtimeJob, loop.Iteration, "confirm_error", step)
				continue
			}
			e.finishRuntimeJob(runtimeJob, "paused", "", "等待用户确认写操作")
			return &ChatResponse{
				Message:        formatConfirmMessage(doc),
				ConversationID: conversationID,
				SkillUsed:      "agent_loop",
				TokensUsed:     totalTokens,
				Data: map[string]interface{}{
					"needs_confirmation": true,
					"confirm_doc":        doc,
					"iterations":         loop.Iteration + 1,
					"history":            loop.History,
				},
			}, nil

		default:
			loop.History = append(loop.History, step)
			e.saveRuntimeCheckpoint(runtimeJob, loop.Iteration, "fallback", map[string]interface{}{"step": step, "raw": resp.Content})
			e.finishRuntimeJob(runtimeJob, "succeeded", resp.Content, "")
			return &ChatResponse{
				Message:        resp.Content,
				ConversationID: conversationID,
				SkillUsed:      "chat",
				TokensUsed:     totalTokens,
			}, nil
		}
	}

	// 达到最大迭代次数，生成总结
	summary := fmt.Sprintf("已完成 %d 步分析。查询: %s。如需更详细的信息，请提供更具体的问题。", loop.Iteration, query)
	e.saveRuntimeCheckpoint(runtimeJob, loop.Iteration, "max_iterations", map[string]interface{}{"summary": summary})
	e.finishRuntimeJob(runtimeJob, "succeeded", summary, "")

	return &ChatResponse{
		Message:        summary,
		ConversationID: conversationID,
		SkillUsed:      "agent_loop",
		TokensUsed:     totalTokens,
		Data: map[string]interface{}{
			"iterations":  loop.Iteration,
			"max_reached": true,
		},
	}, nil
}

// LoopDecision LLM 返回的决策结构
type LoopDecision struct {
	Action          string                 `json:"action"`  // tool_call | final_answer | parallel_calls | clarify | confirm
	Thought         string                 `json:"thought"` // 推理过程
	ToolName        string                 `json:"tool_name,omitempty"`
	ToolInput       map[string]interface{} `json:"tool_input,omitempty"`
	FinalAnswer     string                 `json:"final_answer,omitempty"`
	ParallelCalls   []ParallelCall         `json:"calls,omitempty"`            // 并行调用列表
	ClarifyQuestion string                 `json:"clarify_question,omitempty"` // 向用户提问
	ConfirmDoc      *ConfirmDocument       `json:"confirm_doc,omitempty"`      // 确认文档
}

// ConfirmDocument 需要用户确认的操作文档
type ConfirmDocument struct {
	Title       string      `json:"title"`       // 操作标题
	Description string      `json:"description"` // 操作描述
	Operations  []ConfirmOp `json:"operations"`  // 具体操作列表
	Risk        string      `json:"risk"`        // 风险等级: low, medium, high
}

// ConfirmOp 单个确认操作
type ConfirmOp struct {
	Type    string `json:"type"`    // insert, update, delete, execute
	Target  string `json:"target"`  // 目标: 表名/资源名
	Details string `json:"details"` // 操作详情
}

// parseLoopDecision 从 LLM 输出解析决策 JSON。
// 容错策略：JSON code block → JSON 对象 → 修复截断 JSON → 纯文本 final_answer 兜底。
func parseLoopDecision(content string) (*LoopDecision, error) {
	original := strings.TrimSpace(content)
	if decision, ok := parseFunctionStyleDecision(original); ok {
		return decision, nil
	}
	candidate := extractDecisionJSONCandidate(original)
	if candidate == "" {
		return fallbackDecisionFromText(original), nil
	}

	var decision LoopDecision
	if err := json.Unmarshal([]byte(candidate), &decision); err != nil {
		repaired := repairTruncatedDecisionJSON(candidate)
		if repaired != candidate {
			if repairErr := json.Unmarshal([]byte(repaired), &decision); repairErr == nil {
				candidate = repaired
			} else {
				return fallbackDecisionFromText(original), nil
			}
		} else {
			return fallbackDecisionFromText(original), nil
		}
	}

	if decision.Action == "" {
		if decision.FinalAnswer != "" {
			decision.Action = "final_answer"
		} else {
			return fallbackDecisionFromText(original), nil
		}
	}

	// LLM 有时会直接用工具名作为 action（如 action:"employee_query" 而非 action:"tool_call"）
	// 此时自动修正为 tool_call
	if decision.Action != "tool_call" && decision.Action != "final_answer" &&
		decision.Action != "parallel_calls" && decision.Action != "clarify" &&
		decision.Action != "confirm" {
		if decision.ToolName == "" {
			decision.ToolName = decision.Action
		}
		decision.Action = "tool_call"
	}

	return &decision, nil
}

func parseFunctionStyleDecision(content string) (*LoopDecision, bool) {
	if !strings.Contains(content, "<function=") {
		return nil, false
	}
	fnRe := regexp.MustCompile(`<function=([^>]+)>`)
	fnMatch := fnRe.FindStringSubmatch(content)
	if len(fnMatch) < 2 || strings.TrimSpace(fnMatch[1]) == "" {
		return nil, false
	}
	paramRe := regexp.MustCompile(`<parameter=([^>]+)>([^<]*)`)
	params := map[string]interface{}{}
	for _, m := range paramRe.FindAllStringSubmatch(content, -1) {
		if len(m) >= 3 {
			params[strings.TrimSpace(m[1])] = strings.TrimSpace(m[2])
		}
	}
	return &LoopDecision{
		Action:    "tool_call",
		Thought:   "模型返回 function-call 标记，已自动转换为工具调用。",
		ToolName:  strings.TrimSpace(fnMatch[1]),
		ToolInput: params,
	}, true
}

func extractDecisionJSONCandidate(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	if idx := strings.Index(content, "```json"); idx >= 0 {
		start := idx + len("```json")
		if end := strings.Index(content[start:], "```"); end >= 0 {
			return strings.TrimSpace(content[start : start+end])
		}
		return strings.TrimSpace(content[start:])
	}
	if idx := strings.Index(content, "```"); idx >= 0 {
		start := idx + len("```")
		if end := strings.Index(content[start:], "```"); end >= 0 {
			block := strings.TrimSpace(content[start : start+end])
			if strings.HasPrefix(block, "{") {
				return block
			}
		}
	}
	idx := strings.Index(content, "{")
	if idx < 0 {
		return ""
	}
	candidate := strings.TrimSpace(content[idx:])
	if end := strings.LastIndex(candidate, "}"); end >= 0 {
		return strings.TrimSpace(candidate[:end+1])
	}
	return candidate
}

func repairTruncatedDecisionJSON(content string) string {
	repaired := strings.TrimSpace(content)
	if repaired == "" || !strings.HasPrefix(repaired, "{") {
		return content
	}
	repaired = strings.TrimRight(repaired, ", \n\t")
	if countUnescapedQuotes(repaired)%2 == 1 {
		repaired += "\""
	}
	openBraces := strings.Count(repaired, "{") - strings.Count(repaired, "}")
	openBrackets := strings.Count(repaired, "[") - strings.Count(repaired, "]")
	for openBrackets > 0 {
		repaired += "]"
		openBrackets--
	}
	for openBraces > 0 {
		repaired += "}"
		openBraces--
	}
	return repaired
}

func countUnescapedQuotes(s string) int {
	count := 0
	escaped := false
	for _, r := range s {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '"' {
			count++
		}
	}
	return count
}

func fallbackDecisionFromText(text string) *LoopDecision {
	text = strings.TrimSpace(text)
	if text == "" {
		text = "抱歉，本次模型响应为空或被截断，请重新尝试。"
	}
	return &LoopDecision{
		Action:      "final_answer",
		Thought:     "模型未返回严格 JSON，已使用纯文本兜底回答。",
		FinalAnswer: text,
	}
}

// getAvailableTools 从已注册的 Skill 中派生可用工具列表
// 所有工具均来自 skills 表（内置 + 用户自定义），智能体不能自创工具
// 如果用户选中了特定 Skill，只返回该 Skill 对应的工具（精确边界）
func (e *AgentEngine) getAvailableTools(user *UserContext) []ToolDef {
	// 如果用户选中了特定 Skill，只授那个 Skill 对应的工具
	if selectedSkillID, ok := user.Context["selected_skill_id"].(string); ok && selectedSkillID != "" {
		tools := e.getToolsForSkill(selectedSkillID)
		return append(tools, e.getMCPToolDefs()...)
	}

	// 否则返回所有启用的 Skill 实例工具。不要按底层 tool 去重，否则多个 text2sql Skill 会被折叠成一个。
	var skills []models.Skill
	e.db.Where("enabled = ?", true).
		Order("CASE WHEN category = '系统内置' THEN 1 ELSE 0 END, created_at DESC").
		Find(&skills)

	tools := make([]ToolDef, 0, len(skills))
	for _, sk := range skills {
		baseTool := toolNameFromSkill(sk)
		if baseTool == "" {
			continue
		}
		tools = append(tools, ToolDef{
			Name:        formatSkillToolName(sk.ID),
			Description: fmt.Sprintf("Skill[%s/%s] %s", sk.Name, sk.SkillType, sk.Description),
			Parameters:  toolParamsFromSkill(sk),
		})
	}

	tools = append(tools, e.getMCPToolDefs()...)
	return tools
}

func (e *AgentEngine) getMCPToolDefs() []ToolDef {
	if e == nil || e.mcpProvider == nil {
		return nil
	}
	runtimeTools := e.mcpProvider.ListAvailableTools()
	defs := make([]ToolDef, 0, len(runtimeTools))
	for _, tool := range runtimeTools {
		params := map[string]interface{}{"arguments": "MCP 工具参数对象"}
		if len(tool.InputSchema) > 0 {
			var schema map[string]interface{}
			if json.Unmarshal(tool.InputSchema, &schema) == nil {
				params = schema
			}
		}
		defs = append(defs, ToolDef{
			Name:        formatMCPToolName(tool.ServerID, tool.Name),
			Description: fmt.Sprintf("MCP[%s] %s", tool.ServerName, tool.Description),
			Parameters:  params,
		})
	}
	return defs
}

func formatMCPToolName(serverID, toolName string) string {
	return "mcp__" + serverID + "__" + strings.ReplaceAll(toolName, "__", "_")
}

func parseMCPToolName(name string) (string, string, bool) {
	if !strings.HasPrefix(name, "mcp__") {
		return "", "", false
	}
	parts := strings.SplitN(strings.TrimPrefix(name, "mcp__"), "__", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func formatSkillToolName(skillID string) string {
	return "skill__" + strings.ReplaceAll(skillID, "-", "_")
}

func parseSkillToolName(name string) (string, bool) {
	if !strings.HasPrefix(name, "skill__") {
		return "", false
	}
	id := strings.ReplaceAll(strings.TrimPrefix(name, "skill__"), "_", "-")
	return id, id != ""
}

// getToolsForSkill 返回指定 Skill 对应的工具（精确边界）
func (e *AgentEngine) getToolsForSkill(skillID string) []ToolDef {
	var skill models.Skill
	if err := e.db.Where("id = ? AND enabled = ?", skillID, true).First(&skill).Error; err != nil {
		return nil
	}

	toolName := toolNameFromSkill(skill)
	if toolName == "" {
		return nil
	}

	return []ToolDef{{
		Name:        formatSkillToolName(skill.ID),
		Description: fmt.Sprintf("Skill[%s/%s] %s", skill.Name, skill.SkillType, skill.Description),
		Parameters:  toolParamsFromSkill(skill),
	}}
}

// toolNameFromSkill 从 Skill 类型映射工具名
func toolNameFromSkill(sk models.Skill) string {
	// 从 config 中提取 tool 字段
	if sk.Config != "" {
		var cfg map[string]interface{}
		if json.Unmarshal([]byte(sk.Config), &cfg) == nil {
			if t, ok := cfg["tool"].(string); ok && t != "" {
				return t
			}
		}
	}
	// 默认: skill 类型直接作为工具名
	switch sk.SkillType {
	case "chat":
		return "chat"
	case "text2sql":
		return "text2sql"
	case "env_setup":
		return "setup_env"
	case "script_exec":
		return "execute_script"
	case "pdf", "report":
		if sk.SkillType == "pdf" {
			return "generate_pdf"
		}
		return "generate_report"
	default:
		return sk.SkillType
	}
}

// toolParamsFromSkill 从 Skill 描述推断工具参数
func toolParamsFromSkill(sk models.Skill) map[string]interface{} {
	switch sk.SkillType {
	case "chat":
		return map[string]interface{}{"query": "用户的问题"}
	case "text2sql":
		return map[string]interface{}{"question": "用中文描述要查询什么数据"}
	case "env_setup":
		return map[string]interface{}{"script_name": "脚本名称", "script_content": "脚本内容（用于检测依赖）", "extra_packages": "额外包（逗号分隔，可选）"}
	case "script_exec":
		return map[string]interface{}{"script_path": "脚本文件路径", "script_content": "脚本内容", "language": "bash | python"}
	case "pdf":
		return map[string]interface{}{"title": "报表标题", "content": "报表内容"}
	case "report":
		return map[string]interface{}{"title": "报表名称", "columns": "列名数组", "rows": "数据行数组"}
	default:
		return map[string]interface{}{"input": "操作参数"}
	}
}

// ToolDef 工具定义
type ToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

func (e *AgentEngine) executeSkillTool(ctx context.Context, user *UserContext, skillID string, input map[string]interface{}) (string, error) {
	var skill models.Skill
	if err := e.db.Where("id = ? AND enabled = ?", skillID, true).First(&skill).Error; err != nil {
		return "", fmt.Errorf("Skill 不可用: %s", skillID)
	}
	if user.Context == nil {
		user.Context = map[string]interface{}{}
	}
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
	baseTool := toolNameFromSkill(skill)
	if baseTool == "" {
		return "", fmt.Errorf("Skill 未配置底层工具: %s", skill.Name)
	}
	return e.executeTool(ctx, user, baseTool, input)
}

// executeTool 执行指定工具
func (e *AgentEngine) executeTool(ctx context.Context, user *UserContext, toolName string, input map[string]interface{}) (string, error) {
	if skillID, ok := parseSkillToolName(toolName); ok {
		return e.executeSkillTool(ctx, user, skillID, input)
	}
	if serverID, mcpToolName, ok := parseMCPToolName(toolName); ok {
		if e.mcpProvider == nil {
			return "", fmt.Errorf("MCP 工具提供器未初始化")
		}
		if tpl, ok := input["__template"].(string); ok && tpl != "" {
			if rendered := templating.SafeRender(tpl, map[string]interface{}{"input": input, "user": user}, ""); rendered != "" {
				var renderedInput map[string]interface{}
				if json.Unmarshal([]byte(rendered), &renderedInput) == nil {
					input = renderedInput
				}
			}
		}
		result, err := e.mcpProvider.CallTool(serverID, mcpToolName, input)
		if err != nil {
			return "", err
		}
		return string(result), nil
	}

	switch toolName {
	case "chat":
		query, _ := input["query"].(string)
		if query == "" {
			return "请提供要查询的问题", nil
		}
		// 直接调 LLM
		provider, err := e.providers.GetActiveProvider()
		if err != nil {
			return "", err
		}
		prompt := fmt.Sprintf("用中文简洁回答: %s", query)
		resp, err := e.providers.CallLLM(ctx, provider, prompt)
		if err != nil {
			return "", err
		}
		return resp, nil

	case "text2sql":
		question, _ := input["question"].(string)
		if question == "" {
			return "请提供要查询的问题", nil
		}
		result, err := e.executeText2SQL(question, user)
		if err != nil {
			return "", err
		}
		// 返回结构化结果：SQL + 列名 + 数据 + 统计
		return formatText2SQLResult(result), nil

	case "employee_query":
		// employee_query 功能已合并到 text2sql Skill，返回提示引导用户
		return "employee_query 功能已整合到数据查询 Skill，请直接描述您的问题，如「林烽上月出了多少单」", nil

	case "generate_pdf", "generate_report":
		title, _ := input["title"].(string)
		content, _ := input["content"].(string)
		if content == "" {
			// 尝试从 query/keyword 取内容
			content, _ = input["query"].(string)
		}
		if title == "" {
			title = "数据报表"
		}
		if content == "" {
			content = "无数据内容"
		}
		content = templating.SafeRender(reportPDFContentJinja, map[string]interface{}{"title": title, "content": content}, content)

		// 生成 PDF — 带标题和文本内容
		ctx := context.Background()
		columns := []string{"内容"}
		rows := [][]interface{}{{content}}
		url, err := e.generateReportPDF(ctx, title, columns, rows)
		if err != nil {
			return "", fmt.Errorf("PDF 生成失败: %w", err)
		}
		return fmt.Sprintf("PDF 报表已生成: %s", url), nil

	case "setup_env":
		// 设置 Python 执行环境（uv + 依赖）
		scriptName, _ := input["script_name"].(string)
		scriptContent, _ := input["script_content"].(string)
		extraStr, _ := input["extra_packages"].(string)
		var extraPkgs []string
		if extraStr != "" {
			extraPkgs = strings.Split(extraStr, ",")
		}
		if e.env == nil {
			return "环境管理器未初始化", nil
		}
		envName, err := e.env.SetupScriptEnv(ctx, scriptName, scriptContent, extraPkgs)
		if err != nil {
			return "", fmt.Errorf("环境设置失败: %w", err)
		}
		return fmt.Sprintf("[环境就绪] 环境: %s, uv: %v, Python: %s", envName, e.env.IsUVInstalled(), e.env.GetUVPython(envName)), nil

	case "execute_script":
		// 执行脚本
		scriptContent, _ := input["script_content"].(string)
		scriptPath, _ := input["script_path"].(string)
		language, _ := input["language"].(string)

		if scriptPath == "" && scriptContent == "" {
			return "请提供 script_path 或 script_content", nil
		}

		// 创建临时文件
		if scriptPath == "" && scriptContent != "" {
			ext := ".sh"
			if language == "python" {
				ext = ".py"
			}
			tmpFile, err := os.CreateTemp("", "opentether-script-*"+ext)
			if err != nil {
				return "", fmt.Errorf("创建临时脚本文件失败: %w", err)
			}
			defer os.Remove(tmpFile.Name())
			tmpFile.WriteString(scriptContent)
			tmpFile.Close()
			scriptPath = tmpFile.Name()
		}

		var output string
		var execErr error
		switch language {
		case "python", "py":
			envName := sanitizeEnvName(filepath.Base(scriptPath))
			output, execErr = e.env.RunScript(ctx, envName, scriptPath)
		default: // bash
			cmd := exec.CommandContext(ctx, "bash", scriptPath)
			out, err := cmd.CombinedOutput()
			output = string(out)
			execErr = err
		}

		if execErr != nil {
			return fmt.Sprintf("[脚本执行] 输出:\n%s\n错误: %v", output, execErr), nil
		}
		return fmt.Sprintf("[脚本执行成功] 输出:\n%s", output), nil

	default:
		return fmt.Sprintf("[工具不可用] %s 功能暂未开放，请尝试其他方式描述您的问题", toolName), nil
	}
}

// availableToolNames 提取工具名列表（用于日志和提示）
func availableToolNames(tools []ToolDef) []string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	return names
}

// buildSystemPrompt 构建系统提示词（注入可用工具描述 + 边界约束 + Soul + 记忆）
func (e *AgentEngine) buildSystemPrompt(tools []ToolDef, user *UserContext, query string) string {
	basePrompt := renderReactSystemPrompt(tools, user, e.config.Executor.EmbeddedConfig.MaxLoopIterations)

	// 注入 Soul + 长期记忆（Letta 风格）
	if e.memory != nil && e.memory.longTerm != nil {
		groupIDs := user.getGroupIDs()
		return e.memory.longTerm.BuildSoulPrompt(user.UserID, groupIDs, query, basePrompt)
	}

	return basePrompt
}

// buildLoopMessages 构建本轮 LLM 消息
func (e *AgentEngine) buildLoopMessages(systemPrompt, query string, loop *LoopContext) []llm.Message {
	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
	}

	conversationContext := BuildConversationMemoryPrompt(loop.Memory, query)

	// 第一次迭代：添加用户原始问题 + 当前对话短期记忆/任务工作记忆
	if loop.Iteration == 0 {
		content := templating.SafeRender(loopFirstMessageJinja, map[string]interface{}{
			"conversation_context": conversationContext,
			"query":                query,
			"has_pronoun":          hasPronounReference(query),
		}, fmt.Sprintf("用户当前问题: %s\n\n请分析需求并决定下一步行动。", query))
		messages = append(messages, llm.Message{Role: "user", Content: content})
		return messages
	}

	var steps []map[string]interface{}
	for _, step := range loop.History {
		if step.Action == "tool_call" {
			output := step.ToolOutput
			if len(output) > 500 {
				output = output[:500] + "...(已截断)"
			}
			steps = append(steps, map[string]interface{}{"no": step.StepID + 1, "tool_name": step.ToolName, "output": output})
		}
	}
	content := templating.SafeRender(loopNextMessageJinja, map[string]interface{}{
		"conversation_context": conversationContext,
		"query":                query,
		"steps":                steps,
	}, fmt.Sprintf("用户当前问题: %s\n\n根据历史步骤，请决定下一步。", query))
	messages = append(messages, llm.Message{Role: "user", Content: content})

	return messages
}

// fallbackResponse 生成降级回复
func (e *AgentEngine) fallbackResponse(conversationID, query string, err error) (*ChatResponse, error) {
	log.Printf("[Agent] LLM 不可用，降级处理: %v", err)
	return &ChatResponse{
		Message:        fmt.Sprintf("处理您的请求时遇到问题，请稍后重试。您的问题: %s", query),
		ConversationID: conversationID,
		SkillUsed:      "fallback",
	}, nil
}

// formatConfirmMessage 格式化确认文档为可读消息
func formatConfirmMessage(doc *ConfirmDocument) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## ⚠️ %s\n\n", doc.Title))
	sb.WriteString(fmt.Sprintf("%s\n\n", doc.Description))
	sb.WriteString(fmt.Sprintf("**风险等级**: %s\n\n", doc.Risk))
	sb.WriteString("### 待执行操作:\n")
	sb.WriteString("| 类型 | 目标 | 详情 |\n")
	sb.WriteString("|------|------|------|\n")
	for _, op := range doc.Operations {
		details := op.Details
		if len(details) > 100 {
			details = details[:100] + "..."
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", op.Type, op.Target, details))
	}
	sb.WriteString("\n---\n")
	sb.WriteString("以上操作需要您的确认。请回复 **确认执行** 以继续，或回复具体修改意见。\n")
	return sb.String()
}

// detectWriteOperation 检测 text2sql SQL 是否包含写操作
func detectWriteOperation(sql string) (isWrite bool, opType string) {
	sqlUpper := strings.TrimSpace(sql)
	if len(sqlUpper) < 6 {
		return false, ""
	}
	prefix := sqlUpper[:6]
	switch {
	case prefix == "INSERT":
		return true, "insert"
	case prefix == "UPDATE":
		return true, "update"
	case len(sqlUpper) >= 6 && sqlUpper[:6] == "DELETE":
		return true, "delete"
	case len(sqlUpper) >= 4 && sqlUpper[:4] == "DROP":
		return true, "delete"
	case len(sqlUpper) >= 5 && sqlUpper[:5] == "ALTER":
		return true, "update"
	case len(sqlUpper) >= 6 && sqlUpper[:6] == "CREATE":
		return true, "update"
	}
	return false, ""
}

// formatText2SQLResult 格式化 text2sql 结果（含列名和数据，供 LLM 分析）
func formatText2SQLResult(result *ChatResponse) string {
	var sb strings.Builder

	if result.Data == nil {
		return result.Message
	}

	// SQL 语句
	if sql, ok := result.Data["sql"].(string); ok && sql != "" {
		sb.WriteString(fmt.Sprintf("[SQL] %s\n", sql))
	}

	// 列名
	if columns, ok := result.Data["columns"].([]string); ok && len(columns) > 0 {
		sb.WriteString(fmt.Sprintf("[列] %s\n", strings.Join(columns, " | ")))
	}

	// 数据行（最多展示 25 行）
	if rows, ok := result.Data["rows"].([][]interface{}); ok && len(rows) > 0 {
		sb.WriteString(fmt.Sprintf("[数据] %d 行:\n", len(rows)))
		maxShow := 25
		if len(rows) < maxShow {
			maxShow = len(rows)
		}
		for i := 0; i < maxShow; i++ {
			vals := make([]string, len(rows[i]))
			for j, v := range rows[i] {
				vals[j] = fmt.Sprintf("%v", v)
			}
			sb.WriteString(fmt.Sprintf("  %s\n", strings.Join(vals, " | ")))
		}
		if len(rows) > maxShow {
			sb.WriteString(fmt.Sprintf("  ...(还有 %d 行未展示)\n", len(rows)-maxShow))
		}
	}

	// 统计
	if rc, ok := result.Data["row_count"].(int); ok {
		sb.WriteString(fmt.Sprintf("[统计] 共 %d 条记录\n", rc))
	}

	if sb.Len() == 0 {
		return result.Message
	}
	return sb.String()
}

// formatEmployeeResultForLoop 格式化员工查询结果
func formatEmployeeResultForLoop(result *ChatResponse) string {
	if result.Data == nil {
		return result.Message
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[员工查询] %s\n", result.Message))

	if columns, ok := result.Data["columns"].([]string); ok && len(columns) > 0 {
		sb.WriteString(fmt.Sprintf("[列] %s\n", strings.Join(columns, " | ")))
	}
	if rows, ok := result.Data["rows"].([][]interface{}); ok && len(rows) > 0 {
		sb.WriteString(fmt.Sprintf("[数据] %d 条记录\n", len(rows)))
		maxShow := 10
		if len(rows) < maxShow {
			maxShow = len(rows)
		}
		for i := 0; i < maxShow; i++ {
			vals := make([]string, len(rows[i]))
			for j, v := range rows[i] {
				vals[j] = fmt.Sprintf("%v", v)
			}
			sb.WriteString(fmt.Sprintf("  %s\n", strings.Join(vals, " | ")))
		}
	}

	return sb.String()
}
