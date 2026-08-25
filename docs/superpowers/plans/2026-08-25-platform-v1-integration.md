---
change: platform-v1-integration
design-doc: docs/superpowers/specs/2026-08-25-platform-v1-integration-design.md
base-ref: ac2490ffece58923853ea850f544e28950f4773d
archived-with: 2026-08-25-platform-v1-integration
---

# platform-v1-integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate 20 sub-changes into a unified platform v1 delivery across 6 waves

**Architecture:** Incremental wave-based delivery where each wave builds on the previous wave's stabilized capabilities. Agent Core enhancements enable Sales Analytics capabilities; Skill System enhancements enable Text2SQL configuration; Infrastructure enables everything.

**Tech Stack:** Go + Fiber + GORM + MySQL + SvelteKit + TailwindCSS

## Global Constraints

- 所有数据库变更通过 GORM AutoMigrate 管理，按 wave 分批迁移
- Agent 核心增强通过 feature flag 或新 API 端点暴露，不影响现有对话流程
- 销售分析所有公司特定 schema 通过 Skill 配置管理，不硬编码到 Agent 代码
- 云网站初期 Go Fiber 内嵌 SvelteKit 构建产物，单二进制部署
- 中文需求/中文 workflow 下默认使用中文注释
- 测试覆盖：单元测试、集成测试、权限 scope 测试、FastPath 回归测试

archived-with: 2026-08-25-platform-v1-integration
---

### Wave 1: Infrastructure & Deployment Foundation

**Sub-changes:** `harden-config-secret-loading`, `harden-private-deployment-mvp`

#### Task 1.1: Config Secret Loading

**Files:**
- Modify: `internal/config/config.go`
- Create: `internal/config/secret_loader_test.go`

**Interfaces:**
- Consumes: `config.yaml` raw bytes, environment variables
- Produces: `LoadConfigWithSecrets() error` public method

- [ ] **Step 1: Write failing test for placeholder expansion**

```go
func TestLoadConfig_ExpandsEnvPlaceholders(t *testing.T) {
    t.Setenv("JWT_SECRET", "test-secret")
    yaml := "jwt: ${JWT_SECRET}"
    cfg, err := LoadConfigFromYAML([]byte(yaml))
    assert.NoError(t, err)
    assert.Equal(t, "test-secret", cfg.JWT)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/... -run TestLoadConfig_ExpandsEnvPlaceholders -v`
Expected: FAIL with `undefined: LoadConfigFromYAML` or placeholder not expanded

- [ ] **Step 3: Implement secret loader**

```go
func LoadConfigFromYAML(data []byte) (*Config, error) {
    expanded := expandEnvPlaceholders(string(data))
    var cfg Config
    if err := yaml.Unmarshal(expanded, &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}

func expandEnvPlaceholders(s string) string {
    re := regexp.MustCompile(`\$\{(\w+)\}`)
    return re.ReplaceAllStringFunc(s, func(m string) string {
        name := strings.TrimSuffix(strings.TrimPrefix(m, "${"), "}")
        if v := os.Getenv(name); v != "" {
            return v
        }
        return m
    })
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/... -run TestLoadConfig_ExpandsEnvPlaceholders -v`
Expected: PASS

#### Task 1.2: Private Deployment MVP

**Files:**
- Modify: `internal/config/config.go` (readiness structs)
- Create: `internal/service/readiness.go`, `internal/service/readiness_test.go`
- Create: `internal/middleware/rate_limit.go`, `internal/middleware/rate_limit_test.go`
- Create: `internal/middleware/api_key_scope.go`, `internal/middleware/api_key_scope_test.go`
- Modify: `internal/router/api.go` (register readiness routes, middleware)
- Modify: `config.yaml` (add `security.rate_limit`, `boss_mode`, readiness checks)

**Interfaces:**
- Consumes: `internal/config.ReadinessConfig`, `internal/config.SecurityConfig`
- Produces: `GET /api/v1/admin/readiness`, rate limit middleware, scope enforcement middleware

- [ ] **Step 1: Write failing tests for readiness service**

```go
func TestReadinessService_CheckConfig(t *testing.T) {
    svc := NewReadinessService(&Config{JWTSecret: "weak"})
    report := svc.Check()
    assert.False(t, report.Safe)
    assert.Contains(t, report.Issues, "weak JWT secret")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/... -run TestReadinessService -v`
