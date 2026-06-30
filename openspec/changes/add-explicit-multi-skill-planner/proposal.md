## Why

当前多任务处理主要通过 `multi_task.go` 中的 `BuildMultiTaskPlan` 实现，但该实现仅做文本拆分和简单的关键词匹配，缺乏显式的技能分配能力。用户提交多部分查询时，系统无法明确告知每个子任务将使用哪个 Skill 执行，导致可解释性不足。同时，多任务执行的 plan-execute-summarize 流程缺少专门的 API 入口和前端集成点。

## What Changes

1. **显式多技能规划器** - 新增 `SkillPlanner` 组件，能够分析多部分查询并为每个子任务显式分配最匹配的 Skill
2. **多任务 API 暴露** - 将现有的 `BuildMultiTaskPlan`、`ExecuteMultiTaskPlan`、`buildMultiTaskSummary` 通过专用 API 端点暴露
3. **增强的摘要生成** - 优化多任务结果摘要，包含每个子任务使用的 Skill 和执行状态

## Capabilities

### New Capabilities

- `explicit-multi-skill-planner`: 显式多技能规划器，为多部分查询分配具体 Skill
- `multi-task-plan-execute-summarize`: 多任务规划、执行、摘要 API 工作流

### Modified Capabilities

- 无

## Impact

- 新增 `internal/agent/skill_planner.go`
- 新增 `internal/handler/multi_task_handler.go`
- 修改 `internal/router/api.go` 注册新路由
- 无需修改现有核心逻辑，向后兼容
