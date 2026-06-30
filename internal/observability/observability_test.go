package observability

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestInMemoryMetricCollector(t *testing.T) {
	collector := NewInMemoryMetricCollector()

	def := &MetricDefinition{
		ID:   "cpu_usage",
		Name: "CPU Usage",
		Type: MetricTypeGauge,
	}
	if err := collector.Register(def); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if err := collector.Record(context.Background(), "cpu_usage", 75.5, nil); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	start := time.Now().Add(-time.Hour)
	end := time.Now().Add(time.Hour)
	results, err := collector.Query("cpu_usage", start, end)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 1 || results[0].Value != 75.5 {
		t.Fatalf("Query returned unexpected results: %v", results)
	}

	metrics, err := collector.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(metrics) != 1 || metrics[0].ID != "cpu_usage" {
		t.Fatalf("List returned unexpected results: %v", metrics)
	}

	if err := collector.Record(context.Background(), "unknown_metric", 10.0, nil); err == nil {
		t.Fatal("Expected error recording to unknown metric")
	}
}

func TestInMemoryHookRegistry(t *testing.T) {
	registry := NewInMemoryHookRegistry()

	var called bool
	fn := func(ctx context.Context, name string, value float64, labels map[string]string) {
		called = true
	}

	if err := registry.Register("test_hook", fn); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	registry.Emit(context.Background(), "test_hook", 42.0, nil)
	time.Sleep(50 * time.Millisecond)
	if !called {
		t.Fatal("Hook was not called")
	}

	if err := registry.Unregister("test_hook"); err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	called = false
	registry.Emit(context.Background(), "test_hook", 42.0, nil)
	time.Sleep(50 * time.Millisecond)
	if called {
		t.Fatal("Hook was called after unregister")
	}
}

func TestFileBackedAlertRuleEngineCRUD(t *testing.T) {
	dataDir := t.TempDir()
	engine := NewFileBackedAlertRuleEngine(dataDir)
	engine.SetCollector(NewInMemoryMetricCollector())

	rule := &AlertRule{
		ID:        "rule_1",
		Name:      "High CPU",
		MetricID:  "cpu",
		Condition: "gt",
		Threshold: 90,
		Window:    "1m",
		Enabled:   true,
	}
	if err := engine.CreateRule(rule); err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	rules, err := engine.ListRules()
	if err != nil {
		t.Fatalf("ListRules failed: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(rules))
	}

	updated := &AlertRule{
		Name:      "Updated CPU",
		MetricID:  "cpu",
		Condition: "gt",
		Threshold: 95,
		Window:    "1m",
		Enabled:   true,
	}
	if err := engine.UpdateRule("rule_1", updated); err != nil {
		t.Fatalf("UpdateRule failed: %v", err)
	}

	rules, err = engine.ListRules()
	if err != nil {
		t.Fatalf("ListRules failed: %v", err)
	}
	if rules[0].Name != "Updated CPU" || rules[0].Threshold != 95 {
		t.Fatalf("Rule not updated: %v", rules[0])
	}

	if err := engine.DeleteRule("rule_1"); err != nil {
		t.Fatalf("DeleteRule failed: %v", err)
	}
	rules, err = engine.ListRules()
	if err != nil {
		t.Fatalf("ListRules failed: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("Expected 0 rules after delete, got %d", len(rules))
	}

	if err := engine.UpdateRule("nonexistent", updated); err == nil {
		t.Fatal("Expected error updating nonexistent rule")
	}
	if err := engine.DeleteRule("nonexistent"); err != nil {
		t.Fatalf("DeleteRule should not error for nonexistent rule: %v", err)
	}
}

func TestFileBackedAlertRuleEngineEvaluate(t *testing.T) {
	dataDir := t.TempDir()
	collector := NewInMemoryMetricCollector()
	engine := NewFileBackedAlertRuleEngine(dataDir)
	engine.SetCollector(collector)

	metricDef := &MetricDefinition{
		ID:   "cpu",
		Name: "CPU",
		Type: MetricTypeGauge,
	}
	if err := collector.Register(metricDef); err != nil {
		t.Fatalf("Register metric failed: %v", err)
	}

	rule := &AlertRule{
		ID:        "cpu_high",
		Name:      "High CPU",
		MetricID:  "cpu",
		Condition: "gt",
		Threshold: 80,
		Window:    "5m",
		Enabled:   true,
	}
	if err := engine.CreateRule(rule); err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	now := time.Now()
	if err := collector.Record(context.Background(), "cpu", 85.0, nil); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	alerts, err := engine.Evaluate(now)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("Expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].RuleID != "cpu_high" {
		t.Fatalf("Expected alert for rule cpu_high, got %s", alerts[0].RuleID)
	}

	if err := collector.Record(context.Background(), "cpu", 70.0, nil); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	alerts, err = engine.Evaluate(now)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("Expected 1 alert, got %d", len(alerts))
	}

	disabledRule := &AlertRule{
		ID:        "cpu_low",
		Name:      "Low CPU",
		MetricID:  "cpu",
		Condition: "lt",
		Threshold: 10,
		Window:    "5m",
		Enabled:   false,
	}
	if err := engine.CreateRule(disabledRule); err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	alerts, err = engine.Evaluate(now)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	for _, a := range alerts {
		if a.RuleID == "cpu_low" {
			t.Fatal("Disabled rule should not fire")
		}
	}
}

func TestFileBackedAlertRuleEngineAckAlert(t *testing.T) {
	dataDir := t.TempDir()
	engine := NewFileBackedAlertRuleEngine(dataDir)
	engine.SetCollector(NewInMemoryMetricCollector())

	metricDef := &MetricDefinition{
		ID:   "cpu",
		Name: "CPU",
		Type: MetricTypeGauge,
	}
	collector := NewInMemoryMetricCollector()
	collector.Register(metricDef)
	engine.SetCollector(collector)

	rule := &AlertRule{
		ID:        "cpu_high",
		Name:      "High CPU",
		MetricID:  "cpu",
		Condition: "gt",
		Threshold: 80,
		Window:    "5m",
		Enabled:   true,
	}
	engine.CreateRule(rule)
	collector.Record(context.Background(), "cpu", 85.0, nil)

	now := time.Now()
	alerts, _ := engine.Evaluate(now)
	if len(alerts) != 1 {
		t.Fatalf("Expected 1 alert, got %d", len(alerts))
	}

	if err := engine.AckAlert(alerts[0].ID); err != nil {
		t.Fatalf("AckAlert failed: %v", err)
	}
	if !alerts[0].Resolved {
		t.Fatal("Alert should be resolved")
	}
	if alerts[0].ResolvedAt == nil {
		t.Fatal("ResolvedAt should be set")
	}

	if err := engine.AckAlert("nonexistent"); err == nil {
		t.Fatal("Expected error acking nonexistent alert")
	}
}

func TestFileBackedAlertRuleEnginePersistence(t *testing.T) {
	dataDir := t.TempDir()

	engine := NewFileBackedAlertRuleEngine(dataDir)
	engine.SetCollector(NewInMemoryMetricCollector())

	rule := &AlertRule{
		ID:        "rule_1",
		Name:      "Persist Rule",
		MetricID:  "cpu",
		Condition: "gt",
		Threshold: 90,
		Window:    "1m",
		Enabled:   true,
	}
	if err := engine.CreateRule(rule); err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	rulesPath := dataDir + "/alert_rules.json"
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		t.Fatal("alert_rules.json should exist")
	}

	data, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("Failed to read rules file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Rules file should not be empty")
	}
}