Expected: FAIL

- [ ] **Step 3: Implement readiness service, middleware, routes**

Implement `internal/service/readiness.go` with checks for:
- JWT secret strength
- Encryption key presence
- CORS wildcard warning
- Database connectivity
- Storage availability
- Admin UI embedding configured

Implement `internal/middleware/rate_limit.go` using in-memory token bucket keyed by client IP / API key identity.

Implement `internal/middleware/api_key_scope.go` reading scopes from Fiber locals and rejecting requests missing required scopes.

Register routes in `internal/router/api.go`:
```go
admin := router.Group("/api/v1/admin")
admin.Get("/readiness", readinessHandler)
```

- [ ] **Step 4: Run tests and verify pass**

Run: `go test ./internal/service/... ./internal/middleware/... -v`
Expected: PASS

archived-with: 2026-08-25-platform-v1-integration
---

### Wave 2: Skill System Enhancement

**Sub-changes:** `add-skill-config-versioning`, `add-skill-url-install`

#### Task 2.1: Skill Config Versioning

**Files:**
- Modify: `internal/models/skill.go` (add `SkillConfigVersion` model)
- Create: `internal/service/skill_version.go`, `internal/service/skill_version_test.go`
- Modify: `internal/handler/skill_handler.go` (add version list/restore endpoints)
- Modify: `internal/router/api.go` (register new routes)

**Interfaces:**
- Consumes: `Skill` create/update events, context MD updates
- Produces: `GET /api/v1/admin/skills/:id/versions`, `POST /api/v1/admin/skills/:id/versions/:version/restore`

- [ ] **Step 1: Write failing tests for snapshot creation and restore**

```go
func TestSkillVersionService_SnapshotOnUpdate(t *testing.T) {
    svc := NewSkillVersionService(db)
    skill := &Skill{Name: "test"}
    err := svc.CreateSnapshot(ctx, skill, "update")
    assert.NoError(t, err)
    versions, _ := svc.ListVersions(ctx, skill.ID)
    assert.Len(t, versions, 1)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/... -run TestSkillVersionService -v`
Expected: FAIL

- [ ] **Step 3: Implement version model, service, handlers, routes**

Add `SkillConfigVersion` model with fields: `ID`, `SkillID`, `Version`, `ConfigSnapshot`, `CreatedAt`, `CreatedBy`.

Hook into Skill create/update/context MD update flows to call `CreateSnapshot`.

Add admin handlers for list versions and restore.

- [ ] **Step 4: Run tests and verify pass**

Run: `go test ./internal/service/... ./internal/handler/... -v`
Expected: PASS

#### Task 2.2: Skill URL Install

**Files:**
- Create: `internal/handler/skill_install_handler.go`, `internal/handler/skill_install_handler_test.go`
- Modify: `internal/router/api.go` (register `POST /api/v1/admin/skills/install-from-url`)
- Reuse: existing `fetchTextFromURL`, `UploadMarkdownAndCreateSkill` logic from `internal/service/skill_service.go`

**Interfaces:**
- Consumes: URL string in request body
- Produces: `POST /api/v1/admin/skills/install-from-url` → Skill created

- [ ] **Step 1: Write failing test for URL install**

```go
func TestInstallSkillFromURL(t *testing.T) {
    // mock HTTP server returning valid skill markdown
    req := httptest.NewRequest("POST", "/api/v1/admin/skills/install-from-url", strings.NewReader(`{"url":"http://localhost:9999/skill.md"}`))
    // ... assert 200 and skill created
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/handler/... -run TestInstallSkillFromURL -v`
Expected: FAIL

- [ ] **Step 3: Implement handler**

Implement handler that fetches URL content, parses markdown/JSON skill definition, calls existing `UploadMarkdownAndCreateSkill`.

- [ ] **Step 4: Run test and verify pass**

Run: `go test ./internal/handler/... -v`
Expected: PASS

archived-with: 2026-08-25-platform-v1-integration
---

### Wave 3: Text2SQL & Data Query Enhancement

**Sub-changes:** `add-parameterized-sql-templates`, `impetus-auto-skill-boss-mode-federated-query`

#### Task 3.1: Parameterized SQL Templates

**Files:**
- Modify: `internal/text2sql/text2sql.go` (template rendering before execution)
- Modify: `internal/models/query.go` (add `TemplateVariables` and `ResultFreshness` fields)
- Create: `internal/text2sql/template_renderer.go`, `internal/text2sql/template_renderer_test.go`

