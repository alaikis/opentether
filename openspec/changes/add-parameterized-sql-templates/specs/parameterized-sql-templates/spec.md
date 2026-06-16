## ADDED Requirements

### Requirement: Runtime date variables
The system SHALL render approved SQL templates using runtime date variables from the current request.

#### Scenario: 1 to 6 month range
- **WHEN** a template contains `{{start_date}}` and `{{end_date}}` and the user asks for `1至6月`
- **THEN** the rendered SQL uses the current year's `YYYY-01-01` and `YYYY-07-01`

### Requirement: Reject unresolved variables
The system SHALL reject unresolved template variables and fall back to normal Agent execution.

#### Scenario: Unknown variable
- **WHEN** a template still contains `{{unknown}}` after rendering
- **THEN** FastPath does not execute that template

### Requirement: Live query freshness metadata
The system SHALL mark template query results as live data.

#### Scenario: Template result returned
- **WHEN** a parameterized template executes successfully
- **THEN** the response data includes `template_rendered=true`, `generated_at`, and `freshness=live_query`
