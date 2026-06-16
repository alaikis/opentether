## Context

The verified production-like data model uses `t_profile.user_id` as the true business user ID and `t_order.sale_staff_id` as the order sales-owner field. Boss-level questions such as “今年以来按月的变化曲线” need stable SQL and chart payloads.

## Goals / Non-Goals

**Goals:**

- Detect year-to-date monthly sales trend questions.
- Generate deterministic read-only SQL for monthly order count and performance.
- Apply self-scope for ordinary employees and all-scope for admins or full-access groups.
- Return chart-ready labels and values.

**Non-Goals:**

- Build a full BI semantic layer.
- Support every possible chart type.
- Replace the general Text2SQL path.

## Decisions

Use an agent fast path before generic Text2SQL. The fast path is intentionally narrow: it triggers only when the question contains year-to-date language, monthly/trend language, and sales/order/performance language. SQL uses fixed safe joins and aggregates.

## Risks / Trade-offs

- Narrow matching avoids false positives but may miss unusual phrasing.
- Metric definitions are based on current verified schema and may need extension for future datasets.
