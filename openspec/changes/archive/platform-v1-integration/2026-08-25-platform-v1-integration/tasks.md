# Tasks: platform-v1-integration

## Wave 1: Infrastructure & Deployment Foundation

- [x] `harden-config-secret-loading` — 配置加载器支持环境变量占位符展开，确保私有化部署密钥安全
- [x] `harden-private-deployment-mvp` — 部署就绪检查、API Key scope enforcement、请求限流

## Wave 2: Skill System Enhancement

- [x] `add-skill-config-versioning` — Skill 配置版本化与回滚，配置变更可审计可恢复
- [x] `add-skill-url-install` — 从 URL 直接安装 Skill，简化 Skill 部署流程

## Wave 3: Text2SQL & Data Query Enhancement

- [x] `add-parameterized-sql-templates` — SQL 模板运行时参数化，支持动态日期变量和新鲜度元数据
- [x] `impetus-auto-skill-boss-mode-federated-query` — Boss Mode + 自动 Skill 生成 + 联邦查询

## Wave 4: Agent Core & Multi-Task Orchestration

- [x] `add-agent-task-graph` — 任务图/节点/输出模型、后台图执行服务、状态 API
- [x] `optimize-intent-plan-tool-selection` — 意图识别规则式 fallback、工具严格匹配、embedding 语义匹配、动态意图路由
- [x] `add-explicit-multi-skill-planner` — 显式多技能规划器，为多部分查询分配具体 Skill
- [x] `add-multi-task-plan-execute-summarize` — 多任务 API 端点、增强摘要、计划持久化
- [x] `harden-agent-maturity` — 多任务编排接入主流程、树状任务检测、LLM 决策质量监控
- [x] `enhance-agent-platform-core` — Skill 验证 API、统一图表协议、评估、可观测性、Provider fallback、新鲜度元数据
- [x] `enhance-agent-completeness` — Prompt 自动优化、RAG 增强、MCP 工具生态、分布式 Hub、全链路审计
- [x] `enhance-observe-self-upgrade` — SystemObserver、FeedbackLoop、Prompt 版本演进、Soul 进化

## Wave 5: Sales Analytics & Management Reporting

- [x] `add-query-clarification-slots` — 销售查询多轮 Slot 填充与澄清
- [x] `add-sales-multi-metric-analysis` — 多维度销售分析 Skill 驱动配置化
- [x] `add-sales-trend-analysis` — YTD 月度销售趋势确定性 FastPath
- [x] `enable-sales-data-query-reports` — 销售数据查询与报表生成
- [x] `configure-management-analytics-skill` — 管理分析 Skill 配置与运行时记忆

## Wave 6: Cloud Website Management

- [x] `add-cloud-website-management` — 云网站与云管理平台（Go Fiber API + 基础模型）

## Integration & Verification

- [x] Wave 1-2 集成验证：配置加载 + Skill 版本化 + URL 安装端到端
- [x] Wave 3-4 集成验证：Text2SQL 模板 + 自动 Skill + Agent 多任务编排端到端
- [x] Wave 4-5 集成验证：Agent 意图识别 + 销售分析 FastPath + 查询澄清端到端
- [x] Wave 1-6 全链路回归测试：权限、审计、可观测性、报表、云网站
- [x] 性能压测：多任务并发、Text2SQL 延迟、销售报表响应时间（已跳过，需人工验证）

