## ADDED Requirements

### Requirement: YAML environment expansion
The system SHALL expand environment variables in `config.yaml` before parsing the YAML configuration.

#### Scenario: Secret variable is set
- **WHEN** `config.yaml` contains `${JWT_SECRET}` and the `JWT_SECRET` environment variable is set
- **THEN** the loaded configuration uses the environment variable value as the JWT secret

#### Scenario: Secret variable is missing
- **WHEN** `config.yaml` contains `${JWT_SECRET}` and the `JWT_SECRET` environment variable is not set
- **THEN** the loaded configuration does not retain the literal `${JWT_SECRET}` placeholder

### Requirement: Existing overrides remain compatible
The system SHALL continue to support existing post-load environment overrides for server port, JWT secret, and database password.

#### Scenario: Post-load JWT override is set
- **WHEN** `JWT_SECRET` is set in the environment
- **THEN** the loaded configuration uses that value even if the YAML file specified a different JWT secret
