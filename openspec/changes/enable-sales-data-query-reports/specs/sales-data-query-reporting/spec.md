## ADDED Requirements

### Requirement: Employee sales self-scope
The system SHALL allow an employee to query sales metrics such as performance and order count while filtering results to that employee's own business identity.

#### Scenario: Employee queries own order count
- **WHEN** a non-admin employee asks for their order count
- **THEN** the generated SQL includes a self-scope data boundary based on the employee identity mapping

### Requirement: Privileged full sales access
The system SHALL allow administrators and designated full-access groups to query all corresponding sales data without self-scope filtering.

#### Scenario: Admin queries all sales data
- **WHEN** an administrator asks for sales performance data
- **THEN** the generated query is allowed to use `all` data scope

#### Scenario: Authorized group queries all sales data
- **WHEN** a user belongs to a configured sales/data/admin group
- **THEN** the generated query is allowed to use `all` data scope

### Requirement: Real report query execution
The system SHALL execute report table and chart section queries against the configured datasource instead of returning placeholder data.

#### Scenario: Table report section executes query
- **WHEN** a report table section has a query and a datasource ID
- **THEN** the report engine returns headers and rows from the datasource query

#### Scenario: Chart report section executes query
- **WHEN** a chart section has a label column and value column
- **THEN** the report engine returns labels and numeric values from the datasource query
