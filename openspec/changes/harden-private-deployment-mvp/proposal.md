## Why

OpenTether already has the main agent, skill, Text2SQL, IM, MCP, audit, and admin modules, but the first private-deployment version needs a stronger safety and operability baseline before it can be used with real enterprise data. This change focuses on the minimum viable hardening needed for an internal enterprise deployment: safe configuration, enforced access scopes, basic operational diagnostics, and consistent security limits.

## What Changes

- Add a deployment readiness capability that validates critical configuration before and during operation.
- Add API key scope enforcement for external integration endpoints instead of only recording scopes in request context.
- Add effective request rate limiting middleware driven by existing configuration.
- Add private deployment diagnostics endpoints and service checks for database, storage, admin UI embedding, provider configuration, and security posture.
- Add safer defaults and warnings for weak JWT/encryption configuration, wildcard CORS, and disabled sandbox in private deployment mode.
- Add basic audit coverage for security-relevant configuration and external API key access decisions.

## Capabilities

### New Capabilities
- `private-deployment-readiness`: Validate and report whether a private deployment is safe enough for first production use.
- `api-key-scope-enforcement`: Enforce API key scopes on external integration APIs and privileged user API-key operations.
- `request-rate-limiting`: Apply configurable per-client request throttling for private deployments.

### Modified Capabilities

## Impact

- Backend middleware: authentication, API key handling, rate limiting, request logging.
- Backend services: setup/config diagnostics, API key validation, audit logging.
- Admin/system APIs: readiness/diagnostics endpoints and system configuration responses.
- Configuration: existing `security.rate_limit`, JWT, encryption, CORS, storage, executor sandbox settings become part of readiness checks.
- Tests: middleware and readiness service unit tests.
