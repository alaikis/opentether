# Explicit Multi-Skill Planner

## Requirement

当用户提交包含多个子问题的查询时，系统需要显式地为每个子任务分配最合适的 Skill，而不是在执行时隐式路由。

## Acceptance Criteria

1. `SkillPlanner` 组件能够接收原始查询和子任务列表
2. 对每个子任务，使用向量匹配或关键词匹配找到最合适的 Skill
3. 返回的 `MultiTaskPlan` 中每个 `SubTask` 包含 `SkillUsed` 字段
4. 如果无法匹配到 Skill，子任务的 `SkillUsed` 为空字符串
5. `PlanSkills` 方法响应时间 < 100ms（使用缓存）
