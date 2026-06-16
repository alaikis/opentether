## 1. Remove Hard-coded Business SQL

- [x] 1.1 Remove company-specific sales multi-metric fast path from generic Agent code.
- [x] 1.2 Remove company-specific sales trend SQL from generic Agent code.
- [x] 1.3 Remove temporary company schema probing code.

## 2. Config-driven Access Scope

- [x] 2.1 Replace hard-coded full-access group names with Skill config-driven groups.
- [x] 2.2 Add tests proving unconfigured group names do not grant full access.

## 3. Validation

- [x] 3.1 Run changed backend package tests.
- [x] 3.2 Run full backend test suite with `go test ./...`.
