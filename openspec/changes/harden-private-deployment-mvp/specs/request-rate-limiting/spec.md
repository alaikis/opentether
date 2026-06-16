## ADDED Requirements

### Requirement: Configurable request rate limiting
The system SHALL enforce request rate limiting when `security.rate_limit.enabled` is true.

#### Scenario: Requests exceed limit
- **WHEN** a client exceeds the configured requests-per-minute limit
- **THEN** the system rejects additional requests with HTTP 429 until the window resets

#### Scenario: Rate limiting disabled
- **WHEN** `security.rate_limit.enabled` is false
- **THEN** the system does not reject requests due to rate limits

### Requirement: Identity-aware rate limit keys
The system SHALL prefer authenticated user ID or API key ID for rate limit keys when available and fall back to client IP otherwise.

#### Scenario: API key request is limited
- **WHEN** a request is authenticated with an API key
- **THEN** the system applies the rate limit using the API key identity when possible

#### Scenario: Anonymous request is limited
- **WHEN** a request has no authenticated identity
- **THEN** the system applies the rate limit using the client IP

### Requirement: Rate limit response clarity
The system SHALL return a clear rate limit error response when a request is throttled.

#### Scenario: Throttled request
- **WHEN** the system rejects a request due to rate limiting
- **THEN** the response includes HTTP 429 and a machine-readable rate limit error
