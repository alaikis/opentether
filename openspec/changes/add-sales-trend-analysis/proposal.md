## Why

A business owner does not only ask for point-in-time numbers; they need reliable operating views such as year-to-date monthly order or performance trends. This must be deterministic for core sales questions instead of depending on the LLM to invent SQL each time.

## What Changes

- Add a deterministic sales trend fast path for year-to-date monthly trend questions.
- Support order-count and sales-performance metrics using the verified business relationship `t_order.sale_staff_id = t_profile.user_id`.
- Respect normal employee self-scope and privileged full-data access.
- Return chart-ready labels and values for frontend curve rendering.

## Capabilities

### New Capabilities
- `sales-trend-analysis`: Stable year-to-date monthly sales trend analysis for employees, administrators, and authorized groups.

### Modified Capabilities

## Impact

- Agent fast-path routing and deterministic SQL execution.
- Sales data permission handling.
- Chat response data payload for chart rendering.
- Tests for trend detection, metric selection, and response formatting.
