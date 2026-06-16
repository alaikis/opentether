## 1. Backend Foundation

- [x] 1.1 Add cloud models: product, release, artifact, download log, site content.
- [x] 1.2 Add GORM AutoMigrate registration.
- [x] 1.3 Add cloud service layer for CRUD and download URL resolution.
- [x] 1.4 Add public cloud APIs under `/api/cloud/public`.
- [x] 1.5 Add admin cloud APIs under `/api/cloud/admin` with admin auth.

## 2. Frontend Foundation

- [x] 2.1 Add public website routes using SvelteKit + TailwindCSS.
- [x] 2.2 Add cloud admin routes for releases/artifacts/site content.
- [x] 2.3 Keep API client configurable for future split deployment.
- [x] 2.4 Ensure static build can be embedded by current Go binary.

## 3. Version and Download Flow

- [x] 3.1 Implement release publish/unpublish.
- [x] 3.2 Implement artifact metadata and storage URL binding.
- [x] 3.3 Implement download redirect/logging.
- [x] 3.4 Add basic download statistics.

## 4. Validation

- [x] 4.1 Run backend tests.
- [x] 4.2 Run frontend check/build.
- [x] 4.3 Build Linux binary with embedded frontend.
