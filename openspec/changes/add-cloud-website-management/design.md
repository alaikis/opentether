## Context

The current project is a private-deployment enterprise agent platform with embedded Admin UI. The new cloud system should not disrupt the existing private deployment capabilities. It should add a cloud-facing product website and management surface, using the requested stack: Go Fiber + GORM + MySQL backend, SvelteKit + TailwindCSS + shadcn-style frontend.

## Goals / Non-Goals

**Goals:**

- Provide public website pages: home, product, docs/download entry, release list.
- Provide cloud admin pages: releases, artifacts, download links, site content.
- Store version metadata and downloadable artifacts in MySQL and object storage.
- Embed frontend assets in Go binary for initial deployment.
- Keep API-first boundaries so frontend can be separated later.

**Non-Goals:**

- Full SaaS billing.
- Full customer tenant management.
- Full marketplace or plugin store.
- Replacing existing local/private deployment admin UI.

## Architecture

```text
cloud backend
├── public APIs
│   ├── site content
│   ├── products
│   ├── releases
│   └── downloads
├── admin APIs
│   ├── content CRUD
│   ├── release CRUD
│   ├── artifact upload/bind
│   └── download stats
├── storage
│   ├── local/S3 object storage
│   └── public download URLs
└── frontend
    ├── public website routes
    └── cloud admin console routes
```

## Deployment Model

Initial mode:

```text
Go binary embeds SvelteKit static build
```

Future split mode:

```text
SvelteKit hosted separately → calls /api/cloud/*
Go backend serves only APIs
```

## Data Model Draft

- `cloud_products`: product name, slug, description, status.
- `cloud_releases`: product_id, version, channel, changelog, published_at, status.
- `cloud_artifacts`: release_id, os, arch, file_name, file_url, checksum, size.
- `cloud_download_logs`: artifact_id, ip, user_agent, downloaded_at.
- `cloud_site_contents`: key, title, body_md, status, updated_at.

## API Design

Public:

- `GET /api/cloud/public/products`
- `GET /api/cloud/public/releases`
- `GET /api/cloud/public/releases/:version`
- `GET /api/cloud/public/downloads/:artifact_id`
- `GET /api/cloud/public/site/:key`

Admin:

- `GET/POST/PUT/DELETE /api/cloud/admin/products`
- `GET/POST/PUT/DELETE /api/cloud/admin/releases`
- `GET/POST/PUT/DELETE /api/cloud/admin/artifacts`
- `GET/POST/PUT/DELETE /api/cloud/admin/site-contents`

## Risks / Trade-offs

- Embedding frontend simplifies deployment but requires rebuild for UI changes.
- Public download URLs require correctly configured storage base URL.
- Cloud features should be namespaced to avoid mixing with private deployment admin routes.
