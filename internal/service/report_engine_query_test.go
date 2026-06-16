package service

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newReportQueryTestService(t *testing.T) *ReportEngineService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec("CREATE TABLE sales (employee TEXT, orders INTEGER, amount REAL)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := db.Exec("INSERT INTO sales (employee, orders, amount) VALUES (?, ?, ?), (?, ?, ?)", "林烽", 3, 1200.5, "王五", 2, 800.0).Error; err != nil {
		t.Fatalf("insert sales: %v", err)
	}
	return NewReportEngineService(db, nil)
}

func TestResolveTableDataExecutesQuery(t *testing.T) {
	svc := newReportQueryTestService(t)
	headers, rows, err := svc.resolveTableData(context.Background(), ReportTemplateSection{Query: "SELECT employee, orders FROM sales ORDER BY employee"}, nil, "")
	if err != nil {
		t.Fatalf("resolve table data: %v", err)
	}
	if len(headers) != 2 || headers[0] != "employee" || headers[1] != "orders" {
		t.Fatalf("unexpected headers: %#v", headers)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
}

func TestResolveChartDataExecutesQuery(t *testing.T) {
	svc := newReportQueryTestService(t)
	labels, values, err := svc.resolveChartData(context.Background(), ReportTemplateSection{Query: "SELECT employee, amount FROM sales ORDER BY employee", LabelColumn: "employee", ValueColumn: "amount"}, nil, "")
	if err != nil {
		t.Fatalf("resolve chart data: %v", err)
	}
	if len(labels) != 2 || len(values) != 2 {
		t.Fatalf("unexpected chart data: labels=%#v values=%#v", labels, values)
	}
	if values[0] <= 0 || values[1] <= 0 {
		t.Fatalf("expected positive chart values: %#v", values)
	}
}

func TestReportQueryRejectsWrites(t *testing.T) {
	if err := validateReportReadOnlyQuery("DELETE FROM sales"); err == nil {
		t.Fatal("expected write query to be rejected")
	}
}
