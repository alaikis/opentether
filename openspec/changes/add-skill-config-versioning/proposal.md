## Why

Skill configuration is now the source of truth for company-specific business semantics. Editing mistakes can break production analytics, so full Skill config versioning and rollback are required, not only MD context versioning.

## What Changes

- Add `skill_config_versions` persistence model.
- Create config snapshots when Skills are created, updated, context MD is updated, or restored.
- Add admin APIs to list Skill config versions and restore a previous version.
- Preserve current built-in Skill protection rules.
- Add tests for snapshot creation and restore behavior.

## Capabilities

### New Capabilities
- `skill-config-versioning`: Version and restore complete Skill configuration.

### Modified Capabilities
- Skill create/update and context MD update now record recoverable configuration snapshots.

## Impact

- Database schema migration via GORM AutoMigrate.
- Skill service methods and admin handlers/routes.
- Audit events for config restoration.
