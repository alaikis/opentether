package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// LocalStorage saves files to a local directory.
type LocalStorage struct {
	root    string // absolute path, e.g. /app/data/output
	baseURL string // e.g. http://localhost:8886/downloads
}

func NewLocalStorage(root, baseURL string) (*LocalStorage, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return nil, fmt.Errorf("create storage dir %s: %w", abs, err)
	}
	return &LocalStorage{root: abs, baseURL: baseURL}, nil
}

func (s *LocalStorage) Save(ctx context.Context, path string, data []byte, contentType string) (string, error) {
	fullPath := filepath.Join(s.root, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", err
	}
	return s.PublicURL(path), nil
}

func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	return os.Remove(filepath.Join(s.root, path))
}

func (s *LocalStorage) Exists(ctx context.Context, path string) bool {
	_, err := os.Stat(filepath.Join(s.root, path))
	return err == nil
}

func (s *LocalStorage) PublicURL(path string) string {
	// Normalize Windows backslashes
	urlPath := filepath.ToSlash(path)
	return s.baseURL + "/" + urlPath
}
