## ADDED Requirements

### Requirement: Skill validation
The system SHALL provide validation for Skill semantic configuration.

#### Scenario: Admin validates Skill
- **WHEN** an admin requests validation for a Skill
- **THEN** the system returns errors and warnings for missing data source, empty table selection, missing relations, missing metrics, and invalid config JSON

### Requirement: Unified chart protocol
The system SHALL return structured chart data when query results represent a chartable series.

#### Scenario: Monthly multi-series result
- **WHEN** a query returns a month column and multiple numeric columns
- **THEN** response data includes `type=chart`, labels, series, and chart type

### Requirement: Eval cases and runs
The system SHALL support basic Agent evaluation cases and runs.

#### Scenario: Admin creates eval case
- **WHEN** an admin creates an eval case for a Skill
- **THEN** the case is stored and can be executed later

### Requirement: Runtime timings
The system SHALL return runtime timing metadata for fast path and agent execution.

#### Scenario: Query executes through FastPath
- **WHEN** FastPath handles a query
- **THEN** response data includes timing metadata and `fast_path=true`

### Requirement: Dynamic result freshness
The system SHALL mark dynamic query results with generation time and freshness metadata.

#### Scenario: Query result returned
- **WHEN** a live database query returns
- **THEN** result data includes `generated_at` and `freshness=live_query`
