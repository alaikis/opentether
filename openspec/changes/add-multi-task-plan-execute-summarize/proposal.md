## Why

当前多任务处理逻辑 (`internal/agent/multi_task.go`) 已实现核心的 plan-execute-summarize 流程，但缺少 API 暴露层。前端无法直接触发多任务工作流，也无法获取结构化的执行计划和摘要结果。

## What Changes

1. **多任务 API 端点** - 新增 `POST /api/v1/admin/multi-task/plan`、`POST /api/v1/admin/multi-task/execute`、`GET /api/v1/admin/multi-task/plans`
2. **增强的摘要生成** - 优化 `buildMultiTaskSummary`，包含每个子任务使用的 Skill 和耗时信息
3. **计划持久化** - 将多任务计划存储到数据库，支持历史查询

## Capabilities

### New Capabilities

- `multi-task-plan-execute-summarize`: 多任务规划、执行、摘要 API 工作流

### Modified Capabilities

- 无

## Impact

- 新增 `internal/handler/multi_task_handler.go`
- 修改 `internal/router/api.go`
- 修改 `internal/agent/multi_task.go` 摘要生成
- 新增 `internal/models/multi_task.go` 数据模型
