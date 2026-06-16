## Context

Skill `config` contains business rules, semantic mappings, context MD pointers, metric rules, table relations, access groups, and data scope. A bad edit can break data access. The existing `context_md_versions` only protects markdown content, not the full Skill config.

## Goals / Non-Goals

**Goals:**

- Store complete Skill config snapshots.
- Keep snapshots tied to Skill ID, version, action, and optional actor.
- Allow admins to list versions and restore a selected version.
- Keep restores protected for built-in Skills outside development mode.

**Non-Goals:**

- Git-like diff/merge UI.
- Multi-user conflict resolution.
- External object storage for config snapshots.

## Decisions

Use a relational `SkillConfigVersion` model with full JSON config content. Snapshot version values use UTC timestamp strings to match context MD version style. Restoration first snapshots the current config as a restore backup, then applies the selected version.

## Risks / Trade-offs

- Full config snapshots may duplicate data, but configs are small and rollback reliability matters more.
- Timestamp versions can collide in the same second; service appends a nanosecond suffix when needed.
