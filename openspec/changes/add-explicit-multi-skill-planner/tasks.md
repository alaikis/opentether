## 1. Skill Planner

- [x] 1.1 创建 internal/agent/skill_planner.go，定义 SkillPlanner 结构体
- [x] 1.2 实现 PlanSkills(query string, subTasks []SubTask) 方法，为每个子任务分配 Skill
- [x] 1.3 在 AgentEngine 中集成 SkillPlanner
- [x] 1.4 为 BuildMultiTaskPlanWithLLM 添加 Skill 分配逻辑

## 2. API Endpoints

- [x] 2.1 创建 internal/handler/multi_task_handler.go
- [x] 2.2 实现 PlanMultiTask handler
- [x] 2.3 实现 ExecuteMultiTask handler
- [x] 2.4 在 internal/router/api.go 注册路由

## 3. Tests

- [x] 3.1 编写 SkillPlanner 单元测试
- [x] 3.2 编写 multi-task handler 集成测试
