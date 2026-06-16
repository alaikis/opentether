## 1. Query Slot Filling

- [x] 1.1 Implement local sales query slot extraction for metric, subject, and time range.
- [x] 1.2 Implement follow-up rewrite using recent conversation memory.
- [x] 1.3 Return clarification responses for underspecified short sales queries.

## 2. Empty Response Recovery

- [x] 2.1 Replace query-like empty fallback with actionable clarification text.
- [x] 2.2 Add tests for empty fallback recovery.

## 3. Validation

- [x] 3.1 Add unit tests for slot extraction, rewrite, and clarification behavior.
- [x] 3.2 Run changed backend package tests.
- [x] 3.3 Run full backend test suite with `go test ./...`.