**Interfaces:**
- Consumes: approved `text2sql_template` memories with `{{start_date}}` etc.
- Produces: rendered SQL with runtime variables, `freshness: live|historical` metadata

- [ ] **Step 1: Write failing tests for template rendering**

```go
func TestRenderSQLTemplate(t *testing.T) {
    tmpl := "SELECT * FROM orders WHERE created_at >= '{{start_date}}'"
    vars := map[string]string{"start_date": "2026-01-01"}
    sql, err := RenderTemplate(tmpl, vars)
    assert.NoError(t, err)
    assert.Equal(t, "SELECT * FROM orders WHERE created_at >= '2026-01-01'", sql)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/text2sql/... -run TestRenderSQLTemplate -v`
Expected: FAIL

- [ ] **Step 3: Implement renderer and integrate into Text2SQL FastPath**

Implement `RenderTemplate` with `text/template`, support `{{start_date}}`, `{{end_date}}`, `{{current_year}}`, `{{current_month}}`.

When a `text2sql_template` memory is approved and executed, render variables from current request before passing to safe SQL execution.

- [ ] **Step 4: Run tests and verify pass**

Run: `go test ./internal/text2sql/... -v`
Expected: PASS

#### Task 3.2: Auto Skill, Boss Mode & Federated Query

**Files:**
- Create: `internal/service/auto_skill.go`, `internal/service/auto_skill_test.go`
- Modify: `internal/service/sql_audit.go` (add boss mode bypass check)
- Modify: `internal/text2sql/text2sql.go` (multi-DB schema injection, primary DB execution)
- Modify: `internal/service/setup.go` (wizard auto-skill trigger)
- Modify: `internal/config/config.go` (add `BossMode` struct)
- Modify: `config.yaml` (add `boss_mode` section)
- Modify: `internal/service/service.go` (extend `QueryRequest` with `DataSourceIDs`)

**Interfaces:**
- Consumes: DataSource connection events, `boss_mode.allowed_bypass_groups`, user group membership
- Produces: auto-generated generic query Skill, boss mode audit bypass, cross-DB schema context

- [ ] **Step 1: Write failing tests for boss mode bypass**

```go
func TestSQLAudit_BossModeBypass(t *testing.T) {
    svc := NewSQLAuditService(&Config{BossMode: BossModeConfig{AllowedBypassGroups: []string{"admin"}}})
    result := svc.Check(ctx, &User{Groups: []string{"admin"}})
    assert.Equal(t, ActionAutoApproved, result.Action)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/... -run TestSQLAudit_BossModeBypass -v`
Expected: FAIL

- [ ] **Step 3: Implement auto-skill service, boss mode, federated schema injection**

Auto Skill: on DataSource connection or setup wizard completion, call `AnalyzeDataSource`, generate markdown skill with full schema, register via `SkillFromMarkdownService`.

Boss Mode: in `sql_audit.go`, before recording audit, check if user group is in `allowed_bypass_groups`; if so, return auto-approved without persisting audit record.

Federated Query: extend `QueryRequest.DataSourceIDs`, in schema selector fetch schemas from all specified DataSources, concatenate into unified schema context for LLM prompt. SQL execution still targets primary DataSourceID.

- [ ] **Step 4: Run tests and verify pass**

Run: `go test ./internal/service/... ./internal/text2sql/... -v`
Expected: PASS

archived-with: 2026-08-25-platform-v1-integration
---

### Wave 4: Agent Core & Multi-Task Orchestration

**Sub-changes:** `add-agent-task-graph`, `optimize-intent-plan-tool-selection`, `add-explicit-multi-skill-planner`, `add-multi-task-plan-execute-summarize`, `harden-agent-maturity`, `enhance-agent-platform-core`, `enhance-agent-completeness`, `enhance-observe-self-upgrade`

#### Task 4.1: Agent Task Graph

**Files:**
- Create: `internal/models/task_graph.go`, `internal/models/task_graph_test.go`
- Create: `internal/agent/task_graph_executor.go`, `internal/agent/task_graph_executor_test.go`
- Modify: `internal/agent/agent_service.go` (graph execution integration)
- Modify: `internal/handler/task_graph_handler.go`, `internal/router/api.go`

