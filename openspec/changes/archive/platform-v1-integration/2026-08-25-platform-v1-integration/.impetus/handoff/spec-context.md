# Impetus Spec Projection

- Change: platform-v1-integration
- Phase: design
- Issue: platform-v1-integration
- Mode: beta
- Context hash: 39dbf7e910ca607ce12805dc0a952eb06054d2195818ce74a39e2f9479b00226

Generated-by: impetus-handoff.sh

OpenSpec delta specs remain canonical. This file is a deterministic full-text projection for lower-token Build handoff; regenerate it when source artifacts change.

## openspec/changes/platform-v1-integration/proposal.md

- Role: proposal
- Source: openspec/changes/platform-v1-integration/proposal.md
- Lines: 1-116
- SHA256: e500b876e44aba87a775bbea1dffa15ad50a20576e5a5aeed5c996338dd6f240

```md
## Why

当前 `openspec/changes/` 下有 20 个活跃变更，覆盖 Agent 核心、销售分析、Skill 系统、Text2SQL、平台基础设施和云网站管理。这些变更目标一致但彼此独立推进，存在以下问题：

- 主题高度重叠：多任务编排、意图识别、销售分析能力分散在多个变更中，缺乏统一视角
- 依赖关系不清晰：部分变更需要前置能力（如多任务 API 需要任务图模型先就绪，销售分析需要 Text2SQL 权限模型先稳定）
- 无法统一评估总工作量、优先级和交付节奏：各自独立的 tasks.md 缺乏全局 wave 规划
- 变更间潜在冲突：如 `optimize-intent-plan-tool-selection` 与 `harden-agent-maturity` 都涉及 `internal/agent/` 核心文件

本变更将这 20 个变更整合为统一的产品路线图，按主题群和依赖关系拆分为可交付的 wave，确保能力有序交付、避免重复设计冲突。

## What Changes

将 20 个活跃变更按以下 5 个主题群归并，统一规划、统一交付：

### A. Agent 核心 / 多任务 / 意图规划

- `add-agent-task-graph`：任务图模型、图执行服务、状态 API
- `add-explicit-multi-skill-planner`：显式多技能规划器，为多部分查询分配具体 Skill
- `add-multi-task-plan-execute-summarize`：多任务 plan/execute/summarize API 端点
- `harden-agent-maturity`：多任务编排接入主流程、树状任务检测、LLM 决策监控
- `optimize-intent-plan-tool-selection`：意图识别增强、语义工具匹配、动态意图路由
- `enhance-agent-platform-core`：Skill 验证、统一图表协议、评估、可观测性、fallback
- `enhance-agent-completeness`：任务图可视化、Prompt 自动优化、RAG、MCP 生态、分布式 Hub、审计
- `enhance-observe-self-upgrade`：SystemObserver、FeedbackLoop、Prompt 版本演进、Soul 进化

### B. 销售分析 / 管理报表

- `add-sales-multi-metric-analysis`：多维度销售分析 Skill 驱动
- `add-sales-trend-analysis`：YTD 月度销售趋势确定性 FastPath
- `enable-sales-data-query-reports`：销售数据查询与报表生成
- `configure-management-analytics-skill`：管理分析 Skill 配置

### C. Skill 系统

- `add-skill-config-versioning`：Skill 配置版本化与回滚
- `add-skill-url-install`：从 URL 安装 Skill
- `add-query-clarification-slots`：多轮销售查询 Slot 填充与澄清

### D. Text2SQL / 数据查询

- `add-parameterized-sql-templates`：SQL 模板参数化与运行时渲染
- `impetus-auto-skill-boss-mode-federated-query`：自动 Skill 生成、Boss Mode、联邦查询

### E. 平台 / 部署 / 安全

- `harden-config-secret-loading`：配置密钥环境变量展开
- `harden-private-deployment-mvp`：私有化部署就绪检查、API Key scope enforcement、限流
- `add-cloud-website-management`：云网站与云管理平台（Go Fiber + SvelteKit）

## Capabilities

### New Capabilities

- `agent-task-graph`：长任务图执行、checkpoint、输出引用、状态 API
- `explicit-multi-skill-planner`：显式多技能规划器
- `multi-task-plan-execute-summarize`：多任务规划、执行、摘要 API 工作流
- `multi-task-orchestration`：多任务编排执行，支持用户多部分问题自动拆分并行执行
- `semantic-tool-matching`：基于 embedding 的工具语义匹配
- `dynamic-intent-routing`：动态意图路由规则，数据库配置 intent→skill 映射
- `agent-platform-core`：核心平台增强（验证、图表协议、评估、可观测性、fallback、新鲜度）
- `task-graph-visualization`：任务图实时可视化与执行追踪
- `prompt-auto-optimizer`：基于失败模式的 Prompt 自动优化
- `rag-enhanced`：增强型知识库 RAG 系统
- `mcp-ecosystem`：MCP 工具生态系统
- `distributed-hub`：分布式 Agent 协作中心
- `auto-tuning`：自动超参数调优引擎
- `audit-logging`：全链路安全审计系统
- `system-observer`：系统级监控组件
- `feedback-loop`：统一反馈循环框架
- `dynamic-prompt-evolution`：Prompt 版本动态演进
- `enhanced-soul-evolution`：增强的用户画像进化
- `sales-multi-metric-analysis`：多维度销售分析
- `sales-trend-analysis`：YTD 月度销售趋势
- `sales-data-query-reporting`：销售数据查询与报表
- `management-analytics-skill`：管理分析 Skill 配置
- `query-clarification-slots`：多轮查询 Slot 填充
- `parameterized-sql-templates`：SQL 模板参数化
- `auto-skill-generation`：自动生成并注册查询 Skill
- `boss-mode`：特定组跳过 SQL 审计
- `federated-query`：跨数据源联邦查询
- `setup-wizard-enhancement`：设置向导自动 Skill 生成
- `skill-config-versioning`：Skill 配置版本化与回滚
- `skill-url-install`：从 URL 安装 Skill
- `config-secret-loading`：环境变量密钥加载
- `private-deployment-readiness`：私有化部署就绪检查
- `api-key-scope-enforcement`：API Key 范围 enforcement
- `request-rate-limiting`：请求速率限制
- `cloud-website-management`：云网站与云管理平台

### Modified Capabilities

- `agent-core`：增强 ReAct Loop 鲁棒性、early-stop 条件、工具输出自适应
- `self-learning`：增强为实时失败分类和即时反馈
- `text2sql`：支持多 DataSourceID、Boss Mode 跳过审计
- `sql-audit`：检查 boss_mode.allowed_bypass_groups
- `data-source-connection`：连接后触发自动 Skill 生成

## Impact

- 统一的任务规划和 wave 交付节奏，跨主题能力复用（如 Agent 核心增强直接赋能销售分析 FastPath）
- 数据库 schema 增量演变：多任务、Skill 版本、查询模板、联邦查询、权限、云管理
- 后端服务层扩展：`internal/agent/`、`internal/text2sql/`、`internal/service/`、`internal/handler/`、`internal/models/`、`internal/router/`
- 前端扩展：SvelteKit 多任务可视化、销售报表图表、云管理控制台
- 配置与安全增强：密钥加载、Boss Mode、API Key scope、限流
- 测试覆盖：单元测试、集成测试、权限 scope 测试、FastPath 回归测试

## Scope

### In Scope
- 上述 20 个变更的全部能力范围
- 跨变更依赖整合与 wave 规划
- 统一的测试和验证策略

### Out of Scope
- 无（全部 20 个子变更均已纳入）
```

