package service

import (
	"context"
	"testing"

	"github.com/alaikis/opentether/internal/config"
)

func TestDeriveReadinessStatus(t *testing.T) {
	tests := []struct {
		name   string
		checks []ReadinessCheck
		want   ReadinessStatus
	}{
		{
			name: "ready",
			checks: []ReadinessCheck{
				{Name: "a", Severity: ReadinessOK},
			},
			want: ReadinessReady,
		},
		{
			name: "ready with warnings",
			checks: []ReadinessCheck{
				{Name: "a", Severity: ReadinessOK},
				{Name: "b", Severity: ReadinessWarning},
			},
			want: ReadinessReadyWithWarnings,
		},
		{
			name: "not ready",
			checks: []ReadinessCheck{
				{Name: "a", Severity: ReadinessWarning},
				{Name: "b", Severity: ReadinessCritical},
			},
			want: ReadinessNotReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeriveReadinessStatus(tt.checks); got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestReadinessSecurityConfigChecks(t *testing.T) {
	svc := NewReadinessService(nil, &config.Config{
		Security: config.SecurityConfig{
			JWT:        config.JWTConfig{Secret: "${JWT_SECRET}"},
			Encryption: config.EncryptionConfig{Key: ""},
			CORS:       config.CORSConfig{AllowedOrigins: []string{"*"}},
			RateLimit:  config.RateLimitConfig{Enabled: false},
		},
	}, nil)

	report := svc.Report(context.Background())
	checks := map[string]ReadinessCheck{}
	for _, check := range report.Checks {
		checks[check.Name] = check
	}

	if checks["jwt_secret"].Severity != ReadinessCritical {
		t.Fatalf("expected critical jwt_secret check, got %s", checks["jwt_secret"].Severity)
	}
	if checks["encryption_key"].Severity != ReadinessWarning {
		t.Fatalf("expected warning encryption_key check, got %s", checks["encryption_key"].Severity)
	}
	if checks["cors"].Severity != ReadinessWarning {
		t.Fatalf("expected warning cors check, got %s", checks["cors"].Severity)
	}
	if checks["rate_limit"].Severity != ReadinessWarning {
		t.Fatalf("expected warning rate_limit check, got %s", checks["rate_limit"].Severity)
	}
}
