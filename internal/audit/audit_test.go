package audit

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestInMemoryAuditLogStore(t *testing.T) {
	dataDir := t.TempDir()
	store := NewInMemoryAuditLogStore(dataDir)

	entry := &AuditEntry{
		Operation:   OperationDataAccess,
		ActorID:     "user_1",
		ActorType:   "user",
		Resource:    "document_1",
		Action:      "read",
		IPAddress:   "192.168.1.1",
		UserAgent:   "test-agent",
		Status:      "success",
	}
	if err := store.Append(context.Background(), entry); err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if entry.ID == "" {
		t.Fatal("Append should set entry ID")
	}

	start := time.Now().Add(-time.Hour)
	end := time.Now().Add(time.Hour)
	results, err := store.Query(context.Background(), nil, start, end)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(results))
	}

	filtered, err := store.Query(context.Background(), map[string]interface{}{"actor_id": "user_1"}, start, end)
	if err != nil {
		t.Fatalf("Query with filter failed: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("Expected 1 filtered entry, got %d", len(filtered))
	}

	filtered2, err := store.Query(context.Background(), map[string]interface{}{"actor_id": "user_2"}, start, end)
	if err != nil {
		t.Fatalf("Query with filter failed: %v", err)
	}
	if len(filtered2) != 0 {
		t.Fatalf("Expected 0 entries for user_2, got %d", len(filtered2))
	}

	count, err := store.Count(context.Background(), nil, start, end)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("Expected count 1, got %d", count)
	}

	filteredCount, err := store.Count(context.Background(), map[string]interface{}{"resource": "document_1"}, start, end)
	if err != nil {
		t.Fatalf("Count with filter failed: %v", err)
	}
	if filteredCount != 1 {
		t.Fatalf("Expected filtered count 1, got %d", filteredCount)
	}

	_, err = store.Query(context.Background(), map[string]interface{}{"operation": string(OperationAuth)}, start, end)
	if err != nil {
		t.Fatalf("Query with operation filter failed: %v", err)
	}

	_, err = store.Query(context.Background(), map[string]interface{}{"resource": "document_1"}, start, end)
	if err != nil {
		t.Fatalf("Query with resource filter failed: %v", err)
	}
}

func TestInMemoryAuditLogStorePersistence(t *testing.T) {
	dataDir := t.TempDir()
	store := NewInMemoryAuditLogStore(dataDir)

	entry := &AuditEntry{
		Operation: OperationSystem,
		ActorID:   "system",
	}
	store.Append(context.Background(), entry)

	logPath := dataDir + "/audit_log.json"
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatal("audit_log.json should exist")
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Log file should not be empty")
	}
}

func TestFileComplianceReporter(t *testing.T) {
	dataDir := t.TempDir()
	reporter := NewFileComplianceReporter(dataDir)

	start := time.Now().Add(-time.Hour)
	end := time.Now().Add(time.Hour)
	job, err := reporter.GenerateReport(context.Background(), start, end, ExportFormatJSON)
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}
	if job.Status != "completed" {
		t.Fatalf("Expected completed status, got %s", job.Status)
	}
	if job.FilePath == "" {
		t.Fatal("Expected file path to be set")
	}

	if _, err := os.Stat(job.FilePath); os.IsNotExist(err) {
		t.Fatal("Report file should exist")
	}

	reports, err := reporter.ListReports(start, end)
	if err != nil {
		t.Fatalf("ListReports failed: %v", err)
	}
	if len(reports) != 0 {
		t.Fatalf("Expected 0 reports from ListReports, got %d", len(reports))
	}

	retrieved, err := reporter.GetReport(job.ID)
	if err != nil {
		t.Fatalf("GetReport failed: %v", err)
	}
	if retrieved.ID != job.ID {
		t.Fatalf("Expected report ID %s, got %s", job.ID, retrieved.ID)
	}
}

func TestAuditEntryImmutability(t *testing.T) {
	store := NewInMemoryAuditLogStore("")

	entry := &AuditEntry{
		Operation: OperationAuth,
		ActorID:   "user_1",
		Resource:  "auth",
		Action:    "login",
	}
	if err := store.Append(context.Background(), entry); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	start := time.Now().Add(-time.Hour)
	end := time.Now().Add(time.Hour)
	results, err := store.Query(context.Background(), nil, start, end)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(results))
	}

	queried := results[0]
	if queried.Operation != OperationAuth {
		t.Fatalf("Expected operation auth, got %s", queried.Operation)
	}
	if queried.ActorID != "user_1" {
		t.Fatalf("Expected actor_id user_1, got %s", queried.ActorID)
	}

	entry2 := &AuditEntry{
		Operation: OperationAuth,
		ActorID:   "user_1",
		Resource:  "auth",
		Action:    "logout",
	}
	if err := store.Append(context.Background(), entry2); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	results, err = store.Query(context.Background(), nil, start, end)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(results))
	}

	actions := map[string]bool{}
	for _, r := range results {
		actions[r.Action] = true
	}
	if !actions["login"] {
		t.Fatal("First entry should remain unchanged")
	}
	if !actions["logout"] {
		t.Fatal("Second entry should be present")
	}
}
