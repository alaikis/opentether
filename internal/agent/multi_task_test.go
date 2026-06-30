package agent

import (
	"testing"
)

func TestBuildMultiTaskPlan(t *testing.T) {
	plan := BuildMultiTaskPlan("查询北京销售额和上海销售额")
	if plan == nil {
		t.Fatal("expected multi-task plan")
	}
	if plan.TotalSteps != 2 {
		t.Fatalf("expected 2 steps, got %d", plan.TotalSteps)
	}
	if plan.IsTree {
		t.Fatal("expected flat plan, got tree")
	}
}

func TestBuildMultiTaskPlanSingle(t *testing.T) {
	plan := BuildMultiTaskPlan("查询北京销售额")
	if plan != nil {
		t.Fatalf("expected nil plan for single question, got %v", plan)
	}
}

func TestDetectTaskTree(t *testing.T) {
	tree := detectTaskTree("先查询订单，然后分析利润", []string{"查询订单", "分析利润"})
	if tree == nil {
		t.Fatal("expected tree detection for sequential query")
	}
	if len(tree) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tree))
	}
	if len(tree[1].Dependencies) != 1 || tree[1].Dependencies[0] != 0 {
		t.Fatalf("expected task 1 to depend on task 0")
	}
}

func TestDetectTaskTreeNoSequential(t *testing.T) {
	tree := detectTaskTree("查询北京和上海销售额", []string{"查询北京销售额", "查询上海销售额"})
	if tree != nil {
		t.Fatalf("expected nil tree for parallel query, got %v", tree)
	}
}

func TestExtractContextWords(t *testing.T) {
	words := extractContextWords("查询北京和上海的电子产品销售额")
	if len(words) == 0 {
		t.Fatal("expected context words")
	}
	found := false
	for _, w := range words {
		if w == "北京" || w == "上海" || w == "电子产品" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find context words, got %v", words)
	}
}

func TestExtractMetricFromQuery(t *testing.T) {
	metrics := extractMetricFromQuery("查询各部门的销售额趋势")
	found := false
	for _, m := range metrics {
		if m == "销售额" || m == "趋势" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected metric, got %v", metrics)
	}
}

func TestExtractMetricFromQueryNoMatch(t *testing.T) {
	metrics := extractMetricFromQuery("查询用户信息")
	if len(metrics) != 0 {
		t.Fatalf("expected empty metrics, got %v", metrics)
	}
}