**Interfaces:**
- Consumes: user goal string
- Produces: `agent_task_graphs`, `agent_task_nodes`, `agent_task_outputs` tables; create/status APIs

- [ ] **Step 1: Write failing tests for graph creation and execution**

```go
func TestTaskGraphExecutor_RunSimpleGraph(t *testing.T) {
    executor := NewTaskGraphExecutor(db, agentSvc)
    graph, err := executor.CreateGraph(ctx, "analyze sales and write report")
    assert.NoError(t, err)
    assert.Equal(t, StatusRunning, graph.Status)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/models/... ./internal/agent/... -run TestTaskGraphExecutor -v`
Expected: FAIL

- [ ] **Step 3: Implement models, executor, APIs**

Define GORM models for `agent_task_graphs`, `agent_task_nodes`, `agent_task_outputs`.

Implement default plan generator that splits goal into sequential nodes.

Implement async runner that executes each node via `AgentService` with constrained query, stores output summary and raw JSON.

Expose `POST /api/v1/admin/task-graphs`, `GET /api/v1/admin/task-graphs/:id`.

- [ ] **Step 4: Run tests and verify pass**

Run: `go test ./internal/models/... ./internal/agent/... -v`
Expected: PASS

#### Task 4.2: Intent Plan Tool Selection

**Files:**
- Modify: `internal/agent/engine.go` (`perceiveAndPlan` rule-based fallback, user memory injection)
- Modify: `internal/agent/loop.go` (`filterToolsByPlan` strict matching, tool→query feedback)
- Create: `internal/agent/semantic_match.go`, `internal/agent/semantic_match_test.go`
- Create: `internal/models/skill_intent_rule.go`

**Interfaces:**
- Consumes: `SkillRuntimeMemory`, user long-term memory, tool descriptions
- Produces: fallback skill mapping, embedding similarity scores, feedback records

- [ ] **Step 1: Write failing test for strict tool matching**

```go
func TestFilterToolsByPlan_StrictMatch(t *testing.T) {
    plan := Plan{Tools: []string{"nonexistent_tool"}}
    tools := []Tool{{Name: "real_tool"}}
    filtered := FilterToolsByPlan(tools, plan)
    assert.Empty(t, filtered)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/agent/... -run TestFilterToolsByPlan_StrictMatch -v`
Expected: FAIL

- [ ] **Step 3: Implement rule fallback, strict matching, semantic match, dynamic intent routing**

Add keyword→skill fallback map in `engine.go` used when LLM planning fails.

Change `filterToolsByPlan` to keep only tools whose `Name` exists in plan.

Implement `semantic_match.go` with embedding similarity (reuse existing embedding model or simple TF-IDF if no embedding service).

Add `skill_intent_rules` DB table for dynamic intent→skill mapping.

- [ ] **Step 4: Run tests and verify pass**

Run: `go test ./internal/agent/... -v`
Expected: PASS

#### Task 4.3: Explicit Multi-Skill Planner & Multi-Task API

**Files:**
- Create: `internal/agent/skill_planner.go`, `internal/agent/skill_planner_test.go`
- Create: `internal/handler/multi_task_handler.go`, `internal/handler/multi_task_handler_test.go`
- Modify: `internal/router/api.go` (register multi-task routes)
- Modify: `internal/agent/multi_task.go` (enhanced summary with skill usage and timing)

**Interfaces:**
- Consumes: multi-part user query
- Produces: `SkillPlanner` assignments, `POST /api/v1/admin/multi-task/plan`, `/execute`, `/plans`

- [ ] **Step 1: Write failing test for skill planner**

```go
func TestSkillPlanner_AssignSkills(t *testing.T) {
    planner := NewSkillPlanner(skillRepo)
    plan, err := planner.Plan(ctx, "查询销售额并生成图表")
    assert.NoError(t, err)
    assert.Equal(t, "sales-data-query-reporting", plan.SubTasks[0].SkillName)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/agent/... ./internal/handler/... -run TestSkillPlanner -v`
Expected: FAIL

- [ ] **Step 3: Implement planner, handler, enhanced summary**

`skill_planner.go`: analyze query parts, match each to best Skill using semantic model + keyword fallback.

`multi_task_handler.go`: expose plan/execute/plans endpoints, persist plans to DB.

Enhance `buildMultiTaskSummary` to include per-subtask skill name, status, and duration.

- [ ] **Step 4: Run tests and verify pass**