## openspec/changes/platform-v1-integration/design.md

- Role: open-design-notes
- Source: openspec/changes/platform-v1-integration/design.md
- Lines: 1-122
- SHA256: b4a1a72e8275db526cc990e2f9322086cf92cc7b5ec4a7d9e24e6612c10ad660

```md
## Context

OpenTether 当前处于多变更并行推进状态：Agent 核心能力、销售分析、Skill 系统、Text2SQL、平台基础设施和云网站管理各有独立 proposal 和 design，但彼此存在能力重叠、依赖交叉和潜在冲突。本 Design Doc 将这 20 个变更整合为统一架构，按主题群和依赖关系分 wave 交付。

各子变更的详细设计文档保留作为子文档引用，本文件聚焦跨主题整合架构、依赖关系和统一技术决策。

## Sub-Change Design References

| 主题群 | 子变更 | 设计文档 |
|--------|--------|---------|
| A. Agent 核心 | `add-agent-task-graph` | `openspec/changes/add-agent-task-graph/design.md` |
| A. Agent 核心 | `add-explicit-multi-skill-planner` | `openspec/changes/add-explicit-multi-skill-planner/design.md` |
| A. Agent 核心 | `add-multi-task-plan-execute-summarize` | `openspec/changes/add-multi-task-plan-execute-summarize/design.md` |
| A. Agent 核心 | `harden-agent-maturity` | `openspec/changes/harden-agent-maturity/design.md` |
| A. Agent 核心 | `optimize-intent-plan-tool-selection` | `openspec/changes/optimize-intent-plan-tool-selection/design.md` |
| A. Agent 核心 | `enhance-agent-platform-core` | `openspec/changes/enhance-agent-platform-core/design.md` |
| A. Agent 核心 | `enhance-agent-completeness` | `openspec/changes/enhance-agent-completeness/design.md` |
| A. Agent 核心 | `enhance-observe-self-upgrade` | `openspec/changes/enhance-observe-self-upgrade/design.md` |
| B. 销售分析 | `add-sales-multi-metric-analysis` | `openspec/changes/add-sales-multi-metric-analysis/design.md` |
| B. 销售分析 | `add-sales-trend-analysis` | `openspec/changes/add-sales-trend-analysis/design.md` |
| B. 销售分析 | `enable-sales-data-query-reports` | `openspec/changes/enable-sales-data-query-reports/design.md` |
| B. 销售分析 | `configure-management-analytics-skill` | `openspec/changes/configure-management-analytics-skill/design.md` |
| C. Skill 系统 | `add-skill-config-versioning` | `openspec/changes/add-skill-config-versioning/design.md` |
| C. Skill 系统 | `add-skill-url-install` | `openspec/changes/add-skill-url-install/design.md` |
| C. Skill 系统 | `add-query-clarification-slots` | `openspec/changes/add-query-clarification-slots/design.md` |
| D. Text2SQL | `add-parameterized-sql-templates` | `openspec/changes/add-parameterized-sql-templates/design.md` |
| D. Text2SQL | `impetus-auto-skill-boss-mode-federated-query` | `openspec/changes/impetus-auto-skill-boss-mode-federated-query/design.md` |
| E. 平台/部署 | `harden-config-secret-loading` | `openspec/changes/harden-config-secret-loading/design.md` |
| E. 平台/部署 | `harden-private-deployment-mvp` | `openspec/changes/harden-private-deployment-mvp/design.md` |
| E. 平台/部署 | `add-cloud-website-management` | `openspec/changes/add-cloud-website-management/design.md` |

## Goals / Non-Goals

**Goals:**
- 将 20 个分散变更整合为统一 wave 交付计划，消除能力重叠和依赖冲突
- 建立跨主题能力复用机制（Agent 核心增强赋能销售分析 FastPath，Skill 系统增强赋能 Text2SQL 配置化）
- 保持各子变更原有设计意图不变，仅调整执行顺序和整合点
- 提供全局测试和验证策略

**Non-Goals:**
- 改变任何子变更的核心技术方案
- 新增超出 20 个子变更范围的能力
- 引入新的外部技术栈依赖（云网站模块除外，其设计已包含 SvelteKit）

## Integration Architecture

### Layered View

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                         │
│  Cloud Website (SvelteKit) │ Admin Console │ Public Site     │
├─────────────────────────────────────────────────────────────┤
│                    Capability Layer                          │
│  Sales Analytics │ Multi-Task │ Skill Mgmt │ Query Engine    │
├─────────────────────────────────────────────────────────────┤
│                    Agent Core Layer                          │
│  Task Graph │ Intent/Tool │ Observability │ Self-Learning    │
├─────────────────────────────────────────────────────────────┤
│                    Platform Layer                            │
│  Validation │ Chart Protocol │ Eval │ Fallback │ Freshness  │
├─────────────────────────────────────────────────────────────┤
│                    Infrastructure Layer                      │
│  Config │ Security │ Rate Limit │ Readiness │ DB │ Storage  │
└─────────────────────────────────────────────────────────────┘
```

