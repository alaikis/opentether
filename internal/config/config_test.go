package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFileExpandsEnvironmentVariables(t *testing.T) {
	t.Setenv("JWT_SECRET", "env-secret-value")

	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte("security:\n  jwt:\n    secret: \"${JWT_SECRET}\"\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Security.JWT.Secret != "env-secret-value" {
		t.Fatalf("expected expanded JWT secret, got %q", cfg.Security.JWT.Secret)
	}
}

func TestLoadFromFileMissingEnvironmentVariableDoesNotRetainPlaceholder(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte("security:\n  jwt:\n    secret: \"${JWT_SECRET}\"\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Security.JWT.Secret == "${JWT_SECRET}" {
		t.Fatalf("placeholder was retained")
	}
	if cfg.Security.JWT.Secret != "" {
		t.Fatalf("expected missing env to expand to empty string, got %q", cfg.Security.JWT.Secret)
	}
}
