package agent

import (
	"strings"
	"testing"
)

func TestFallbackDecisionSanitizesToolCallLeak(t *testing.T) {
	text := `我可以生成图片。
{"action":"tool_call","tool_name":"skill__abc","tool_input":{"language":"python","script_content":"import matplotlib`
	decision := fallbackDecisionFromText(text)
	if decision.Action != "final_answer" {
		t.Fatalf("expected final_answer, got %s", decision.Action)
	}
	if decision.FinalAnswer == text {
		t.Fatal("raw tool call leak was returned")
	}
	if strings.Contains(decision.FinalAnswer, "script_content") || strings.Contains(decision.FinalAnswer, "tool_name") {
		t.Fatalf("fallback leaked tool fields: %s", decision.FinalAnswer)
	}
}