Run: `go test ./internal/agent/... ./internal/handler/... -v`
Expected: PASS

#### Task 4.4: Agent Maturity

**Files:**
- Modify: `internal/agent/multi_task.go` (integrate `ExecuteMultiTaskPlan` into main flow)
- Modify: `internal/agent/loop.go` (fix `detectTaskTree`, `extractContextWords`, `extractMetricFromQuery` stubs)
- Modify: `internal/agent/engine.go` (enhance `isCompleteToolResult` robustness)
- Modify: `internal/agent/self_learning.go` (add validation data injection and back-testing)
- Create: `internal/models/validation_data.go`

**Interfaces:**
- Consumes: user multi-part questions, tool output in various formats
- Produces: tree task detection, nested subtask dispatch, LLM decision quality metrics

- [ ] **Step 1: Write failing test for tree task detection**

```go
func TestDetectTaskTree_MultiPart(t *testing.T) {
    tree, err := DetectTaskTree("查询销售额并生成趋势图，同时列出top 10产品")
    assert.NoError(t, err)
    assert.True(t, tree.IsTree)
    assert.Len(t, tree.Children, 2)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/agent/... -run TestDetectTaskTree_MultiPart -v`
Expected: FAIL

- [ ] **Step 3: Implement tree detection, robust completion check, self-learning validation**

Replace stubs with real implementations using LLM classification + keyword patterns.

Add `isCompleteToolResult` to handle JSON, text, error, and empty output formats.

Add validation data injection API and periodic back-test loop in `self_learning.go`.

- [ ] **Step 4: Run tests and verify pass**

Run: `go test ./internal/agent/... -v`
Expected: PASS

#### Task 4.5: Platform Core Enhancements

**Files:**
- Create: `internal/models/eval.go`, `internal/models/observability.go`
- Create: `internal/service/eval.go`, `internal/service/eval_test.go`
- Create: `internal/service/observability.go`, `internal/service/observability_test.go`
- Modify: `internal/agent/engine.go` (timing logs, fallback provider selection)
- Modify: `internal/handler/validation_handler.go` (Skill validation API)
- Modify: `internal/router/api.go` (register eval, observability, validation routes)

**Interfaces:**
- Consumes: LLM calls, SQL executions, tool invocations
- Produces: `POST /api/v1/admin/skills/validate`, eval case/run APIs, runtime observability logs, freshness metadata

- [ ] **Step 1: Write failing tests for observability and eval**

```go
func TestObservabilityService_RecordLLMTiming(t *testing.T) {
    svc := NewObservabilityService(db)
    svc.RecordLLMCall(ctx, "gpt-4", 150*time.Millisecond, 1200, false)
    metrics, _ := svc.GetLLMMetrics(ctx, time.Minute)
    assert.Equal(t, 1, metrics.CallCount)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/... -run TestObservabilityService -v`
Expected: FAIL

- [ ] **Step 3: Implement models, services, handlers**

Add `agent_eval_cases`, `agent_eval_runs`, `agent_observability_logs` tables.

Implement timing instrumentation in Agent engine wrapping LLM/SQL/tool calls.

Add provider fallback: on LLM call failure, try next configured provider.

Add Skill validation API that checks semantic config against schema before publishing.

- [ ] **Step 4: Run tests and verify pass**

Run: `go test ./internal/service/... -v`
Expected: PASS

#### Task 4.6: Agent Completeness

**Files:**
- Create: `internal/agent/prompt_optimizer.go`, `internal/agent/prompt_optimizer_test.go`
- Create: `internal/rag/rag_enhanced.go`, `internal/rag/rag_enhanced_test.go`
- Create: `internal/mcp/mcp_ecosystem.go`, `internal/mcp/mcp_ecosystem_test.go`
- Create: `internal/distributed/hub.go`, `internal/distributed/hub_test.go`
- Create: `internal/agent/audit_logger.go`, `internal/agent/audit_logger_test.go`
- Modify: `internal/agent/observer.go` (enhance with skill/ds/llm metrics)
- Modify: `internal/router/api.go` (register new admin endpoints)

**Interfaces:**
- Consumes: failure logs, tool results, knowledge base
- Produces: prompt variants, incremental index, dynamic tool discovery, distributed task coordination, audit events

- [ ] **Step 1: Write failing tests for one subsystem (e.g., prompt optimizer)**

