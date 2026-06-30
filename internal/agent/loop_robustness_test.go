package agent

import (
	"testing"
)

func TestIsCompleteToolResult(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{"empty", "", false},
		{"truncated", "查询结果被截断...", false},
		{"failed", "工具执行失败: timeout", false},
		{"legacy markers", "[SQL] SELECT * FROM orders\n[列] id, name\n[数据] 1, 2\n[统计] 2 rows", true},
		{"json structured", `{"columns":["id","name"],"rows":[{"id":1,"name":"A"}],"row_count":1}`, true},
		{"json incomplete", `{"columns":["id"]}`, true},
		{"short complete", "查询成功，共返回 100 条数据，详见下表...", false},
		{"random text", "some random output without markers", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCompleteToolResult(tt.output)
			if result != tt.expected {
				t.Fatalf("isCompleteToolResult(%q) = %v, want %v", tt.output, result, tt.expected)
			}
		})
	}
}

func TestSummarizeCompleteParallelResults(t *testing.T) {
	steps := []LoopStep{
		{ToolOutput: `{"columns":["id"],"rows":[{"id":1}],"row_count":1}`, Error: ""},
		{ToolOutput: "工具执行失败: timeout", Error: "timeout"},
	}
	summary, ok := summarizeCompleteParallelResults(steps)
	if !ok {
		t.Fatal("expected true for mixed results")
	}
	if !Contains(summary, "1个任务执行失败") {
		t.Fatalf("expected failure notice in summary, got: %s", summary)
	}
}

func Contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && ContainsHelper(s, substr))
}

func ContainsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
