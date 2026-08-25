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
