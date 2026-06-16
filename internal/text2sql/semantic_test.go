package text2sql

import (
	"strings"
	"testing"
)

func TestBuildSecureSemanticContextResolvesSelfPolicy(t *testing.T) {
	cfg := `{
		"text2sql_mode":"query_plan",
		"semantic_model":{
			"entities":[{
				"name":"order","label":"订单","table":"t_order","alias":"o",
				"fields":[
					{"name":"amount","label":"销售额","column":"pay_amount","type":"number"},
					{"name":"owner_user_id","label":"归属员工","column":"company_user_id","type":"string"}
				]
			}]
		},
		"access_policies":[{
			"name":"本人订单","scope":"self","target_entity":"order","target_field":"owner_user_id","principal_field":"company_user_id","missing_principal":"deny"
		}]
	}`

	ctx, err := BuildSecureSemanticContext(cfg, map[string]interface{}{"company_user_id": "U1001"})
	if err != nil {
		t.Fatalf("BuildSecureSemanticContext returned error: %v", err)
	}
	if !ctx.Enabled {
		t.Fatal("expected secure context enabled")
	}
	if len(ctx.Policies) != 1 {
		t.Fatalf("expected 1 resolved policy, got %d", len(ctx.Policies))
	}
	policy := ctx.Policies[0]
	if policy.TargetEntity != "order" || policy.TargetField != "owner_user_id" || policy.Values[0] != "U1001" {
		t.Fatalf("unexpected policy: %+v", policy)
	}
}

func TestBuildSecureSemanticContextDeniesMissingPrincipal(t *testing.T) {
	cfg := `{
		"text2sql_mode":"query_plan",
		"semantic_model":{"entities":[{"name":"employee","table":"t_profile","fields":[{"name":"department","column":"department_name"}]}]},
		"access_policies":[{"name":"部门隔离","target_entity":"employee","target_field":"department","principal_field":"department","missing_principal":"deny"}]
	}`

	_, err := BuildSecureSemanticContext(cfg, map[string]interface{}{})
	if err == nil || !strings.Contains(err.Error(), "缺少当前用户字段") {
		t.Fatalf("expected missing principal error, got %v", err)
	}
}

func TestBuildSQLAppliesPolicyAndRequiredJoin(t *testing.T) {
	ctx := SecureSemanticContext{
		Enabled: true,
		Model: SemanticModel{
			Entities: []SemanticEntity{
				{Name: "order", Table: "t_order", Alias: "o", Fields: []SemanticField{{Name: "amount", Column: "pay_amount", Type: "number"}, {Name: "owner_user_id", Column: "company_user_id"}}},
				{Name: "employee", Table: "t_profile", Alias: "e", Fields: []SemanticField{{Name: "company_user_id", Column: "company_user_id"}, {Name: "department", Column: "department_name"}}},
			},
			Relations: []SemanticRelation{{FromEntity: "order", FromField: "owner_user_id", ToEntity: "employee", ToField: "company_user_id"}},
		},
		Policies: []ResolvedPolicy{{Name: "部门隔离", TargetEntity: "employee", TargetField: "department", Operator: "=", Values: []string{"销售部"}, Required: true}},
	}
	plan := QueryPlan{
		Select: []SelectExpr{{Entity: "order", Field: "amount", Agg: "SUM", Alias: "sales_amount"}},
		From:   "order",
		Limit:  100,
	}

	sql, err := BuildSQL(plan, ctx)
	if err != nil {
		t.Fatalf("BuildSQL returned error: %v", err)
	}
	wantParts := []string{
		"SELECT SUM(o.pay_amount) AS sales_amount",
		"FROM t_order o",
		"JOIN t_profile e ON o.company_user_id = e.company_user_id",
		"WHERE e.department_name = '销售部'",
		"LIMIT 100",
	}
	for _, part := range wantParts {
		if !strings.Contains(sql, part) {
			t.Fatalf("expected SQL to contain %q, got %s", part, sql)
		}
	}
}

func TestBuildSQLRejectsUnknownField(t *testing.T) {
	ctx := SecureSemanticContext{
		Enabled: true,
		Model:   SemanticModel{Entities: []SemanticEntity{{Name: "order", Table: "t_order", Alias: "o", Fields: []SemanticField{{Name: "amount", Column: "pay_amount"}}}}},
	}
	_, err := BuildSQL(QueryPlan{From: "order", Select: []SelectExpr{{Entity: "order", Field: "password"}}}, ctx)
	if err == nil || !strings.Contains(err.Error(), "字段不存在") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}
