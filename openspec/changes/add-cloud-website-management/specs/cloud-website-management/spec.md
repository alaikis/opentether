## ADDED Requirements

### Requirement: Public website content
The system SHALL expose public website APIs for product and site content.

#### Scenario: Visitor opens product website
- **WHEN** a visitor requests public product/site content
- **THEN** the system returns published content without requiring authentication

### Requirement: Unified version management
The system SHALL allow administrators to manage product versions and release metadata.

#### Scenario: Admin publishes release
- **WHEN** an admin creates a release with artifacts
- **THEN** the release appears in public release APIs after being marked published

### Requirement: Download center
The system SHALL expose download endpoints for release artifacts and record download logs.

#### Scenario: User downloads artifact
- **WHEN** a user downloads an artifact
- **THEN** the system redirects or returns the artifact URL and records a download log

### Requirement: Embedded-first frontend
The system SHALL support embedding the SvelteKit static build into the Go backend.

#### Scenario: Single binary deployment
- **WHEN** the backend binary starts in embedded mode
- **THEN** it serves the cloud website and cloud admin frontend assets

### Requirement: Future frontend/backend separation
The system SHALL keep frontend/backend boundaries API-first so the frontend can be hosted separately later.

#### Scenario: Split frontend deployment
- **WHEN** the frontend is deployed separately
- **THEN** it can consume the same `/api/cloud/*` APIs without backend UI coupling
