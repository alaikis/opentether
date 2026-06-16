## Context

The current test database contains a specific company's schema, but OpenTether is a private-deployment agent platform. Platform code must not assume `t_order`, `t_profile`, `delivery_time`, `sub_total`, MPN, SKU, buyer, or country columns. Those are tenant/company business semantics.

## Goals / Non-Goals

**Goals:**

- Keep platform Agent generic.
- Represent sales metrics, shipment口径, dimensions, filters, product fields, buyer fields, country fields, and full-access groups in Skill config.
- Allow business templates to provide deterministic behavior for a specific company without changing platform code.
- Make missing semantic config produce clarification or setup guidance.

**Non-Goals:**

- Hard-code current company's order status enum.
- Hard-code current company's sales tables or fields.
- Build a full visual semantic-model designer in this change.

## Decisions

Use existing Text2SQL semantic model, metric rules, entity rules, table relations, business rules, access policies, and selected Skill config as the source of truth. Full-data access is granted only by admin role or explicit Skill config group lists.

## Risks / Trade-offs

- Determinism depends on good Skill configuration or approved templates.
- Existing test data can be used to create one company template, but that template belongs in data/config, not generic Agent code.
