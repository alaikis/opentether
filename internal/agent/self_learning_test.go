package agent

import (
	"testing"
)

func TestSelfLearningBacktestNil(t *testing.T) {
	var sl *SelfLearning
	result := sl.Backtest("test")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.TotalPatterns != 0 {
		t.Fatalf("expected 0 patterns for nil self-learning, got %d", result.TotalPatterns)
	}
}

func TestNormalizeForLearning(t *testing.T) {
	query := "查询北京销售额？"
	normalized := normalizeForLearning(query)
	if normalized == "" {
		t.Fatal("expected non-empty normalized query")
	}
}
