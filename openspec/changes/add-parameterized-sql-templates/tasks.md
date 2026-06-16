## 1. Template Rendering

- [x] 1.1 Add runtime renderer for `{{start_date}}`, `{{end_date}}`, `{{current_year}}`, and `{{current_month}}`.
- [x] 1.2 Reject templates with unresolved variables.
- [x] 1.3 Select from multiple approved templates and skip non-matching templates.

## 2. Execution

- [x] 2.1 Add `RawSQL` support to Text2SQL QueryRequest.
- [x] 2.2 Execute rendered SQL through read-only validation, limits, audit, and data-boundary code path.
- [x] 2.3 Add live-query freshness metadata.

## 3. Validation

- [x] 3.1 Add unit tests for rendering and unresolved variables.
- [x] 3.2 Run changed backend package tests.
