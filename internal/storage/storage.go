// Package storage provides file storage with multiple backends:
//   - local: filesystem
//   - s3: S3-compatible (AWS S3 / Alibaba OSS / MinIO / Tencent COS / Huawei OBS etc.)
//
// Uses only stdlib — zero external dependencies.
package storage

import (
	"context"
)

// Driver abstracts a storage backend.
type Driver interface {
	// Save saves data and returns a publicly accessible URL.
	Save(ctx context.Context, path string, data []byte, contentType string) (url string, err error)
	// Delete removes an object.
	Delete(ctx context.Context, path string) error
	// Exists checks whether an object exists.
	Exists(ctx context.Context, path string) bool
	// PublicURL returns the public URL for a path without verifying existence.
	PublicURL(path string) string
}