### Cross-Cutting Integration Points

1. **Agent Core → Sales Analytics**
   - `add-agent-task-graph` 提供任务图模型，`add-sales-trend-analysis` 和 `add-sales-multi-metric-analysis` 的复杂分析可作为长任务图节点执行
   - `optimize-intent-plan-tool-selection` 的意图识别直接决定销售查询路由到 FastPath 还是 LLM 生成
   - `add-query-clarification-slots` 的 Slot 填充在 Agent fast-path preprocessing 层接入

2. **Skill System → Text2SQL / Sales**
   - `add-skill-config-versioning` 为 `configure-management-analytics-skill` 和 `add-sales-multi-metric-analysis` 提供配置版本化管理
   - `add-skill-url-install` 简化 Skill 部署流程
   - `impetus-auto-skill-boss-mode-federated-query` 的自动 Skill 生成依赖 Skill 注册服务，后者属于 Skill System

3. **Platform Core → All Themes**
   - `enhance-agent-platform-core` 的 Skill validation API 在 Skill 发布前校验配置正确性
   - 统一 chart protocol 被销售分析、任务可视化、云网站共用
   - 可观测性指标（LLM/SQL/Tool 耗时）覆盖所有能力层

4. **Infrastructure → Everything**
   - `harden-config-secret-loading` 是 `harden-private-deployment-mvp` 的前置
   - `harden-private-deployment-mvp` 的 readiness 检查应覆盖云网站模块
   - Boss Mode 配置属于基础设施配置层

