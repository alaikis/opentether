package audit

import (
	"context"
	"time"
)

type OperationType string

const (
	OperationAuth     OperationType = "auth"
	OperationDataAccess OperationType = "data_access"
	OperationConfigChange OperationType = "config_change"
	OperationToolCall  OperationType = "tool_call"
	OperationSystem    OperationType = "system"
)

type AuditEntry struct {
	ID          string            `json:"id"`
	Operation   OperationType     `json:"operation"`
	ActorID     string            `json:"actor_id"`
	ActorType   string            `json:"actor_type"`
	Resource    string            `json:"resource"`
	Action      string            `json:"action"`
	Details     map[string]interface{} `json:"details"`
	IPAddress   string            `json:"ip_address"`
	UserAgent   string            `json:"user_agent"`
	Status      string            `json:"status"`
	Error       string            `json:"error,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatPDF  ExportFormat = "pdf"
)

type ExportJob struct {
	ID        string        `json:"id"`
	Format    ExportFormat  `json:"format"`
	Filters   map[string]interface{} `json:"filters"`
	Status    string        `json:"status"`
	FilePath  string        `json:"file_path"`
	ExpiresAt time.Time     `json:"expires_at"`
	CreatedAt time.Time     `json:"created_at"`
}

type AuditLogStore interface {
	Append(ctx context.Context, entry *AuditEntry) error
	Query(ctx context.Context, filters map[string]interface{}, start, end time.Time) ([]*AuditEntry, error)
	Count(ctx context.Context, filters map[string]interface{}, start, end time.Time) (int64, error)
}

type ComplianceReporter interface {
	GenerateReport(ctx context.Context, start, end time.Time, format ExportFormat) (*ExportJob, error)
	GetReport(jobID string) (*ExportJob, error)
	ListReports(start, end time.Time) ([]*ExportJob, error)
}
