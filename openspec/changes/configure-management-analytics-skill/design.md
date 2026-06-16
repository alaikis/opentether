## Context

The current company schema contains order, order item, amount, profit, inventory, purchase, customer, address, product, warehouse, after-sales, and advertising tables. The platform should stay generic while the company-specific Skill stores mappings and metrics.

## Goals / Non-Goals

**Goals:**

- Configure leadership-level metrics and dimensions.
- Preserve delivery_time as default sales口径.
- Encode duplicate-calculation safeguards.
- Include profit/cost, inventory, purchase, customer, product, after-sales, and advertising analysis.

**Non-Goals:**

- Hard-code company schema in Agent code.
- Build a visual semantic model editor.

## Decisions

Store the management analytics semantics in Skill config, generated context MD, and runtime memories. Use `COUNT(DISTINCT t_order.id)` for order counts and explicit metric definitions to avoid duplicate JOIN aggregation.

## Risks / Trade-offs

- Some tables may need additional manual validation before production use.
- Text2SQL quality still depends on model performance and the richness of Skill context.
