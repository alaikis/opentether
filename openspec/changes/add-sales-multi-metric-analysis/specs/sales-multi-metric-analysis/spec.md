## ADDED Requirements

### Requirement: Config-driven sales semantics
The system SHALL support multi-dimensional sales analysis through Skill semantic configuration rather than hard-coded platform SQL.

#### Scenario: Company-specific shipment口径
- **WHEN** a company defines shipment口径 in Skill business rules/default filters
- **THEN** the agent uses that configured semantic context instead of assuming a platform field name

#### Scenario: Product and buyer dimensions
- **WHEN** product category, type, buyer, MPN, SKU, or title-regex dimensions are configured in the Skill semantic model
- **THEN** the agent can use those dimensions through Text2SQL/query-plan generation

### Requirement: Configured full-data access
The system SHALL grant full-data query scope only to admins or groups explicitly configured in the Skill config.

#### Scenario: Unconfigured sales group
- **WHEN** a user belongs to a group named `sales_manager` but the Skill config does not grant that group full access
- **THEN** the user does not receive all-scope permissions

#### Scenario: Configured group
- **WHEN** a user's group ID/code/name appears in `full_access_groups` or `allowed_all_scope_groups`
- **THEN** the Skill may resolve query scope to `all`
