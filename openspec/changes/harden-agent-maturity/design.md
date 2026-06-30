## Context

当前智能体核心执行路径位于 `internal/agent/loop.go` 的 `ExecuteLoopWithEvents`，支持单轮 ReAct 推理、Fast Path 路由、并行工具调用。多任务编排代码存在于 `internal/agent/multi_task.go`，但 `ExecuteMultiTaskPlan` 未被任何入口调用。自学习框架在 `self_learning.go` 中实现了失败模式记录和反馈记忆，但缺乏验证数据回测机制。`isCompleteToolResult` 硬编码关键词检测，鲁棒性不足。

## Goals / Non-Goals

**Goals:**
- 将多任务编排接入主对话流程，用户多部分问题自动触发并行执行
- 修复 `detectTaskTree`、`extractContextWords`、`extractMetricFromQuery` 桩代码
- 增强工具输出完成检测的鲁棒性，支持多种格式
- 为自学习系统增加验证数据回测接口
- 增加 LLM 决策质量监控与降级策略

**Non-Goals:**
- 不改动现有 LLM provider 接入方式
- 不引入新的外部依赖
- 不改变数据库 schema（验证数据复用现有表结构）

## Decisions

**1. 多任务编排接入点**
在 `services.go` 的 `ChatStream` 和 `Chat` 入口中，在进入 Agentic Loop 之前，先检测输入是否包含多个独立子问题（通过 `BuildMultiTaskPlan`），若命中则调用 `ExecuteMultiTaskPlan`，将汇总结果作为最终回复返回。

*Rationale:* 保留现有 Fast Path 和 Query Clarification 优先级，多任务作为独立分支插入，最小侵入现有逻辑。

**2. 树状任务检测**
基于分隔符（`、`，`；`，换行）和查询相似度阈值（>0.8）判断是否为多部分问题。`detectTaskTree` 复用 `BuildMultiTaskPlan` 的扁平拆分结果，通过 `isTree` 标记区分。

*Rationale:* 当前 `detectTaskTree` 恒返回 nil，直接复用已有拆分逻辑，避免过度设计。

**3. 嵌套子任务生成**
`extractContextWords` 和 `extractMetricFromQuery` 基于父任务查询提取上下文词和指标词，用于生成子任务查询模板。

*Rationale:* 父任务结果中的列表项（如部门、产品线）自动展开为子查询，保持上下文连贯性。

**4. 工具输出完成检测**
将 `isCompleteToolResult` 改为基于 JSON 结构检测（检查是否包含 `columns`、`rows`、`row_count` 等标准字段），并增加对非结构化输出的启发式检测（长度、关键词组合）。

*Rationale:* 硬编码 `[SQL][列][数据][统计]` 过于脆弱，改为基于结构化字段和语义组合的检测。

**5. 自学习验证回测**
新增 `SelfLearning.Backtest(query string) *BacktestResult` 方法，对历史失败模式进行回测，记录改善率。回测数据不持久化，仅用于运行时监控。

*Rationale:* 提供自学习效果的可观测性，但不增加持久化复杂度。

**6. LLM 决策质量监控**
在 `retryLoopDecision` 中记录每次 LLM 响应的解析耗时和重试次数，超过阈值时触发降级策略（直接 fallback 回答）。

*Rationale:* 复杂推理质量依赖 LLM，需要监控和降级机制保障可用性。

## Risks / Trade-offs

- **多任务并行增加 token 消耗** → 限制最大并行度和子任务数量（最多 5 个）
- **树状任务递归深度无限** → 设置最大递归深度（3 层）
- **LLM 决策质量波动** → 增加超时和重试上限，超限后直接 fallback
- **自学习回测耗时** → 仅在后台异步执行，不阻塞主流程

## Migration Plan

1. 修改 `services.go` 入口，增加多任务检测分支
2. 实现 `multi_task.go` 桩代码
3. 增强 `isCompleteToolResult` 鲁棒性
4. 增加自学习回测和监控接口
5. 运行现有测试，确保无回归
6. 手动验证多任务场景

## Open Questions

- 多任务最大并行数量是否需要在配置中暴露？
- 自学习回测结果是否需要持久化到数据库？