```go
func TestPromptAutoOptimizer_GenerateVariant(t *testing.T) {
    optimizer := NewPromptAutoOptimizer(db)
    variant, err := optimizer.GenerateVariant(ctx, "failed-query-123", FailureTypeSyntax)
    assert.NoError(t, err)
    assert.NotEmpty(t, variant.Prompt)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/agent/... -run TestPromptAutoOptimizer -v`
Expected: FAIL

- [ ] **Step 3: Implement subsystems one by one**

Prompt Optimizer: analyze failed queries, generate prompt variants via LLM, store for A/B testing.

RAG Enhanced: add incremental indexing, hybrid retrieval (keyword + embedding).

MCP Ecosystem: dynamic tool discovery, hot-load MCP server definitions.

Distributed Hub: task coordination, load balancing, state sync (in-process for MVP).

Audit Logger: structured audit events for security-relevant actions.

- [ ] **Step 4: Run tests and verify pass**

Run: `go test ./internal/agent/... ./internal/rag/... ./internal/mcp/... ./internal/distributed/... -v`
Expected: PASS

#### Task 4.7: Observe & Self-Upgrade

**Files:**
- Create: `internal/agent/observer.go` (already exists, enhance), `internal/agent/feedback_loop.go`, `internal/agent/prompt_evolution.go`, `internal/agent/soul_evolution.go`
- Modify: `internal/agent/memory.go` (implicit feedback collection)

**Interfaces:**
- Consumes: runtime metrics, failure classifications, user behavior signals
- Produces: `/api/v1/admin/observer/*` endpoints, dynamic thresholds, prompt version A/B testing, persona evolution

- [ ] **Step 1: Write failing tests for SystemObserver**

```go
func TestSystemObserver_SkillMetrics(t *testing.T) {
    obs := NewSystemObserver(db)
    obs.RecordSkillCall(ctx, "sales-data-query-reporting", 200*time.Millisecond, true)
    metrics, _ := obs.GetSkillMetrics(ctx, "sales-data-query-reporting")
    assert.Equal(t, 1, metrics.CallCount)
    assert.Equal(t, 200*time.Millisecond, metrics.AvgLatency)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/agent/... -run TestSystemObserver -v`
Expected: FAIL

- [ ] **Step 3: Implement observer, feedback loop, prompt evolution, soul evolution**

SystemObserver: collect Skill/DS/LLM metrics, store time-series.

FeedbackLoop: unified observation collection channel, insight processing pipeline.

PromptEvolution: dynamic threshold adjustment, A/B testing framework, auto-select best version.

SoulEvolution: implicit feedback (session length, follow-up rate), multi-dimensional persona evolution.

- [ ] **Step 4: Run tests and verify pass**

Run: `go test ./internal/agent/... -v`
Expected: PASS

archived-with: 2026-08-25-platform-v1-integration
---

### Wave 5: Sales Analytics & Management Reporting

**Sub-changes:** `add-query-clarification-slots`, `add-sales-multi-metric-analysis`, `add-sales-trend-analysis`, `enable-sales-data-query-reports`, `configure-management-analytics-skill`

#### Task 5.1: Query Clarification Slots

**Files:**
- Modify: `internal/agent/fast_path.go` (slot extraction before routing)
- Modify: `internal/agent/engine.go` (follow-up query rewriting with conversation memory)
- Create: `internal/agent/slot_extractor.go`, `internal/agent/slot_extractor_test.go`

**Interfaces:**
- Consumes: multi-turn sales queries, recent conversation memory
- Produces: extracted slots (metric, time range, subject, trend intent), clarification messages

- [ ] **Step 1: Write failing test for slot extraction**

```go
func TestSlotExtractor_SalesQuery(t *testing.T) {
    extractor := NewSlotExtractor()
    slots, err := extractor.Extract(ctx, "卖多少钱？", GetRecentMemory(ctx))
    assert.NoError(t, err)
    assert.Equal(t, MetricSalesAmount, slots.Metric)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/agent/... -run TestSlotExtractor -v`
Expected: FAIL

- [ ] **Step 3: Implement slot extractor and rewrite logic**

Deterministic slot extraction for metric, time range, subject, trend intent.

Follow-up rewrite inherits missing slots from recent conversation memory.

Replace empty-model fallback with actionable clarification message when required slots are missing.

- [ ] **Step 4: Run tests and verify pass**

