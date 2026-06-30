package service

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/alaikis/opentether/internal/audit"
	"github.com/alaikis/opentether/internal/storage"
)

type AuditLogService struct {
	store    audit.AuditLogStore
	reporter audit.ComplianceReporter
	fileStore storage.Driver
}

func NewAuditLogService(db interface{}, fileStore storage.Driver) *AuditLogService {
	dataDir := "data/audit"
	return &AuditLogService{
		store:     audit.NewInMemoryAuditLogStore(dataDir),
		reporter:  audit.NewFileComplianceReporter(dataDir),
		fileStore: fileStore,
	}
}

func (s *AuditLogService) Append(entry *audit.AuditEntry) error {
	return s.store.Append(nil, entry)
}

func (s *AuditLogService) Query(filters map[string]interface{}, start, end time.Time) ([]*audit.AuditEntry, error) {
	return s.store.Query(nil, filters, start, end)
}

func (s *AuditLogService) Count(filters map[string]interface{}, start, end time.Time) (int64, error) {
	return s.store.Count(nil, filters, start, end)
}

func (s *AuditLogService) Audit(userID, userName, action, resourceType, resourceID, details, ipAddress, userAgent string) {
	entry := &audit.AuditEntry{
		ID:        "",
		Operation: audit.OperationType(action),
		ActorID:   userID,
		ActorType: "user",
		Resource:  resourceType,
		Action:    action,
		Details: map[string]interface{}{
			"resource_id": resourceID,
			"details":     details,
		},
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Status:    "success",
	}
	_ = s.Append(entry)
}

func (s *AuditLogService) ExportLogs(format audit.ExportFormat) (*audit.ExportJob, error) {
	return s.reporter.GenerateReport(nil, time.Now().Add(-24*time.Hour), time.Now(), format)
}

func (s *AuditLogService) GenerateComplianceReport(format audit.ExportFormat) (*audit.ExportJob, error) {
	return s.reporter.GenerateReport(nil, time.Now().Add(-24*time.Hour), time.Now(), format)
}

func (s *AuditLogService) ListReports() ([]*audit.ExportJob, error) {
	return s.reporter.ListReports(time.Now().Add(-24*time.Hour), time.Now())
}

func (s *AuditLogService) GetReport(jobID string) (*audit.ExportJob, error) {
	return s.reporter.GetReport(jobID)
}

func (s *AuditLogService) QueryForensic(filters map[string]interface{}, start, end time.Time) ([]*audit.AuditEntry, error) {
	return s.Query(filters, start, end)
}

func (s *AuditLogService) ExportToS3(jobID, bucket, key string) error {
	if jobID == "" || bucket == "" {
		return errors.New("invalid s3 export parameters")
	}
	if s.fileStore == nil {
		return errors.New("storage driver not configured")
	}
	entries, err := s.store.Query(nil, map[string]interface{}{}, time.Now().Add(-30*24*time.Hour), time.Now())
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	objectKey := "audit-exports/" + key
	_, err = s.fileStore.Save(nil, objectKey, data, "application/json")
	return err
}
