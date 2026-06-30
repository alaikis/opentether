## ADDED Requirements

### Requirement: All operations are logged
The system SHALL log all auditable operations with sufficient detail for compliance and investigation.

#### Scenario: Log user login
- **WHEN** a user successfully authenticates
- **THEN** the system SHALL record timestamp, user_id, source_ip, and auth_method

#### Scenario: Log data access
- **WHEN** user queries data through the agent
- **THEN** the system SHALL record the query, data source, and rows accessed

#### Scenario: Log configuration change
- **WHEN** admin modifies system configuration
- **THEN** the system SHALL record changed keys, old values, new values, and admin_id

### Requirement: Audit logs are immutable
The system SHALL ensure audit logs cannot be modified or deleted after creation.

#### Scenario: Attempt to delete audit log
- **WHEN** any user or process attempts to delete audit log entries
- **THEN** the system SHALL reject the request and log the attempted violation

### Requirement: Audit logs support compliance reporting
The system SHALL generate compliance reports from audit logs.

#### Scenario: Generate audit report
- **WHEN** user requests audit report for a date range
- **THEN** the system SHALL generate a PDF/CSV report with all relevant events

### Requirement: Audit logs support forensic query
The system SHALL provide query capabilities for investigating security incidents.

#### Scenario: Search audit logs by user
- **WHEN** admin queries audit logs filtered by user_id
- **THEN** the system SHALL return all events associated with that user

#### Scenario: Search audit logs by IP
- **WHEN** admin queries audit logs filtered by source_ip
- **THEN** the system SHALL return all events originating from that IP

### Requirement: Audit logs are exported to external storage
The system SHALL support exporting audit logs to S3-compatible storage for long-term retention.

#### Scenario: Configure audit export
- **WHEN** admin sets audit.export.enabled=true and configures S3 credentials
- **THEN** the system SHALL automatically export audit logs to S3 in batches