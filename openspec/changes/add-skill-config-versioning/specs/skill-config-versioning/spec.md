## ADDED Requirements

### Requirement: Skill config snapshots
The system SHALL persist complete Skill config snapshots when a Skill is created or updated.

#### Scenario: Skill update
- **WHEN** an administrator updates a Skill config
- **THEN** the system stores the resulting config as a new version

### Requirement: Skill config version listing
The system SHALL expose an admin API to list config versions for a Skill.

#### Scenario: Admin lists versions
- **WHEN** an admin requests versions for a Skill
- **THEN** the system returns versions ordered newest first

### Requirement: Skill config restore
The system SHALL expose an admin API to restore a Skill config from a previous version.

#### Scenario: Admin restores version
- **WHEN** an admin restores a previous config version
- **THEN** the system snapshots the current config and applies the selected version

#### Scenario: Built-in Skill protected
- **WHEN** a non-development deployment attempts to restore a built-in Skill
- **THEN** the system rejects the restore using the existing built-in Skill protection
