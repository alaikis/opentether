package agent

import (
	"testing"
)

func TestRuleBasedPlan(t *testing.T) {
	e := &AgentEngine{db: nil}
	plan := e.ruleBasedPlan("查询北京销售额")
	if plan == nil {
		t.Fatal("expected plan")
	}
	if plan.Intent != "text2sql" {
		t.Fatalf("expected text2sql intent, got %s", plan.Intent)
	}
	if len(plan.Tools) == 0 {
		t.Fatal("expected at least one tool")
	}
}

func TestRuleBasedPlanChat(t *testing.T) {
	e := &AgentEngine{db: nil}
	plan := e.ruleBasedPlan("你好")
	if plan == nil {
		t.Fatal("expected plan")
	}
	if plan.Intent != "chat" {
		t.Fatalf("expected chat intent, got %s", plan.Intent)
	}
}

func TestRuleBasedPlanPDF(t *testing.T) {
	e := &AgentEngine{db: nil}
	plan := e.ruleBasedPlan("导出PDF报表")
	if plan == nil {
		t.Fatal("expected plan")
	}
	found := false
	for _, tool := range plan.Tools {
		if tool == "pdf" || tool == "report" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected pdf or report tool, got %v", plan.Tools)
	}
}
