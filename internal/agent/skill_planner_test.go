package agent

import (
	"testing"

	"github.com/alaikis/opentether/internal/models"
)

func TestSkillPlanner_PlanSkills(t *testing.T) {
	planner := NewSkillPlanner(nil, nil)
	plan := &MultiTaskPlan{
		Original:   "查询北京和上海的销售额",
		SubTasks:   []SubTask{{Query: "北京销售额", SkillUsed: ""}, {Query: "上海销售额", SkillUsed: ""}},
		TotalSteps: 2,
	}
	planner.PlanSkills(plan)
	for _, sub := range plan.SubTasks {
		if sub.SkillUsed != "" {
			t.Errorf("expected no skill assignment without skills manager, got: %s", sub.SkillUsed)
		}
	}
}

func TestSkillPlanner_MatchSkill(t *testing.T) {
	planner := NewSkillPlanner(nil, nil)
	testSkills := []models.Skill{
		{SkillType: "text2sql", Name: "Text2SQL", Description: "查询数据库", Keywords: `["查询","sql"]`},
		{SkillType: "report", Name: "Report", Description: "生成报表", Keywords: `["报表","报告"]`},
	}
	matched := planner.matchSkill("查询", testSkills)
	if matched == nil || matched.SkillType != "text2sql" {
		t.Errorf("expected text2sql skill, got: %v", matched)
	}
	matched = planner.matchSkill("报告", testSkills)
	if matched == nil || matched.SkillType != "report" {
		t.Errorf("expected report skill, got: %v", matched)
	}
}

func TestSkillPlanner_ComputeSimilarity(t *testing.T) {
	planner := NewSkillPlanner(nil, nil)
	score := planner.computeSimilarity("查询", models.Skill{Name: "Text2SQL", Description: "数据库查询", Keywords: `["查询","sql"]`})
	if score <= 0 {
		t.Errorf("expected positive similarity score, got: %f", score)
	}
}

func TestSkillPlanner_SkillAlreadyAssigned(t *testing.T) {
	planner := NewSkillPlanner(nil, nil)
	plan := &MultiTaskPlan{
		SubTasks: []SubTask{{Query: "查询销售额", SkillUsed: "text2sql"}},
	}
	planner.PlanSkills(plan)
	if plan.SubTasks[0].SkillUsed != "text2sql" {
		t.Errorf("expected skill to remain text2sql, got: %s", plan.SubTasks[0].SkillUsed)
	}
}
