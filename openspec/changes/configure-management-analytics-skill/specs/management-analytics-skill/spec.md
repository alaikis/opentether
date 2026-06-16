## ADDED Requirements

### Requirement: Management analytics configuration
The system SHALL provide company-specific Skill configuration covering management analytics domains.

#### Scenario: Leadership asks broad analytics
- **WHEN** leadership asks about sales, profit, cost, inventory, purchase, customer, product, warehouse, after-sales, or advertising
- **THEN** the Skill provides relevant table, metric, dimension, and relationship context to Text2SQL

### Requirement: Duplicate calculation safeguards
The Skill configuration SHALL include guidance to prevent duplicate counting and duplicated amount aggregation.

#### Scenario: Joining order items
- **WHEN** a query joins order items or amount rows
- **THEN** order counts use `COUNT(DISTINCT t_order.id)` and amount calculations avoid Cartesian multiplication
