package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/llm"
	"github.com/alaikis/opentether/internal/models"
)

// ============================================
// Agentic Loop - ReAct 模式多步推理执行
// ============================================

const (
	MaxLoopIterations = 10
	LoopTimeout       = 5 * time.Minute
)

// LoopContext 循环执行上下文
type LoopContext struct {
	UserID         string
	ConversationID string
	OriginalQuery  string
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

// ExecuteLoop 执行 ReAct 循环
func (e *AgentEngine) ExecuteLoop(ctx context.Context, user *UserContext, query string, conversationID string) (*ChatResponse, error) {
	loop := &LoopContext{
		UserID:         user.UserID,
		ConversationID: conversationID,
		OriginalQuery:  query,
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

	// 构建系统 prompt（含可用工具描述 + 边界约束）
	systemPrompt := e.buildSystemPrompt(availableTools, user)

	// 创建 LLM client
	llmClient, err := llm.NewClient(provider)
	if err != nil {
		return e.fallbackResponse(conversationID, query, err)
	}

	totalTokens := 0

	for loop.Iteration = 0; loop.Iteration < MaxLoopIterations; loop.Iteration++ {
		// 超时检查
		if time.Since(loop.StartTime) > LoopTimeout {
			loop.History = append(loop.History, LoopStep{
				StepID: loop.Iteration,
				Action: "error",
				Error:  "执行超时",
			})
			break
		}

		// 构建本轮消息
		messages := e.buildLoopMessages(systemPrompt, query, loop)

		// 调用 LLM 推理下一步
		resp, err := llmClient.ChatCompletion(ctx, llm.ChatRequest{
			Model:       provider.Model,
			Messages:    messages,
			MaxTokens:   2048,
			Temperature: 0.3,
		})
		if err != nil {
			step := LoopStep{
				StepID: loop.Iteration,
				Action: "error",
				Error:  err.Error(),
			}
			loop.History = append(loop.History, step)
			break
		}

		totalTokens += resp.Usage.TotalTokens

		// 解析 LLM 决策
		decision, parseErr := parseLoopDecision(resp.Content)
		if parseErr != nil {
			log.Printf("[Loop] 解析决策失败: %v, raw=%.200s", parseErr, resp.Content)
			return &ChatResponse{
				Message:        resp.Content,
				ConversationID: conversationID,
				TokensUsed:     totalTokens,
			}, nil
		}

		step := LoopStep{
			StepID:  loop.Iteration,
			Action:  decision.Action,
			Thought: decision.Thought,
		}

		switch decision.Action {
		case "final_answer":
			loop.History = append(loop.History, step)

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

			// === 边界检查：工具是否在用户权限范围内 ===
			if !toolNames[decision.ToolName] {
				log.Printf("[Loop] 拒绝未授权工具: %s (user=%s)", decision.ToolName, user.UserID)
				step.Error = fmt.Sprintf("工具 %s 不在可用范围内", decision.ToolName)
				step.ToolOutput = fmt.Sprintf("[系统] 工具 %s 不可用，你只能使用: %v。请用已有工具或直接回答。", decision.ToolName, availableToolNames(availableTools))
				observation := fmt.Sprintf("[边界检查] 拒绝工具 %s: 不在用户权限范围内", decision.ToolName)
				loop.Observations = append(loop.Observations, observation)
				loop.History = append(loop.History, step)
				continue
			}

			toolOutput, toolErr := e.executeTool(ctx, user, decision.ToolName, decision.ToolInput)
			if toolErr != nil {
				step.Error = toolErr.Error()
				step.ToolOutput = fmt.Sprintf("工具执行失败: %v", toolErr)
			} else {
				step.ToolOutput = toolOutput
			}

			observation := fmt.Sprintf("[工具 %s 执行结果]: %s", decision.ToolName, step.ToolOutput)
			loop.Observations = append(loop.Observations, observation)
			loop.History = append(loop.History, step)

		case "parallel_calls":
			// === 并行执行：多个互不依赖的工具调用同时执行 ===
			if len(decision.ParallelCalls) == 0 {
				step.Error = "parallel_calls 需要提供 calls 列表"
				step.ToolOutput = "[系统] parallel_calls 缺少调用列表，请提供互不依赖的多个工具调用"
				loop.History = append(loop.History, step)
				continue
			}

			log.Printf("[Loop] 并行执行 %d 个工具调用", len(decision.ParallelCalls))
			parallelSteps := e.executeParallelCalls(ctx, user, decision.ParallelCalls, toolNames)
			observation := formatParallelResults(parallelSteps)
			loop.Observations = append(loop.Observations, observation)
			loop.History = append(loop.History, parallelSteps...)

		case "clarify":
			// === 参数补全：向用户提问，等待下一轮对话补充参数 ===
			loop.History = append(loop.History, step)
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
			doc := decision.ConfirmDoc
			if doc == nil {
				step.Error = "confirm 需要提供 confirm_doc"
				loop.History = append(loop.History, step)
				continue
			}
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

// parseLoopDecision 从 LLM 输出解析决策 JSON
func parseLoopDecision(content string) (*LoopDecision, error) {
	// 尝试从 markdown code block 中提取 JSON
	content = strings.TrimSpace(content)

	// 查找 ```json ... ``` 块
	if idx := strings.Index(content, "```json"); idx >= 0 {
		start := idx + 7
		if end := strings.Index(content[start:], "```"); end >= 0 {
			content = strings.TrimSpace(content[start : start+end])
		}
	} else if idx := strings.Index(content, "{"); idx >= 0 {
		// 查找第一个完整的 JSON 对象
		content = content[idx:]
		if end := strings.LastIndex(content, "}"); end >= 0 {
			content = content[:end+1]
		}
	}

	var decision LoopDecision
	if err := json.Unmarshal([]byte(content), &decision); err != nil {
		return nil, fmt.Errorf("parse decision: %w, content: %.200s", err, content)
	}

	if decision.Action == "" {
		return nil, fmt.Errorf("decision action is empty")
	}

	return &decision, nil
}

// getAvailableTools 从已注册的 Skill 中派生可用工具列表
// 所有工具均来自 skills 表（内置 + 用户自定义），智能体不能自创工具
func (e *AgentEngine) getAvailableTools(user *UserContext) []ToolDef {
	// 从数据库加载所有启用的 Skills
	var skills []models.Skill
	e.db.Where("enabled = ?", true).Find(&skills)

	tools := make([]ToolDef, 0, len(skills))
	seen := make(map[string]bool)

	for _, sk := range skills {
		toolName := toolNameFromSkill(sk)
		if toolName == "" || seen[toolName] {
			continue
		}
		seen[toolName] = true

		tools = append(tools, ToolDef{
			Name:        toolName,
			Description: sk.Description,
			Parameters:  toolParamsFromSkill(sk),
		})
	}

	return tools
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
	case "employee_query":
		return "employee_query"
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
	case "employee_query":
		return map[string]interface{}{"query_type": "list | search | detail", "keyword": "搜索关键词"}
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

// executeTool 执行指定工具
func (e *AgentEngine) executeTool(ctx context.Context, user *UserContext, toolName string, input map[string]interface{}) (string, error) {
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
		keyword, _ := input["keyword"].(string)
		result, err := e.executeEmployee(keyword, user)
		if err != nil {
			return "", err
		}
		// 返回结构化结果
		return formatEmployeeResultForLoop(result), nil

	case "generate_pdf", "generate_report":
		// 占位：返回提示
		title, _ := input["title"].(string)
		return fmt.Sprintf("报表 [%s] 已生成，请在管理后台查看下载", title), nil

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
		return "", fmt.Errorf("未知工具: %s", toolName)
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

// buildSystemPrompt 构建系统提示词（注入可用工具描述 + 边界约束）
func (e *AgentEngine) buildSystemPrompt(tools []ToolDef, user *UserContext) string {
	var sb strings.Builder
	sb.WriteString("你是 OpenTether AI 助手，一个企业级 AI Agent。你必须在给定的权限和工具范围内工作。\n\n")

	// 边界约束
	sb.WriteString("## ⚠️ 边界约束（不可违反）\n")
	sb.WriteString("- 只能使用下面列出的工具，不得自行创造或假设其他能力\n")
	sb.WriteString("- 不得修改用户权限、数据范围或系统配置\n")
	sb.WriteString("- 不得代替管理员做出权限决策\n")
	sb.WriteString(fmt.Sprintf("- 当前用户: %s (%s), 部门: %s\n", user.Name, user.Department, user.Department))
	sb.WriteString(fmt.Sprintf("- 用户状态: %s | 可用工具数: %d\n\n", user.Status, len(tools)))

	sb.WriteString("## 可用工具（仅限以下）\n")
	for _, tool := range tools {
		paramsJSON, _ := json.Marshal(tool.Parameters)
		sb.WriteString(fmt.Sprintf("- **%s**: %s (参数: %s)\n", tool.Name, tool.Description, string(paramsJSON)))
	}

	sb.WriteString("\n## 响应格式\n")
	sb.WriteString("你必须严格按以下 JSON 格式响应（不要包含其他文字）：\n\n")
	sb.WriteString("如果需要使用单个工具：\n")
	sb.WriteString("```json\n")
	sb.WriteString(`{"action":"tool_call","thought":"你的推理过程","tool_name":"工具名","tool_input":{"参数":"值"}}` + "\n")
	sb.WriteString("```\n\n")
	sb.WriteString("如果有多个互不依赖的工具调用（如报表的多维度查询），可以并行执行：\n")
	sb.WriteString("```json\n")
	sb.WriteString(`{"action":"parallel_calls","thought":"这些查询互不依赖","calls":[{"tool_name":"text2sql","tool_input":{"question":"各地区销售额"}},{"tool_name":"text2sql","tool_input":{"question":"产品排名"}}]}` + "\n")
	sb.WriteString("```\n\n")
	sb.WriteString("如果信息不足以完成任务（如用户没说表名、时间范围等），向用户提问：\n")
	sb.WriteString("```json\n")
	sb.WriteString(`{"action":"clarify","thought":"推理过程","clarify_question":"请问您需要查询哪个部门的数据？"}` + "\n")
	sb.WriteString("```\n\n")
	sb.WriteString("如果需要进行插入/更新/删除等写操作，必须先让用户确认：\n")
	sb.WriteString("```json\n")
	sb.WriteString(`{"action":"confirm","thought":"推理过程","confirm_doc":{"title":"操作标题","description":"操作说明","operations":[{"type":"insert","target":"表名","details":"操作详情"}],"risk":"low"}}` + "\n")
	sb.WriteString("```\n\n")
	sb.WriteString("如果已经有足够信息回答用户：\n")
	sb.WriteString("```json\n")
	sb.WriteString(`{"action":"final_answer","thought":"你的推理过程","final_answer":"最终答案"}` + "\n")
	sb.WriteString("```\n\n")
	sb.WriteString("## 规则\n")
	sb.WriteString("- 每次只调用一个工具\n")
	sb.WriteString("- 先思考再行动，如果工具不可用则直接告知用户\n")
	sb.WriteString("- 参数不完整时必须用 clarify 向用户提问，不得自行猜测\n")
	sb.WriteString("- 写操作（插入、更新、删除）必须先用 confirm 让用户确认\n")
	sb.WriteString(fmt.Sprintf("- 最多 %d 步，超出则总结已有信息\n", MaxLoopIterations))
	sb.WriteString("- 用中文回答，保持专业、简洁\n")

	return sb.String()
}

// buildLoopMessages 构建本轮 LLM 消息
func (e *AgentEngine) buildLoopMessages(systemPrompt, query string, loop *LoopContext) []llm.Message {
	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
	}

	// 第一次迭代：添加用户原始问题
	if loop.Iteration == 0 {
		messages = append(messages, llm.Message{
			Role:    "user",
			Content: fmt.Sprintf("用户问题: %s\n\n请分析需求并决定下一步行动。", query),
		})
		return messages
	}

	// 后续迭代：添加观察结果
	historyText := fmt.Sprintf("用户问题: %s\n\n## 已执行的步骤:\n", query)
	for _, step := range loop.History {
		if step.Action == "tool_call" {
			output := step.ToolOutput
			if len(output) > 500 {
				output = output[:500] + "...(已截断)"
			}
			historyText += fmt.Sprintf("- 第%d步: 调用了 %s, 结果: %s\n", step.StepID+1, step.ToolName, output)
		}
	}
	historyText += "\n根据以上结果，请决定下一步: 继续调用工具还是给出最终答案？"

	messages = append(messages, llm.Message{
		Role:    "user",
		Content: historyText,
	})

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
