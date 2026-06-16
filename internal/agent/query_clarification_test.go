package agent

import (
	"testing"

	"github.com/alaikis/opentether/internal/models"
)

func TestResolveQueryClarificationSkipsMultiPartQuestions(t *testing.T) {
	result := ResolveQueryClarification("林烽 3月份 出了多少单？累计卖了多少钱？卖到了哪些国家？每个国家多少单？", nil, "conv-1")
	if result.NeedsClarify {
		t.Fatal("multi-part query should not be clarified")
	}
	if result.Rewritten {
		t.Fatal("multi-part query should not be rewritten")
	}
}

func TestSplitMultiPartQuestions(t *testing.T) {
	parts := SplitMultiPartQuestions("出了多少单？卖到哪些国家？每个国家多少单？")
	if len(parts) < 3 {
		t.Fatalf("expected at least 3 parts, got %d: %#v", len(parts), parts)
	}
}

func TestResolveQueryClarificationSkipsWhenRecentSalesContext(t *testing.T) {
	recent := []models.Message{{Role: "user", Content: "查询林烽五月订单数"}}
	result := ResolveQueryClarification("卖多少钱？", recent, "conv-1")
	if result.NeedsClarify {
		t.Fatalf("expected no clarification with recent sales context, got: %s", result.Response.Message)
	}
}

func TestResolveQueryClarificationClarifiesVagueQueryWithoutContext(t *testing.T) {
	result := ResolveQueryClarification("卖多少钱？", nil, "conv-1")
	if !result.NeedsClarify {
		t.Fatal("expected clarification for vague query without context")
	}
}
