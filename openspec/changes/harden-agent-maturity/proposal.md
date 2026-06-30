## Why

当前智能体在标准单轮查询场景下已可稳定运行，但在复杂多步推理和多任务编排方面存在明显短板：多任务编排能力已编码但未接入主流程，导致用户的多部分问题无法触发并行执行；自学习和记忆机制有框架但缺乏实际验证数据；复杂推理质量依赖 LLM 决策，存在不确定性。需要在现有 AgentEngine 基础上补齐这些能力，提升生产级稳健性。

## What Changes

- 将 `ExecuteMultiTaskPlan` 接入主对话流程，支持用户多部分问题自动拆分并行执行
- 修复 `detectTaskTree`、`extractContextWords`、`extractMetricFromQuery` 桩代码，实现树状任务检测与嵌套子任务派发
- 增强 `isCompleteToolResult` 的鲁棒性，支持多种工具输出格式的完成检测
- 为自学习系统增加验证数据注入和效果回测机制
- 增加 LLM 决策质量监控与降级策略

## Capabilities

### New Capabilities
- `multi-task-orchestration`: 多任务编排执行能力，支持用户多部分问题自动拆分、并行执行、结果汇总
- `agent-validation`: 智能体验证与监控，包括自学习效果回测、LLM 决策质量监控、降级策略

### Modified Capabilities
- `agent-core`: 增强 ReAct Loop 的鲁棒性，支持 early-stop 条件多样化、工具输出格式自适应

## Impact

- 影响 `internal/agent/multi_task.go`、`internal/agent/loop.go`、`internal/agent/engine.go`、`internal/agent/self_learning.go`
- 影响 `internal/service/services.go` 主对话入口
- 新增验证数据存储模型和监控接口
