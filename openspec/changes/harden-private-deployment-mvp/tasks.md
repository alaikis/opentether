## 1. Readiness Service and API

- [x] 1.1 Add a deployment readiness service that returns overall status and check results.
- [x] 1.2 Implement checks for database, storage, JWT secret, encryption key, CORS, rate limit, sandbox, provider configuration, and admin UI mode.
- [x] 1.3 Add an authenticated admin readiness endpoint under `/api/v1/admin/system/readiness`.
- [x] 1.4 Add unit tests for readiness status derivation and security configuration checks.

## 2. API Key Scope Enforcement

- [x] 2.1 Add API key scope middleware that rejects requests without required scopes.
- [x] 2.2 Require API key authentication for `/api/v1/external` endpoints.
- [x] 2.3 Assign minimal external scopes for bind IM, list users, and confirm IM binding endpoints.
- [x] 2.4 Add tests for missing API key, missing scope, and allowed scoped API key requests.

## 3. Request Rate Limiting

- [x] 3.1 Add configurable in-memory request rate limiting middleware.
- [x] 3.2 Use API key ID, user ID, or client IP as the rate limit identity key.
- [x] 3.3 Wire rate limiting into the main middleware stack using existing config.
- [x] 3.4 Add tests for enabled, disabled, and exceeded rate limit behavior.

## 4. Validation

- [x] 4.1 Run Go unit tests for changed backend packages.
- [x] 4.2 Run full backend test suite with `go test ./...`.
- [x] 4.3 Run frontend typecheck/build only if frontend files change.