Run: `go test ./internal/agent/... -v`
Expected: PASS

#### Task 5.2: Sales Multi-Metric Analysis

**Files:**
- Modify: `internal/text2sql/text2sql.go` (remove hard-coded sales SQL, use Skill config)
- Modify: `internal/service/service.go` (permission scope from `full_access_groups` / `allowed_all_scope_groups`)
- Create: `data/output/skills/context/sales-multi-metric-config.md` (Skill semantic model template)
- Modify: `data/opentether.db` Skill configuration

**Interfaces:**
- Consumes: Skill semantic model, metric rules, field mappings, access policies
- Produces: configurable multi-dimensional sales analysis queries

- [ ] **Step 1: Write failing tests for configurable access scope**

```go
func TestSalesScope_FromSkillConfig(t *testing.T) {
    scope := ResolveScope(ctx, &User{Groups: []string{"sales"}}, skillConfig)
    assert.False(t, scope.FullAccess) // unconfigured group should not grant full access
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/text2sql/... ./internal/service/... -run TestSalesScope -v`
Expected: FAIL

- [ ] **Step 3: Implement configurable scope resolution and metric rules**

Remove company-specific SQL from generic Agent fast paths.

Read `full_access_groups` / `allowed_all_scope_groups` from Skill config.

Add metric rules (order count, sales amount, performance) and field mappings to Skill context MD.

- [ ] **Step 4: Run tests and verify pass**

Run: `go test ./internal/text2sql/... ./internal/service/... -v`
Expected: PASS

#### Task 5.3: Sales Trend Analysis

**Files:**
- Modify: `internal/agent/fast_path.go` (add deterministic YTD monthly trend detection)
- Modify: `internal/models/response.go` (add chart-ready labels/values payload)
- Create: `internal/agent/sales_trend.go`, `internal/agent/sales_trend_test.go`

**Interfaces:**
- Consumes: `t_order.sale_staff_id = t_profile.user_id` relationship, employee/admin scope
- Produces: chart-ready `{labels: ["2026-01", ...], values: [123, ...]}` payload

- [ ] **Step 1: Write failing test for trend detection**

```go
func TestSalesTrendDetection_YTD(t *testing.T) {
    resp := HandleSalesTrendQuery(ctx, "今年每月的订单趋势")
    assert.True(t, resp.IsChart)
    assert.NotEmpty(t, resp.ChartData.Labels)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/agent/... -run TestSalesTrendDetection -v`
Expected: FAIL

- [ ] **Step 3: Implement deterministic trend FastPath**

Add intent detection for YTD monthly trend questions.

Execute deterministic SQL using verified business relationship.

Respect self-scope for employees, full-data for admins/authorized groups.

Return chart-ready payload.

- [ ] **Step 4: Run tests and verify pass**

Run: `go test ./internal/agent/... -v`
Expected: PASS

#### Task 5.4: Sales Data Query & Reporting

**Files:**
- Modify: `internal/text2sql/text2sql.go` (sales scoping enforcement)
- Modify: `internal/service/report.go` (real data resolution for table/chart sections)
- Create: `internal/service/report_test.go`

**Interfaces:**
- Consumes: user role, group membership, external data source configs
- Produces: scoped sales metrics, real report table/chart data

- [ ] **Step 1: Write failing tests for report data resolution**

```go
func TestReportEngine_SalesDataResolution(t *testing.T) {
    engine := NewReportEngine(db, dataSourceConfig)
    report, err := engine.Generate(ctx, "sales-summary", adminUser)
    assert.NoError(t, err)
    assert.NotEmpty(t, report.Tables[0].Rows)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/... -run TestReportEngine -v`
Expected: FAIL

- [ ] **Step 3: Implement scoped query execution and report resolution**

Enforce self-scope for employees, all-data scope for admins/authorized groups in Text2SQL.

Replace report placeholder data with real queries against configured external data sources.

- [ ] **Step 4: Run tests and verify pass**

Run: `go test ./internal/service/... -v`
Expected: PASS

#### Task 5.5: Management Analytics Skill Configuration

**Files:**
- Analyze: `data/opentether.db` business schema
- Create: `data/output/skills/context/management-analytics-context.md`
- Modify: `data/opentether.db` Skill configuration and runtime memories

