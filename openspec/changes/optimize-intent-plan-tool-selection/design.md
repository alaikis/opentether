## Context

当前智能体三阶段流水线（意图识别 → 任务规划 → 工具选用）存在以下问题：
1. `recognizeIntent` 仅依赖关键词子串匹配，无语义理解
2. `perceiveAndPlan` LLM 调用失败时静默返回 nil，规划完全失效
3. `planExecution` 中 skillMap 硬编码，新增 skill 需改源码
4. `filterToolsByPlan` 使用 `strings.Contains(tool.Name, "skill__")` 宽松匹配，plan 过滤形同虚设
5. `selectRelevantTools` 截断描述到 220 字符，丢失关键参数信息
6. 工具选型结果无反馈机制，无法持续优化

## Goals / Non-Goals

**Goals:**
- `perceiveAndPlan` 增加规则式 fallback，LLM 失败时仍能产出可执行规划
- `filterToolsByPlan` 严格匹配 plan 声明的工具，不存在的工具不保留
- 意图识别注入用户长期记忆（历史 skill 使用偏好）
- 工具选用结果记录反馈到 `SkillRuntimeMemory`
- 新增 embedding 语义匹配模块作为 `selectRelevantTools` 的增强选项
- skillMap 从硬编码迁移到数据库配置表 `skill_intent_rules`

**Non-Goals:**
- 不替换现有 LLM-based `perceiveAndPlan`，仅在其失败时 fallback
- 不改变现有路由优先级（Fast Path > Query Clarification > Multi-task > Agentic Loop）
- 不引入新的外部 embedding 服务，复用现有 `internal/embedding` 模块

## Decisions

**1. perceiveAndPlan 规则 fallback**
在 `perceiveAndPlan` 的 LLM 调用失败或解析失败时，调用新方法 `ruleBasedPlan(query, tools)`，基于关键词匹配和可用工具列表生成最小可行规划。

*Rationale:* 保证规划阶段永远有输出，避免裸 Loop。

**2. filterToolsByPlan 严格匹配**
将 `strings.Contains(tool.Name, "skill__")` 改为精确匹配 plan 中的 tool name 列表，只有显式声明的工具才保留。

*Rationale:* plan 工具过滤应真正约束可用工具范围。

**3. 用户记忆注入意图识别**
在 `recognizeIntent` 中，加载用户最近使用的 skill 列表，作为额外上下文参与打分。

*Rationale:* 用户历史偏好是强信号，可提升个性化准确率。

**4. 工具选用反馈记录**
在 `selectRelevantTools` 返回结果后，记录 (query, selected_tools, score) 到 `SkillRuntimeMemory`，供后续分析。

*Rationale:* 积累匹配数据，为后续优化提供依据。

**5. embedding 语义匹配**
新增 `semantic_match.go`，提供 `SemanticMatchTools(query, tools, limit)` 接口，基于 embedding 向量相似度排序工具。

*Rationale:* 解决关键词匹配的同义词、语序、表述差异问题。

**6. skillMap 数据库化**
新增 `skill_intent_rules` 表，支持 `(intent, skill_name, skill_type, priority)` 配置，`planExecution` 从数据库查询而非硬编码 map。

*Rationale:* 新增 skill 类型无需改源码，支持动态扩展。

## Risks / Trade-offs

- **规则 fallback 质量低于 LLM** → 仅作为兜底，不影响正常 LLM 路径
- **严格匹配导致 plan 工具过滤过严** → plan 生成时 LLM 可声明 `__all__` 表示不限制
- **embedding 匹配增加延迟** → 对工具列表（通常 <20）进行 embedding 计算，延迟可控
- **数据库 skillMap 查询增加一次 DB 访问** → 结果可缓存，TTL 5 分钟

## Migration Plan

1. 修改 `perceiveAndPlan` 增加规则 fallback
2. 修改 `filterToolsByPlan` 严格匹配
3. 修改 `recognizeIntent` 注入用户记忆
4. 修改 `selectRelevantTools` 增加反馈记录
5. 新增 `semantic_match.go`
6. 新增 `skill_intent_rules` 表和迁移
7. 修改 `planExecution` 查询数据库 skillMap
8. 全量测试验证

## Open Questions

- embedding 匹配是否作为默认启用还是可选开关？
- skill_intent_rules 的优先级如何与硬编码 fallback 交互？
