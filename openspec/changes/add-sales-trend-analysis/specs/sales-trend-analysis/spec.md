## ADDED Requirements

### Requirement: Year-to-date monthly sales trend
The system SHALL answer year-to-date monthly sales trend questions with deterministic chart-ready data.

#### Scenario: Monthly order-count trend
- **WHEN** a user asks for this year's monthly order-count trend
- **THEN** the system returns month labels and order-count values

#### Scenario: Monthly performance trend
- **WHEN** a user asks for this year's monthly performance or sales-amount trend
- **THEN** the system returns month labels and amount values

### Requirement: Sales trend permission scope
The system SHALL apply self-scope for normal users and full scope for administrators or authorized data/sales groups.

#### Scenario: Normal employee trend
- **WHEN** a normal employee asks for monthly sales trend
- **THEN** the query filters by the employee's business user ID

#### Scenario: Privileged trend
- **WHEN** an administrator or authorized group member asks for monthly sales trend
- **THEN** the query may aggregate all corresponding sales data
