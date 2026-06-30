package agent

import (
	"testing"
)

func TestFilterToolsByPlanStrict(t *testing.T) {
	tools := []ToolDef{
		{Name: "tool_a", Description: "Tool A"},
		{Name: "tool_b", Description: "Tool B"},
		{Name: "tool_c", Description: "Tool C"},
	}
	filtered := filterToolsByPlan(tools, []string{"tool_a", "tool_b"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(filtered))
	}
	if filtered[0].Name != "tool_a" || filtered[1].Name != "tool_b" {
		t.Fatalf("unexpected tool order: %v", filtered)
	}
}

func TestFilterToolsByPlanWildcard(t *testing.T) {
	tools := []ToolDef{
		{Name: "tool_a", Description: "Tool A"},
		{Name: "tool_b", Description: "Tool B"},
	}
	filtered := filterToolsByPlan(tools, []string{"__all__"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 tools with wildcard, got %d", len(filtered))
	}
}

func TestFilterToolsByPlanNonexistent(t *testing.T) {
	tools := []ToolDef{
		{Name: "tool_a", Description: "Tool A"},
	}
	filtered := filterToolsByPlan(tools, []string{"tool_nonexistent"})
	if len(filtered) != 1 {
		t.Fatalf("expected fallback to all tools, got %d", len(filtered))
	}
}
