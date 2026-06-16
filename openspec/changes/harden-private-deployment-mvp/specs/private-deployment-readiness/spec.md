## ADDED Requirements

### Requirement: Admin readiness report
The system SHALL provide an authenticated admin readiness report for private deployment that evaluates configuration and core dependencies without mutating system state.

#### Scenario: Admin requests readiness
- **WHEN** an authenticated admin requests the readiness report
- **THEN** the system returns an overall status and individual checks for database, storage, JWT secret, encryption key, CORS, rate limiting, sandbox, provider configuration, and admin UI availability

#### Scenario: Non-admin requests readiness
- **WHEN** a non-admin user requests the readiness report
- **THEN** the system denies the request

### Requirement: Readiness severity classification
The system SHALL classify readiness checks as `ok`, `warning`, or `critical` and derive the overall readiness status from those severities.

#### Scenario: Critical check exists
- **WHEN** at least one readiness check is `critical`
- **THEN** the overall readiness status is `not_ready`

#### Scenario: Only warnings exist
- **WHEN** no readiness check is `critical` and at least one check is `warning`
- **THEN** the overall readiness status is `ready_with_warnings`

#### Scenario: All checks pass
- **WHEN** all readiness checks are `ok`
- **THEN** the overall readiness status is `ready`

### Requirement: Security configuration checks
The system SHALL detect unsafe private deployment security settings including missing or placeholder JWT secrets, missing encryption keys, wildcard CORS, disabled rate limiting, and disabled sandbox.

#### Scenario: Placeholder JWT secret
- **WHEN** the configured JWT secret is empty or still contains an unresolved placeholder
- **THEN** the readiness report marks the JWT secret check as `critical`

#### Scenario: Wildcard CORS
- **WHEN** wildcard CORS is configured
- **THEN** the readiness report marks the CORS check as `warning`
