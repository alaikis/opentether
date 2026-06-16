## Context

OpenTether is currently a modular monolith for private enterprise agent deployment. It already has JWT authentication, API keys, request/audit logging, storage, setup, admin APIs, Agent Engine, Skills, Text2SQL, MCP, and IM integration points. For the first usable private-deployment version, the system must be safe enough to connect internal data sources and expose external integration APIs without requiring a full enterprise platform rewrite.

The design keeps the current monolith and adds a minimal control-plane baseline: readiness diagnostics, API key scope enforcement, and effective rate limiting. These changes are cross-cutting but intentionally small so they can be delivered before larger identity, policy, model-gateway, and evaluation initiatives.

## Goals / Non-Goals

**Goals:**

- Enforce API key scopes for external integration endpoints.
- Apply configurable request rate limiting using existing `security.rate_limit` settings.
- Provide a private deployment readiness report that detects unsafe configuration and unavailable core dependencies.
- Surface readiness through an authenticated admin API and a reusable service for future CLI/doctor commands.
- Add test coverage for the MVP hardening behavior.

**Non-Goals:**

- Full LDAP/OIDC/SAML identity center.
- Full ABAC policy engine or tenant-wide query rewriting.
- Full prompt-injection detection and tool-risk approval workflows.
- Full Prometheus/OpenTelemetry observability.
- Full backup/restore/upgrade CLI tooling.

## Decisions

### Add readiness as a service, not only a handler

Readiness logic will live in a service so it can be reused by admin APIs, setup flows, and future `doctor` CLI commands. The service will inspect configuration and dependency state without mutating system state.

Alternatives considered:
- Handler-only checks: faster but not reusable.
- Separate CLI now: useful, but too broad for the first hardening pass.

### Enforce scopes in middleware and route composition

API key validation already sets `auth_method`, `api_key_id`, and `scopes` in Fiber locals. A new scope middleware will check those values and reject missing scopes before handlers run. External integration routes will require API key authentication and specific scopes.

Alternatives considered:
- Handler-level scope checks: easier to miss and harder to test consistently.
- Full policy engine: better long term but too large for MVP.

### Implement simple in-memory rate limiting

Use an in-process rate limiter keyed by client IP and authenticated user/API-key identity when available. This is sufficient for single-node private deployment MVP and matches the current monolith architecture.

Alternatives considered:
- Redis distributed limiter: necessary for HA later, but introduces a new dependency.
- Fiber contrib limiter only by IP: simple but less useful for API-key/user throttling.

### Classify readiness checks by severity

Readiness checks will return `ok`, `warning`, or `critical`. The overall status is `ready` only when no critical checks exist. Warnings allow first use while making security debt visible.

## Risks / Trade-offs

- In-memory rate limiting resets on restart and is not shared across nodes → acceptable for single-node MVP; document as non-HA behavior.
- Readiness checks may flag deployments that intentionally use permissive settings in isolated labs → classify as warnings unless unsafe for real use.
- API key scope enforcement may break existing integrations without scopes → provide clear error messages and minimal required scopes.
- This change does not solve full enterprise identity or ABAC → keep follow-up OpenSpec changes for identity/policy center.
