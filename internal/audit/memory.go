package audit

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type InMemoryAuditLogStore struct {
	mu       sync.RWMutex
	entries  []*AuditEntry
	dataDir  string
}

func NewInMemoryAuditLogStore(dataDir string) *InMemoryAuditLogStore {
	return &InMemoryAuditLogStore{
		entries: []*AuditEntry{},
		dataDir: dataDir,
	}
}

func (s *InMemoryAuditLogStore) Append(ctx context.Context, entry *AuditEntry) error {
	if entry == nil || entry.Operation == "" {
		return errors.New("invalid audit entry")
	}
	entry.ID = generateAuditID()
	entry.CreatedAt = time.Now()
	if entry.Status == "" {
		entry.Status = "success"
	}
	s.mu.Lock()
	s.entries = append(s.entries, entry)
	s.mu.Unlock()
	return s.persist()
}

func (s *InMemoryAuditLogStore) Query(ctx context.Context, filters map[string]interface{}, start, end time.Time) ([]*AuditEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*AuditEntry
	for _, e := range s.entries {
		if e.CreatedAt.Before(start) || e.CreatedAt.After(end) {
			continue
		}
		if actorID, ok := filters["actor_id"].(string); ok && actorID != "" && e.ActorID != actorID {
			continue
		}
		if op, ok := filters["operation"].(string); ok && op != "" && string(e.Operation) != op {
			continue
		}
		if resource, ok := filters["resource"].(string); ok && resource != "" && e.Resource != resource {
			continue
		}
		result = append(result, e)
	}
	return result, nil
}

func (s *InMemoryAuditLogStore) Count(ctx context.Context, filters map[string]interface{}, start, end time.Time) (int64, error) {
	entries, err := s.Query(ctx, filters, start, end)
	if err != nil {
		return 0, err
	}
	return int64(len(entries)), nil
}

func (s *InMemoryAuditLogStore) persist() error {
	if s.dataDir == "" {
		return nil
	}
	_ = os.MkdirAll(s.dataDir, 0755)
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, _ := json.Marshal(s.entries)
	return os.WriteFile(filepath.Join(s.dataDir, "audit_log.json"), data, 0644)
}

type FileComplianceReporter struct {
	dataDir string
}

func NewFileComplianceReporter(dataDir string) *FileComplianceReporter {
	return &FileComplianceReporter{dataDir: dataDir}
}

func (r *FileComplianceReporter) GenerateReport(ctx context.Context, start, end time.Time, format ExportFormat) (*ExportJob, error) {
	job := &ExportJob{
		ID:        generateExportID(),
		Format:    format,
		Filters:   map[string]interface{}{"start": start, "end": end},
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	if r.dataDir != "" {
		_ = os.MkdirAll(r.dataDir, 0755)
		path := filepath.Join(r.dataDir, job.ID+"."+string(format))
		job.FilePath = path
		_ = os.WriteFile(path, []byte("compliance report placeholder\n"), 0644)
		job.Status = "completed"
	}
	return job, nil
}

func (r *FileComplianceReporter) GetReport(jobID string) (*ExportJob, error) {
	return &ExportJob{ID: jobID, Status: "completed"}, nil
}

func (r *FileComplianceReporter) ListReports(start, end time.Time) ([]*ExportJob, error) {
	return []*ExportJob{}, nil
}

func generateAuditID() string {
	return "audit_" + time.Now().Format("20060102_150405_999999")
}

func generateExportID() string {
	return "export_" + time.Now().Format("20060102_150405")
}
