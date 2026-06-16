## Why

OpenTether needs a cloud-facing system for the official website and cloud management platform. It should publish product information, versions, downloads, release metadata, and later support cloud-side customer/license/device/tenant management. The initial architecture should embed the SvelteKit frontend into the Go backend for simple deployment, while keeping APIs and build artifacts separable for future frontend/backend split deployment.

## What Changes

- Add a cloud system module using Go Fiber + GORM + MySQL.
- Add a SvelteKit + TailwindCSS + shadcn-style frontend for website and admin console.
- Support embedded frontend assets in backend for early-stage single-binary deployment.
- Design API boundaries so frontend can later run independently.
- Add core cloud-domain models for product site content, versions, release files, downloads, and admin management.
- Add public website APIs and authenticated cloud admin APIs.

## Capabilities

### New Capabilities
- `cloud-website-management`: Official website, cloud admin, unified version management, and download center.

## Impact

- New backend module/packages for cloud website and release management.
- New frontend routes/pages for public website and cloud admin console.
- Database schema additions for releases, downloads, files, products, and site content.
- Build pipeline should support both embedded and decoupled deployment modes.
