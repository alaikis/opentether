## Why

The agent must support multi-dimensional sales analysis without baking one company's schema into platform code. Different private deployments have different table names, status dictionaries, shipment fields, amount fields, product fields, buyer fields, and permission groups. These must be expressed as Skill semantic configuration and business templates.

## What Changes

- Remove company-specific sales SQL from generic Agent fast paths.
- Treat the current company's schema as Skill configuration/business template data, not platform logic.
- Add configurable full-access group handling through Skill config instead of hard-coded group names.
- Keep the Agent role as: intent detection → read Skill semantic model/rules → query generation/execution → clarify missing fields.

## Capabilities

### New Capabilities
- `sales-multi-metric-analysis`: Multi-dimensional sales analysis driven by Skill semantic model, metric rules, field mappings, access policies, and business templates.

### Modified Capabilities
- Text2SQL access scope resolution uses `full_access_groups` / `allowed_all_scope_groups` from Skill config rather than hard-coded sales/data group names.

## Impact

- Agent permission scope resolution.
- Text2SQL Skill configuration contract.
- Tests proving unconfigured group names do not grant full access.
