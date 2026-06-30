## 1. Model

- [x] 1.1 创建 internal/models/multi_task.go，定义 MultiTaskPlanModel 和 MultiTaskExecutionModel

## 2. Handler

- [x] 2.1 创建 internal/handler/multi_task_handler.go
- [x] 2.2 实现 PlanMultiTask handler
- [x] 2.3 实现 ExecuteMultiTask handler
- [x] 2.4 实现 ListMultiTaskPlans handler

## 3. Router

- [x] 3.1 在 internal/router/api.go 注册 /api/v1/admin/multi-task 路由

## 4. Agent Integration

- [x] 4.1 在 AgentEngine 中暴露 BuildMultiTaskPlan 和 ExecuteMultiTaskPlan
- [x] 4.2 增强 buildMultiTaskSummary 包含 Skill 信息

## 5. Tests

- [x] 5.1 编写 handler 集成测试
- [x] 5.2 编写 API 路由测试