## Delivery Waves

| Wave | 主题群 | 子变更 | 依赖前置 |
|------|--------|--------|---------|
| 1 | E. 平台/部署 | `harden-config-secret-loading` → `harden-private-deployment-mvp` | 无 |
| 2 | C. Skill 系统 | `add-skill-config-versioning` → `add-skill-url-install` | Wave 1 |
| 3 | D. Text2SQL | `add-parameterized-sql-templates` → `impetus-auto-skill-boss-mode-federated-query` | Wave 2 |
| 4 | A. Agent 核心 | `add-agent-task-graph` → `optimize-intent-plan-tool-selection` → `add-explicit-multi-skill-planner` → `add-multi-task-plan-execute-summarize` → `harden-agent-maturity` → `enhance-agent-platform-core` → `enhance-agent-completeness` → `enhance-observe-self-upgrade` | Wave 3 |
| 5 | B. 销售分析 | `add-query-clarification-slots` → `add-sales-multi-metric-analysis` → `add-sales-trend-analysis` → `enable-sales-data-query-reports` → `configure-management-analytics-skill` | Wave 4 |
| 6 | E. 云网站 | `add-cloud-website-management` | Wave 1, 2, 3, 4, 5（并行开发，集成在最后） |

## Key Technical Decisions

### D1: 数据库 Schema 增量迁移
所有新增表通过 GORM AutoMigrate 管理，按 wave 交付时分批添加。避免一次性大迁移。

### D2: Agent 核心增强不破坏现有 FastPath
意图识别、工具匹配、任务图的增强通过 feature flag 或新 API 端点暴露，不影响现有对话流程。主流程切换在 `harden-agent-maturity` 阶段由用户确认后执行。

### D3: 销售分析配置化
所有公司特定 schema（表名、状态字典、金额字段、权限组）通过 Skill 配置管理，不硬编码到 Agent 代码。`configure-management-analytics-skill` 为测试库生成初始配置模板。

### D4: 云网站嵌入式部署
初期 Go Fiber 后端内嵌 SvelteKit 构建产物，单二进制部署。API 边界保持前后端分离，为未来独立部署留空间。

## Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| 20 个子变更体量过大，单次交付周期过长 | Wave 规划确保每 wave 可独立验证和交付 |
| 子变更间代码冲突（特别是 `internal/agent/`） | Wave 4 内部按文件修改范围拆分 task，避免并行修改同一文件 |
| 销售分析依赖 Text2SQL 权限模型 | Wave 3 先稳定 Text2SQL 权限和配置化，Wave 5 再叠加销售场景 |
| 云网站技术栈差异大 | Wave 6 独立分支开发，最后集成 |
```

## openspec/changes/platform-v1-integration/tasks.md

- Role: tasks
- Source: openspec/changes/platform-v1-integration/tasks.md
- Lines: 1-47
- SHA256: 4772f10d3146bbe3e83135c39665bc0985b42328631821b8718bcee193eeed3e

```md
# Tasks: platform-v1-integration

## Wave 1: Infrastructure & Deployment Foundation

- [ ] `harden-config-secret-loading` — 配置加载器支持环境变量占位符展开，确保私有化部署密钥安全
- [ ] `harden-private-deployment-mvp` — 部署就绪检查、API Key scope enforcement、请求限流

## Wave 2: Skill System Enhancement

- [ ] `add-skill-config-versioning` — Skill 配置版本化与回滚，配置变更可审计可恢复
- [ ] `add-skill-url-install` — 从 URL 直接安装 Skill，简化 Skill 部署流程

## Wave 3: Text2SQL & Data Query Enhancement

- [ ] `add-parameterized-sql-templates` — SQL 模板运行时参数化，支持动态日期变量和新鲜度元数据
- [ ] `impetus-auto-skill-boss-mode-federated-query` — 数据源连接后自动生成 Skill、Boss Mode 跳过 SQL 审计、多数据源联邦查询

## Wave 4: Agent Core & Multi-Task Orchestration

- [ ] `add-agent-task-graph` — 任务图/节点/输出模型、后台图执行服务、状态 API
- [ ] `optimize-intent-plan-tool-selection` — 意图识别规则式 fallback、工具严格匹配、embedding 语义匹配、动态意图路由
- [ ] `add-explicit-multi-skill-planner` — 显式多技能规划器，为多部分查询分配具体 Skill
- [ ] `add-multi-task-plan-execute-summarize` — 多任务 API 端点、增强摘要、计划持久化
- [ ] `harden-agent-maturity` — 多任务编排接入主流程、树状任务检测、LLM 决策质量监控
- [ ] `enhance-agent-platform-core` — Skill 验证 API、统一图表协议、评估模型、可观测性日志、Provider fallback、新鲜度元数据
- [ ] `enhance-agent-completeness` — 任务图可视化、Prompt 自动优化、RAG 增强、MCP 工具生态、分布式 Hub、全链路审计
- [ ] `enhance-observe-self-upgrade` — SystemObserver、FeedbackLoop、Prompt 版本演进、Soul 进化

## Wave 5: Sales Analytics & Management Reporting

- [ ] `add-query-clarification-slots` — 销售查询多轮 Slot 填充与澄清
- [ ] `add-sales-multi-metric-analysis` — 多维度销售分析 Skill 驱动配置化
- [ ] `add-sales-trend-analysis` — YTD 月度销售趋势确定性 FastPath
- [ ] `enable-sales-data-query-reports` — 销售数据权限 Scope 查询与报表生成
- [ ] `configure-management-analytics-skill` — 管理分析 Skill 配置与运行时记忆

## Wave 6: Cloud Website Management

- [ ] `add-cloud-website-management` — 云网站与云管理平台（Go Fiber + SvelteKit、嵌入式前端、版本管理、下载中心）

## Integration & Verification

- [ ] Wave 1-2 集成验证：配置加载 + Skill 版本化 + URL 安装端到端
- [ ] Wave 3-4 集成验证：Text2SQL 模板 + 自动 Skill + Agent 多任务编排端到端
- [ ] Wave 4-5 集成验证：Agent 意图识别 + 销售分析 FastPath + 查询澄清端到端
- [ ] Wave 1-6 全链路回归测试：权限、审计、可观测性、报表、云网站
- [ ] 性能压测：多任务并发、Text2SQL 延迟、销售报表响应时间
```

