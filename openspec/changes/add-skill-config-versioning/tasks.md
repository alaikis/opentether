## 1. Persistence

- [x] 1.1 Add SkillConfigVersion model and AutoMigrate registration.
- [x] 1.2 Add snapshot helper in SkillService.

## 2. Service and API

- [x] 2.1 Snapshot config on Skill create/update and context MD update.
- [x] 2.2 Add SkillService list versions method.
- [x] 2.3 Add SkillService restore version method.
- [x] 2.4 Add admin handlers/routes for listing and restoring versions.

## 3. Validation

- [x] 3.1 Add tests for snapshot and restore behavior.
- [x] 3.2 Run changed backend package tests.
- [x] 3.3 Run full backend test suite with `go test ./...`.
