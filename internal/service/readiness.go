package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/models"
	"github.com/alaikis/opentether/internal/storage"
	"gorm.io/gorm"
)

type ReadinessStatus string

type ReadinessSeverity string

const (
	ReadinessReady             ReadinessStatus   = "ready"
	ReadinessReadyWithWarnings ReadinessStatus   = "ready_with_warnings"
	ReadinessNotReady          ReadinessStatus   = "not_ready"
	ReadinessOK                ReadinessSeverity = "ok"
	ReadinessWarning           ReadinessSeverity = "warning"
	ReadinessCritical          ReadinessSeverity = "critical"
)

type ReadinessCheck struct {
	Name     string            `json:"name"`
	Severity ReadinessSeverity `json:"severity"`
	Message  string            `json:"message"`
	Details  map[string]string `json:"details,omitempty"`
}

type ReadinessReport struct {
	Status ReadinessStatus  `json:"status"`
	Checks []ReadinessCheck `json:"checks"`
}

type ReadinessService struct {
	db    *gorm.DB
	cfg   *config.Config
	store storage.Driver
}

func NewReadinessService(db *gorm.DB, cfg *config.Config, store storage.Driver) *ReadinessService {
	return &ReadinessService{db: db, cfg: cfg, store: store}
}

func (s *ReadinessService) Report(ctx context.Context) ReadinessReport {
	checks := []ReadinessCheck{
		s.checkDatabase(ctx),
		s.checkStorage(),
		s.checkJWTSecret(),
		s.checkEncryptionKey(),
		s.checkCORS(),
		s.checkRateLimit(),
		s.checkSandbox(),
		s.checkProviders(),
		s.checkAdminUI(),
	}
	return ReadinessReport{Status: DeriveReadinessStatus(checks), Checks: checks}
}

func DeriveReadinessStatus(checks []ReadinessCheck) ReadinessStatus {
	hasWarning := false
	for _, check := range checks {
		switch check.Severity {
		case ReadinessCritical:
			return ReadinessNotReady
		case ReadinessWarning:
			hasWarning = true
		}
	}
	if hasWarning {
		return ReadinessReadyWithWarnings
	}
	return ReadinessReady
}

func (s *ReadinessService) checkDatabase(ctx context.Context) ReadinessCheck {
	if s.db == nil {
		return readinessCheck("database", ReadinessCritical, "database is not initialized", nil)
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return readinessCheck("database", ReadinessCritical, err.Error(), nil)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return readinessCheck("database", ReadinessCritical, err.Error(), nil)
	}
	return readinessCheck("database", ReadinessOK, "database connection is available", map[string]string{"type": s.cfg.Database.Type})
}

func (s *ReadinessService) checkStorage() ReadinessCheck {
	if s.store == nil {
		return readinessCheck("storage", ReadinessCritical, "storage driver is not initialized", nil)
	}
	return readinessCheck("storage", ReadinessOK, "storage driver is initialized", map[string]string{"type": s.cfg.Storage.Type})
}

func (s *ReadinessService) checkJWTSecret() ReadinessCheck {
	secret := strings.TrimSpace(s.cfg.Security.JWT.Secret)
	if secret == "" || hasPlaceholder(secret) {
		return readinessCheck("jwt_secret", ReadinessCritical, "jwt secret is empty or unresolved", nil)
	}
	if len(secret) < 32 {
		return readinessCheck("jwt_secret", ReadinessWarning, "jwt secret is shorter than 32 characters", nil)
	}
	return readinessCheck("jwt_secret", ReadinessOK, "jwt secret is configured", nil)
}

func (s *ReadinessService) checkEncryptionKey() ReadinessCheck {
	key := strings.TrimSpace(s.cfg.Security.Encryption.Key)
	if key == "" || hasPlaceholder(key) {
		return readinessCheck("encryption_key", ReadinessWarning, "encryption key is empty or unresolved", nil)
	}
	if len(key) < 32 {
		return readinessCheck("encryption_key", ReadinessWarning, "encryption key is shorter than 32 characters", nil)
	}
	return readinessCheck("encryption_key", ReadinessOK, "encryption key is configured", nil)
}

func (s *ReadinessService) checkCORS() ReadinessCheck {
	for _, origin := range s.cfg.Security.CORS.AllowedOrigins {
		if strings.TrimSpace(origin) == "*" {
			return readinessCheck("cors", ReadinessWarning, "wildcard CORS origin is enabled", nil)
		}
	}
	return readinessCheck("cors", ReadinessOK, "CORS origins are restricted", nil)
}

func (s *ReadinessService) checkRateLimit() ReadinessCheck {
	if !s.cfg.Security.RateLimit.Enabled {
		return readinessCheck("rate_limit", ReadinessWarning, "request rate limiting is disabled", nil)
	}
	if s.cfg.Security.RateLimit.RequestsPerMinute <= 0 {
		return readinessCheck("rate_limit", ReadinessCritical, "requests_per_minute must be greater than zero", nil)
	}
	return readinessCheck("rate_limit", ReadinessOK, "request rate limiting is enabled", map[string]string{"requests_per_minute": stringInt(s.cfg.Security.RateLimit.RequestsPerMinute)})
}

func (s *ReadinessService) checkSandbox() ReadinessCheck {
	if !s.cfg.Executor.EmbeddedConfig.Sandbox.Enabled {
		return readinessCheck("sandbox", ReadinessWarning, "script sandbox is disabled", nil)
	}
	return readinessCheck("sandbox", ReadinessOK, "script sandbox is enabled", nil)
}

func (s *ReadinessService) checkProviders() ReadinessCheck {
	if s.db == nil {
		return readinessCheck("providers", ReadinessWarning, "provider configuration cannot be checked without database", nil)
	}
	var count int64
	if err := s.db.Model(&models.Provider{}).Where("enabled = ?", true).Count(&count).Error; err != nil {
		return readinessCheck("providers", ReadinessWarning, err.Error(), nil)
	}
	if count == 0 {
		return readinessCheck("providers", ReadinessWarning, "no enabled LLM provider configured", nil)
	}
	return readinessCheck("providers", ReadinessOK, "enabled LLM provider exists", map[string]string{"enabled_count": stringInt(int(count))})
}

func (s *ReadinessService) checkAdminUI() ReadinessCheck {
	return readinessCheck("admin_ui", ReadinessOK, "admin UI route is configured", nil)
}

func readinessCheck(name string, severity ReadinessSeverity, message string, details map[string]string) ReadinessCheck {
	return ReadinessCheck{Name: name, Severity: severity, Message: message, Details: details}
}

func hasPlaceholder(value string) bool {
	return strings.Contains(value, "${") || strings.Contains(value, "your-secret")
}

func stringInt(value int) string {
	return strconv.Itoa(value)
}
