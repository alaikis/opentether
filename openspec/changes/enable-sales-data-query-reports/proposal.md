## Why

The first usable private-deployment agent must support a concrete sales-data scenario: an employee can query only their own sales metrics, while administrators or authorized groups can query all sales data and generate real reports. Current Text2SQL has permission primitives, but report generation still returns placeholder data and the sales scenario is not guaranteed end-to-end.

## What Changes

- Add a sales-query contract for self, admin, and authorized-group data access.
- Enforce privileged full-data access through admin role or explicit full-access group membership.
- Execute report table/chart queries against configured external data sources instead of returning placeholders.
- Add tests for sales data scoping and report data resolution.

## Capabilities

### New Capabilities
- `sales-data-query-reporting`: Query scoped sales metrics and generate real report data for private enterprise sales scenarios.

### Modified Capabilities

## Impact

- Text2SQL permission resolution and query-plan SQL generation.
- Agent user context and group privilege checks.
- Report engine query resolution for table and chart sections.
- Tests for sales self-scope, all-data scope, and report execution.
