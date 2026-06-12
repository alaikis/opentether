package agent

import (
	"encoding/json"
	"fmt"

	"github.com/alaikis/opentether/internal/templating"
)

const reactSystemPromptJinja = `你是 OpenTether AI 助手，一个企业级 AI Agent。你必须在给定的权限和工具范围内工作。

## ⚠️ 边界约束（不可违反）
- 只能使用下面列出的工具，不得自行创造或假设其他能力
- 不得修改用户权限、数据范围或系统配置
- 不得代替管理员做出权限决策
- 当前用户: {{ user.name }} ({{ user.department }}), 部门: {{ user.department }}
- 用户状态: {{ user.status }} | 可用工具数: {{ tools_count }}

## 可用工具（仅限以下）
{% for tool in tools %}
- **{{ tool.name }}**: {{ tool.description }} (参数: {{ tool.params_json }})
{% endfor %}

## 响应格式
你必须严格按以下 JSON 格式响应（不要包含其他文字）：

如果需要使用单个工具：
~~~json
{"action":"tool_call","thought":"你的推理过程","tool_name":"工具名","tool_input":{"参数":"值"}}
~~~

如果需要同时使用多个互不依赖的工具（并行调用）：
~~~json
{"action":"parallel_calls","thought":"你的推理过程","calls":[{"tool_name":"工具名","tool_input":{"参数":"值"}},{"tool_name":"工具名2","tool_input":{"参数":"值"}}]}
~~~

如果已经有足够信息回答用户：
~~~json
{"action":"final_answer","thought":"你的推理过程","final_answer":"最终答案"}
~~~

## 规则
- 优先使用 parallel_calls 同时调用多个互不依赖的工具
- 多个独立查询（如同时查两张不同的表）必须用并行调用
- 写操作必须先 confirm
- 最多 {{ max_iterations }} 步
- 用中文回答，保持专业、简洁
- 上下文有严格作用域：当前任务 > 当前对话 > 当前用户 > 当前部门/组 > 公司。不得把其它任务、其它对话、其它用户的实体混入当前任务。
- 如果用户使用“他/她/它/这个人/该员工/刚才那个/上面的人”等指代，必须优先从当前任务状态和最近对话中解析。只有存在多个候选实体时才反问。
- 调用查询类工具时，tool_input.question 必须是结合上下文补全后的完整问题，例如把“他上季度多少单”补全为“林烽上季度订单数是多少”。
- 如果话题路由动作为 clarify，必须先向用户列出候选话题并澄清，不要擅自调用工具。
`

const loopFirstMessageJinja = `{% if conversation_context %}{{ conversation_context }}{% endif %}用户当前问题: {{ query }}

{% if has_pronoun %}注意：当前问题包含指代词。请先用上面的当前任务/最近对话解析指代，并把补全后的完整问题传给工具。

{% endif %}请分析需求并决定下一步行动。`

const loopNextMessageJinja = `{% if conversation_context %}{{ conversation_context }}{% endif %}用户当前问题: {{ query }}

## 已执行的步骤:
{% for step in steps %}- 第{{ step.no }}步: 调用了 {{ step.tool_name }}, 结果: {{ step.output }}
{% endfor %}

根据以上结果，请决定下一步: 继续调用工具还是给出最终答案？`

const reportPDFContentJinja = `# {{ title }}

{{ content }}
`

const runtimeCheckpointSummaryJinja = `第 {{ step }} 步 [{{ type }}] {% if tool_name %}工具 {{ tool_name }}{% endif %} {{ summary }}`

const conversationSummaryCompressJinja = `请把以下企业智能体对话记忆压缩为一段结构化短期摘要，限制 500 字以内。
要求：
1. 保留当前任务、活跃实体、关键指标、时间范围和用户偏好。
2. 删除寒暄、重复过程、SQL 细节和无关任务噪声。
3. 如果出现多个任务，用简短条目区分，不要混淆实体。
4. 输出纯文本，不要 Markdown 标题。

待压缩摘要：
{{ summary }}`

func renderReactSystemPrompt(tools []ToolDef, user *UserContext, maxIterations int) string {
	toolData := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		params, _ := json.Marshal(tool.Parameters)
		toolData = append(toolData, map[string]interface{}{"name": tool.Name, "description": tool.Description, "params_json": string(params)})
	}
	data := map[string]interface{}{
		"tools":          toolData,
		"tools_count":    len(tools),
		"max_iterations": maxIterations,
		"user": map[string]interface{}{
			"name":       user.Name,
			"department": user.Department,
			"status":     user.Status,
		},
	}
	fallback := fmt.Sprintf("你是 OpenTether AI 助手。可用工具数: %d。必须严格返回 JSON 决策。", len(tools))
	return templating.SafeRender(reactSystemPromptJinja, data, fallback)
}
