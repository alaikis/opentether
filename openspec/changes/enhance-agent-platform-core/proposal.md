## Why

The platform already has Skills, Text2SQL, MCP, IM, memory, reports, storage, and FastPath. To become a robust enterprise agent platform beyond private-deployment packaging, it needs platform-level core enhancements: Skill validation, unified chart protocol, lightweight evaluation, runtime observability, provider fallback, and structured memory freshness rules.

This change intentionally excludes private-deployment operations such as systemd installers, offline upgrade packs, license activation, backup/restore, and disaster recovery.

## What Changes

- Add Skill validation APIs to detect invalid semantic configuration before publishing.
- Add a unified chart response protocol for web, IM, and reports.
- Add basic Agent evaluation case/run models and APIs.
- Add runtime observability logs for LLM, SQL, tool, and FastPath timing.
- Add provider fallback support for LLM calls.
- Add freshness metadata for dynamic query results to reduce stale memory pollution.

## Capabilities

### New Capabilities
- `agent-platform-core`: Core platform enhancements for validation, chart protocol, evaluation, observability, fallback, and freshness metadata.

## Impact

- Backend models, services, handlers, and routes.
- Agent response data shape for charts and timings.
- Admin APIs for validation and eval.
- Frontend can progressively adopt chart protocol and eval pages later.
