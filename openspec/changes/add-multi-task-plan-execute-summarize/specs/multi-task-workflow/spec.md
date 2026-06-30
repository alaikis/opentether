# Multi-Task Plan-Execute-Summarize Workflow

## Requirement

将现有的多任务处理能力通过 API 暴露，支持前端显式触发多任务工作流。

## Acceptance Criteria

1. `POST /api/v1/admin/multi-task/plan` - 接收消息，返回多任务计划
2. `POST /api/v1/admin/multi-task/execute` - 执行多任务计划，返回结果和摘要
3. `GET /api/v1/admin/multi-task/plans` - 列出历史多任务计划
4. 摘要包含每个子任务的 Skill 使用信息和执行状态
5. 支持并行执行（最多 5 个并发子任务）
