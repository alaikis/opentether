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
