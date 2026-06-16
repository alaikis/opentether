package agent

import (
	"strings"
	"testing"
	"time"
)

func TestRenderSQLTemplateUsesRuntimeTimeRange(t *testing.T) {
	tpl := "SELECT * FROM orders WHERE delivery_time >= {{start_date}} AND delivery_time < {{end_date}}"
	sql, ok := renderSQLTemplate(tpl, "林烽 1至6月销售额", time.Date(2026, 6, 16, 0, 0, 0, 0, time.Local))
	if !ok {
		t.Fatal("expected template render to succeed")
	}
	if !strings.Contains(sql, "'2026-01-01'") || !strings.Contains(sql, "'2026-07-01'") {
		t.Fatalf("unexpected rendered sql: %s", sql)
	}
	if strings.Contains(sql, "{{") {
		t.Fatalf("unresolved variable: %s", sql)
	}
}

func TestRenderSQLTemplateRejectsUnresolvedVariables(t *testing.T) {
	_, ok := renderSQLTemplate("SELECT * FROM t WHERE x={{unknown}}", "今年", time.Now())
	if ok {
		t.Fatal("expected unresolved template to be rejected")
	}
}