**Interfaces:**
- Consumes: test database schema (sales, margin, cost, inventory, purchasing, customers, products, warehouse, after-sales, advertising)
- Produces: Skill configuration with domains, metrics, dimensions, entities, relations, business 口径

- [ ] **Step 1: Analyze schema and draft Skill context**

Use SQL introspection to enumerate tables, columns, foreign keys.

Draft context MD covering:
- Domains: sales, margin, cost, inventory, purchasing, customers, products, warehouse, after-sales, advertising
- Metrics per domain
- Dimensions and entities
- Relations
- Business 口径 (Chinese business terminology mappings)

- [ ] **Step 2: Seed runtime memories**

Create runtime memories for common management analytics patterns.

Add duplicate-calculation safeguards (e.g., "profit = revenue - cost - tax").

- [ ] **Step 3: Verify end-to-end management queries**

Run sample management analytics questions through Agent and validate answers use correct tables and calculations.

archived-with: 2026-08-25-platform-v1-integration
---

### Wave 6: Cloud Website Management

**Sub-changes:** `add-cloud-website-management`

#### Task 6.1: Cloud Backend Module

**Files:**
- Create: `internal/cloud/` package (models, services, handlers)
- Create: `internal/cloud/product.go`, `internal/cloud/version.go`, `internal/cloud/download.go`, `internal/cloud/admin.go`
- Modify: `internal/router/api.go` (register public and admin cloud routes)
- Modify: `config.yaml` (cloud website config)

**Interfaces:**
- Consumes: product info, version metadata, release files
- Produces: public website APIs, authenticated cloud admin APIs

- [ ] **Step 1: Write failing tests for product/version models and APIs**

```go
func TestCloudProductService_Create(t *testing.T) {
    svc := NewCloudProductService(db)
    p := &Product{Name: "Wisehoof", Slug: "wisehoof"}
    err := svc.Create(ctx, p)
    assert.NoError(t, err)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cloud/... -v`
Expected: FAIL

- [ ] **Step 3: Implement cloud backend**

Define models: `products`, `versions`, `release_files`, `downloads`, `site_content`.

Implement public APIs: product list, version history, download links.

Implement admin APIs: CRUD products, versions, upload release files, manage site content.

Embed SvelteKit build output in Go Fiber static file serving.

- [ ] **Step 4: Run tests and verify pass**

Run: `go test ./internal/cloud/... -v`
Expected: PASS

#### Task 6.2: SvelteKit Frontend (Parallel Track)

**Files:**
- Create: `admin-ui/src/routes/cloud/` (public website pages)
- Create: `admin-ui/src/routes/admin/cloud/` (admin console pages)
- Create: `admin-ui/src/lib/server/cloud/` (data loading functions)

**Interfaces:**
- Consumes: cloud backend APIs
- Produces: public website, admin console

- [ ] **Step 1: Scaffold SvelteKit routes and layouts**

Create `/cloud` (public home, products, versions, downloads) and `/admin/cloud` (admin dashboard, product management, version management, file upload).

- [ ] **Step 2: Implement pages with TailwindCSS + shadcn-style components**

Use existing admin-ui component library patterns.

- [ ] **Step 3: Wire up API data loading and form submission**

Connect to backend APIs created in Task 6.1.

- [ ] **Step 4: Verify embedded build works**

Run `npm run build` in `admin-ui`, verify Go backend serves built assets correctly.

archived-with: 2026-08-25-platform-v1-integration
---

### Integration & Final Verification

- [ ] Wave 1-2 集成验证：配置加载 + Skill 版本化 + URL 安装端到端
- [ ] Wave 3-4 集成验证：Text2SQL 模板 + 自动 Skill + Agent 多任务编排端到端
- [ ] Wave 4-5 集成验证：Agent 意图识别 + 销售分析 FastPath + 查询澄清端到端
- [ ] Wave 1-6 全链路回归测试：权限、审计、可观测性、报表、云网站
- [ ] 性能压测：多任务并发、Text2SQL 延迟、销售报表响应时间

archived-with: 2026-08-25-platform-v1-integration
---

## Final Build Commit

完成所有 task 且实现 Review 窗口通过后，按 commit convention 执行一次最终 build 提交：

- commit message 格式: `story#platform-v1-integration 平台 v1 整合交付：6 Wave 20 子变更统一实现`
- 描述语言: 中文（跟随 Spec 语言）
- issue_id: platform-v1-integration（非 null，自动提交）

