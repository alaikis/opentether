## Why

当前意图识别、任务规划和工具选用三块存在明确的脆弱点和优化空间：意图识别依赖硬编码关键词匹配和置信度阈值，任务规划的 LLM 决策无规则式 fallback，工具选用存在截断损失和过滤失效问题。这些短板导致复杂 query 的准确率和稳健性不足，需系统化补齐。

## What Changes

- `perceiveAndPlan` 增加规则式 fallback：LLM 失败时按关键词映射到 skill，确保规划不失效
- `filterToolsByPlan` 改为严格匹配：plan 中声明的工具必须真实存在才保留，过滤逻辑生效
- 意图识别注入用户长期记忆：将历史偏好和常用 skill 作为上下文输入，提升个性化准确率
- 工具选用增加反馈记录：将 tool→query 的匹配效果写入 `SkillRuntimeMemory`，用于后续优化
- 新增 embedding 语义匹配模块：对 tool description 和 query 做 embedding 相似度匹配，解决关键词盲区
- skillMap 数据库化：将 intent→skill 映射从硬编码改为数据库配置表 `skill_intent_rules`

## Capabilities

### New Capabilities
- `semantic-tool-matching`: 基于 embedding 的工具语义匹配，解决纯关键词匹配的同义词和语序问题
- `dynamic-intent-routing`: 动态意图路由规则，支持数据库配置 intent→skill 映射

### Modified Capabilities
- `agent-core`: 增强 `perceiveAndPlan` fallback、`filterToolsByPlan` 严格匹配、意图识别用户记忆注入、工具选用反馈记录

## Impact

- 影响 `internal/agent/engine.go`、`internal/agent/loop.go`、`internal/agent/fast_path.go`
- 新增 `internal/agent/semantic_match.go`
- 新增数据库表 `skill_intent_rules`
- 影响 `internal/service/services.go` 和 `internal/router/api.go`
