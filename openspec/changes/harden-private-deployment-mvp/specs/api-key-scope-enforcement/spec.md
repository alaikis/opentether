## ADDED Requirements

### Requirement: API key authentication for external integrations
The system SHALL require valid API key authentication for external integration endpoints under `/api/v1/external`.

#### Scenario: External endpoint without API key
- **WHEN** a request reaches an external integration endpoint without a valid `X-API-Key`
- **THEN** the system rejects the request with an API-key-required error

#### Scenario: External endpoint with valid API key
- **WHEN** a request reaches an external integration endpoint with a valid `X-API-Key`
- **THEN** the system authenticates the request as the API key owner before handler execution

### Requirement: API key scope enforcement
The system SHALL enforce required API key scopes before executing scoped endpoints.

#### Scenario: API key lacks required scope
- **WHEN** an API key authenticated request does not include the endpoint's required scope
- **THEN** the system rejects the request with a forbidden scope error

#### Scenario: API key has required scope
- **WHEN** an API key authenticated request includes the endpoint's required scope
- **THEN** the system allows the request to continue

### Requirement: Scope-aware auditing
The system SHALL make API key identity and scope decisions available for audit logging.

#### Scenario: Scope check fails
- **WHEN** a scoped API key request is rejected because required scopes are missing
- **THEN** the system records enough request context to identify the API key and missing scope
