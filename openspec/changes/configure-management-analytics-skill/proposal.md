## Why

Company leadership needs broad operational analytics across sales, margin, cost, inventory, purchasing, customers, products, warehouse, after-sales, and advertising. These business semantics belong in Skill configuration and runtime memory, not generic Agent code.

## What Changes

- Analyze the current test database business schema.
- Generate a management analytics Skill context MD.
- Update the current company Skill with management domains, metrics, dimensions, entities, relations, and business口径.
- Seed runtime memories for common management analytics patterns and duplicate-calculation safeguards.

## Capabilities

### New Capabilities
- `management-analytics-skill`: Company-specific management analytics configuration for the test database.

## Impact

- `data/opentether.db` Skill configuration and runtime memories.
- `data/output/skills/context/...` generated context MD.
- No generic Agent hard-coding.
