## Context

The platform has moved away from hard-coded business logic and now relies on Skill semantic configuration and LLM reasoning. This makes Skill configuration quality and runtime observability critical. The system also needs a unified chart output contract so results can render in web, IM fallback, and reports without relying on text parsing.

## Goals / Non-Goals

**Goals:**

- Validate Skill configuration before production use.
- Return structured chart payloads from query results.
- Record runtime timing for LLM/SQL/tool/FastPath paths.
- Provide basic eval cases/runs to check agent behavior.
- Support fallback LLM provider when the primary provider fails.
- Mark dynamic query results as live/stale-aware.

**Non-Goals:**

- Full report template designer.
- Full private deployment ops toolkit.
- Full SaaS billing/license/customer tenant management.
- Full external observability stack integration.

## Decisions

### Skill validation
Validation is implemented as a service that inspects config JSON, selected tables, relations, metric rules, entity rules, dimension rules, and data source existence. It returns warnings/errors without mutating Skill.

### Chart protocol
Text output remains for IM fallback, but `ChatResponse.Data` gets a standard chart object:

```json
{
  "type": "chart",
  "chart_type": "bar",
  "labels": [],
  "series": [{"name":"订单数","values":[],"unit":"单"}],
  "fallback_text": "..."
}
```

### Eval MVP
Eval cases store question, expected fragments, expected SQL fragments, and Skill ID. Eval runs execute cases and record pass/fail with output.

### Observability MVP
Timing metadata is returned and stored where available. Full Prometheus/OpenTelemetry integration is left for future changes.

### Provider fallback
LLM provider calls should try role-specific provider first, then active provider, then lower-priority enabled providers.

## Risks / Trade-offs

- Validation cannot prove query correctness; it catches configuration defects.
- Eval MVP starts simple and should later evolve into richer scoring.
- Chart detection remains heuristic until all Text2SQL paths emit structured data directly.
