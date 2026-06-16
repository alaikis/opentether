## Why

Private deployments must load secrets from environment variables without leaving placeholder strings in runtime configuration. The current YAML examples use `${JWT_SECRET}` and similar placeholders, but the configuration loader does not expand them before unmarshalling.

## What Changes

- Expand environment variables in `config.yaml` before YAML parsing.
- Keep explicit environment variable overrides for compatibility.
- Add tests for placeholder expansion and missing secret behavior.

## Capabilities

### New Capabilities
- `config-secret-loading`: Load private-deployment secrets from environment variables safely and predictably.

### Modified Capabilities

## Impact

- Backend configuration loader.
- Tests for config loading behavior.
- Readiness checks benefit because unresolved secrets become empty or weak values instead of literal placeholders.
