## Context

`internal/agent/multi_task.go` 已实现：
- `BuildMultiTaskPlan` - 多部分查询拆分
- `BuildMultiTaskPlanWithLLM` - LLM 增强拆分
- `ExecuteMultiTaskPlan` - 并行/串行执行
- `buildMultiTaskSummary` - 结果摘要

但以上功能仅在 Agent Engine 内部使用，无 API 暴露。

## Goals / Non-Goals

**Goals:**
1. 暴露多任务规划 API
2. 暴露多任务执行 API
3. 增强摘要信息

**Non-Goals:**
- 不修改 Agent Engine 核心执行逻辑
- 不引入新的存储依赖

## Decisions

### 1. API 设计

**决定**: RESTful API，plan 和 execute 分离

**理由**: 前端可以先展示计划，用户确认后再执行

### 2. 摘要增强

**决定**: 在摘要中添加每个子任务的 Skill 使用信息和状态图标

**理由**: 提升可观测性
