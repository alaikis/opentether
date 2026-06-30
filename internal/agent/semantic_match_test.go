package agent

import (
	"testing"
)

func TestSemanticMatchTools(t *testing.T) {
	tools := []ToolDef{
		{Name: "text2sql", Description: "Query database using natural language"},
		{Name: "report", Description: "Generate reports and analytics"},
	}
	result := SemanticMatchTools("查询数据", tools, 2)
	if len(result) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(result))
	}
}

func TestSemanticMatchToolsEmpty(t *testing.T) {
	result := SemanticMatchTools("test", nil, 5)
	if result != nil {
		t.Fatalf("expected nil for empty tools, got %v", result)
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{0, 1, 0}
	score := cosineSimilarity(a, b)
	if score != 0 {
		t.Fatalf("expected 0 for orthogonal vectors, got %f", score)
	}

	c := []float64{1, 0, 0}
	d := []float64{1, 0, 0}
	score2 := cosineSimilarity(c, d)
	if score2 != 1.0 {
		t.Fatalf("expected 1.0 for identical vectors, got %f", score2)
	}
}
