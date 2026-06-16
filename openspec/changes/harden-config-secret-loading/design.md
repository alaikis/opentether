## Context

`config.yaml` documents secrets as `${JWT_SECRET}`, `${ENCRYPTION_KEY}`, and similar values. Go's YAML unmarshal currently reads those as literal strings, which can accidentally make placeholder text the active secret. Private deployments need a predictable secret-loading path that works in offline installations and service managers.

## Goals / Non-Goals

**Goals:**

- Expand `${VAR}` and `$VAR` values in `config.yaml` using the process environment before unmarshalling.
- Preserve existing post-unmarshal overrides such as `JWT_SECRET`, `SERVER_PORT`, and `DB_PASSWORD`.
- Make missing environment variables resolve to empty strings so readiness checks can flag them.

**Non-Goals:**

- Introduce external secret managers or Vault integrations.
- Encrypt the YAML file at rest.
- Change the public config schema.

## Decisions

Use `os.ExpandEnv` on raw YAML bytes before `yaml.Unmarshal`. This keeps behavior simple and applies consistently to all YAML fields without adding reflection-based post-processing.

## Risks / Trade-offs

- Missing variables become empty strings → readiness checks and tests cover this explicitly.
- Literal dollar signs in YAML require escaping by operators → acceptable for deployment configuration.
