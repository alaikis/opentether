## Why

Approved SQL templates must be reusable across time. Hard-coded dates such as 2025 make historical results stale and cause future queries to reuse old periods. Templates should render runtime variables from the current request and execute live queries.

## What Changes

- Add runtime SQL template rendering for approved `text2sql_template` memories.
- Support dynamic date variables such as `{{start_date}}`, `{{end_date}}`, `{{current_year}}`, and `{{current_month}}`.
- Execute rendered SQL directly through Text2SQL safe execution instead of falling back to LLM SQL generation.
- Attach result freshness metadata so historical snapshots are distinguishable from live data.

## Capabilities

### New Capabilities
- `parameterized-sql-templates`: Approved SQL templates with runtime variables and live-query freshness metadata.

## Impact

- Agent FastPath template execution.
- Text2SQL QueryRequest supports safe raw SQL execution for already-approved templates.
- Tests for template rendering and unresolved variables.
