## Context

当前多任务处理模块 (`internal/agent/multi_task.go`) 已具备基础的多部分查询拆分和执行能力，但存在以下不足：
1. 子任务到 Skill 的映射是隐式的（在执行时通过 `ExecuteLoop` 自动路由），用户无法预知
2. 缺少显式的技能规划器组件
3. 多任务工作流缺少 API 暴露，前端无法直接调用

## Goals / Non-Goals

**Goals:**
1. 实现显式多技能规划器，为每个子任务分配最匹配的 Skill
2. 暴露多任务 plan-execute-summarize API 端点
3. 增强摘要生成，包含 Skill 使用信息

**Non-Goals:**
- 不修改现有 `ExecuteLoop` 核心逻辑
- 不改变现有 Skill 路由机制
- 不引入外部依赖

## Decisions

### 1. 显式多技能规划器

**决定**: 新增 `SkillPlanner` 结构体，使用现有 `SkillManager` 的向量匹配能力为子任务分配 Skill

**理由**: 
- 复用现有 `SkillManager.MatchByVector` 和 `ListEnabledSkills`
- 最小化代码改动

**备选**: 使用 LLM 进行 Skill 分配 - 增加延迟和成本

### 2. API 设计

**决定**: 新增 `POST /api/v1/admin/multi-task/plan` 和 `POST /api/v1/admin/multi-task/execute`

**理由**: 
- 与现有 admin API 风格一致
- 分离规划和执行阶段，便于前端控制

## Risks / Trade-offs

[风险] 显式规划可能增加首字延迟 → mitigation: 缓存规划结果

[风险] 向量匹配可能不准确 → mitigation: 设置合理的 similarity threshold (0.3)

## Migration Plan

1. 第一阶段: 新增 `SkillPlanner` 组件
2. 第二阶段: 新增 API 端点和 Handler
3. 第三阶段: 集成测试和文档更新
