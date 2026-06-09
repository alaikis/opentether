package main

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

func TestEmbeddedAdminUIFileSystemContainsRequiredFiles(t *testing.T) {
	subFS := embeddedAdminUITestFS(t)

	for _, file := range []string{"index.html", "setup/index.html", "_app/version.json"} {
		if _, err := fs.Stat(subFS, file); err != nil {
			t.Fatalf("embedded admin UI missing %q: %v", file, err)
		}
	}

	foundEntryJS, err := hasEmbeddedFile(subFS, "_app/immutable/entry", ".js")
	if err != nil {
		t.Fatalf("failed to inspect embedded entry assets: %v", err)
	}
	if !foundEntryJS {
		t.Fatalf("embedded admin UI is missing SvelteKit _app entry JavaScript files; use //go:embed all:admin-ui/build")
	}
}

func TestEmbeddedAdminUIIndexReferencesExistingAssets(t *testing.T) {
	subFS := embeddedAdminUITestFS(t)

	index, err := fs.ReadFile(subFS, "index.html")
	if err != nil {
		t.Fatalf("failed to read embedded index.html: %v", err)
	}

	assetPattern := regexp.MustCompile(`/admin/(_app/[^"'<>\s]+)`)
	matches := assetPattern.FindAllStringSubmatch(string(index), -1)
	if len(matches) == 0 {
		t.Fatalf("embedded index.html does not reference any /admin/_app assets")
	}

	seen := make(map[string]struct{})
	for _, match := range matches {
		assetPath := strings.TrimSpace(match[1])
		if _, ok := seen[assetPath]; ok {
			continue
		}
		seen[assetPath] = struct{}{}

		if _, err := fs.Stat(subFS, assetPath); err != nil {
			t.Fatalf("embedded index.html references missing asset %q: %v", assetPath, err)
		}
	}
}

func embeddedAdminUITestFS(t *testing.T) fs.FS {
	t.Helper()

	subFS, err := fs.Sub(adminUI, "admin-ui/build")
	if err != nil {
		t.Fatalf("failed to create embedded admin UI sub filesystem: %v", err)
	}
	return subFS
}
