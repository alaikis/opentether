package text2sql

import (
	"encoding/json"
	"fmt"
	"strings"
)

// QueryPlan is the structured query plan that LLM generates instead of raw SQL.
type QueryPlan struct {
	Select  []SelectExpr `json:"select"`
	From    string       `json:"from"`
	Joins   []JoinExpr   `json:"joins,omitempty"`
	Filters []FilterExpr `json:"filters,omitempty"`
	GroupBy []FieldRef   `json:"group_by,omitempty"`
	OrderBy []OrderExpr  `json:"order_by,omitempty"`
	Limit   int          `json:"limit"`
}

type SelectExpr struct {
	Entity string `json:"entity"`
	Field  string `json:"field"`
	Agg    string `json:"agg,omitempty"`
	Alias  string `json:"alias,omitempty"`
}

type JoinExpr struct {
	Entity     string `json:"entity"`
	FromEntity string `json:"from_entity"`
	FromField  string `json:"from_field"`
	ToField    string `json:"to_field"`
	Type       string `json:"type"` // inner / left
}

type FilterExpr struct {
	Entity string `json:"entity"`
	Field  string `json:"field"`
	Op     string `json:"op"`
	Value  string `json:"value"`
}

type FieldRef struct {
	Entity string `json:"entity"`
	Field  string `json:"field"`
}

type OrderExpr struct {
	Entity string `json:"entity"`
	Field  string `json:"field"`
	Desc   bool   `json:"desc,omitempty"`
}

// BuildSQL generates a safe SQL query from a QueryPlan, semantic context, and optional permissions.
func BuildSQL(plan QueryPlan, ctx SecureSemanticContext, perms ...DataPermission) (string, error) {
	if !ctx.Enabled {
		return "", fmt.Errorf("安全语义上下文未启用")
	}

	if err := validateAndEnrich(&plan, &ctx); err != nil {
		return "", err
	}

	var sql strings.Builder

	// SELECT
	selectParts := make([]string, 0, len(plan.Select))
	for _, sel := range plan.Select {
		ent, _ := ctx.Model.FindEntity(sel.Entity)
		field, _ := ent.FindField(sel.Field)
		col := fmt.Sprintf("%s.%s", ent.Alias, safeSQLIdent(field.Column))
		if sel.Agg != "" {
			col = fmt.Sprintf("%s(%s)", strings.ToUpper(sel.Agg), col)
		}
		if sel.Alias != "" || sel.Agg != "" {
			alias := sel.Alias
			if alias == "" {
				alias = sel.Field
			}
			col += " AS " + safeSQLIdent(alias)
		}
		selectParts = append(selectParts, col)
	}
	sql.WriteString("SELECT " + strings.Join(selectParts, ", ") + "\n")

	// FROM
	mainEntity, _ := ctx.Model.FindEntity(plan.From)
	sql.WriteString(fmt.Sprintf("FROM %s %s\n", safeSQLIdent(mainEntity.Table), mainEntity.Alias))

	// JOINs (auto-injected required joins from policies)
	for _, pair := range ctx.RequiredJoinPairs {
		rel, ok := ctx.Model.FindRelation(pair.FromEntity, pair.ToEntity)
		if !ok {
			return "", fmt.Errorf("未找到实体关系: %s -> %s", pair.FromEntity, pair.ToEntity)
		}
		fromEnt, _ := ctx.Model.FindEntity(rel.FromEntity)
		toEnt, _ := ctx.Model.FindEntity(rel.ToEntity)
		sql.WriteString(fmt.Sprintf("JOIN %s %s ON %s.%s = %s.%s\n",
			safeSQLIdent(toEnt.Table), toEnt.Alias,
			fromEnt.Alias, safeSQLIdent(fromEnt.FindFieldColumn(rel.FromField)),
			toEnt.Alias, safeSQLIdent(toEnt.FindFieldColumn(rel.ToField)),
		))
	}

	// Explicit user joins
	for _, j := range plan.Joins {
		toEnt, ok := ctx.Model.FindEntity(j.Entity)
		if !ok {
			return "", fmt.Errorf("JOIN 引用的实体 %s 在语义模型中不存在", j.Entity)
		}
		rel, ok := ctx.Model.FindRelation(j.FromEntity, j.Entity)
		if !ok {
			return "", fmt.Errorf("未找到实体关系: %s -> %s", j.FromEntity, j.Entity)
		}
		fromEnt, ok := ctx.Model.FindEntity(j.FromEntity)
		if !ok {
			return "", fmt.Errorf("JOIN 源实体 %s 在语义模型中不存在", j.FromEntity)
		}
		joinType := "JOIN"
		if strings.ToUpper(j.Type) == "LEFT" {
			joinType = "LEFT JOIN"
		}
		sql.WriteString(fmt.Sprintf("%s %s %s ON %s.%s = %s.%s\n",
			joinType, safeSQLIdent(toEnt.Table), toEnt.Alias,
			fromEnt.Alias, safeSQLIdent(fromEnt.FindFieldColumn(rel.FromField)),
			toEnt.Alias, safeSQLIdent(toEnt.FindFieldColumn(rel.ToField)),
		))
	}

	// WHERE (permissions + policy conditions + user filters)
	var conditions []string

	// 1. 数据权限（由系统强制执行，不由 LLM 决定）
	for _, perm := range perms {
		// Inject required JOINs from permission path
		for _, rel := range perm.JoinPath {
			toEnt, ok := ctx.Model.FindEntity(rel.ToEntity)
			if !ok {
				continue
			}
			fromEnt, ok := ctx.Model.FindEntity(rel.FromEntity)
			if !ok {
				continue
			}
			sql.WriteString(fmt.Sprintf("JOIN %s %s ON %s.%s = %s.%s\n",
				safeSQLIdent(toEnt.Table), toEnt.Alias,
				fromEnt.Alias, safeSQLIdent(fromEnt.FindFieldColumn(rel.FromField)),
				toEnt.Alias, safeSQLIdent(toEnt.FindFieldColumn(rel.ToField)),
			))
		}

		ent, ok := ctx.Model.FindEntity(perm.TargetEntity)
		if !ok {
			continue
		}
		field, ok := ent.FindField(perm.TargetField)
		if !ok {
			continue
		}
		if len(perm.Values) == 1 {
			cond := fmt.Sprintf("%s.%s %s '%s'",
				ent.Alias, safeSQLIdent(field.Column),
				perm.Operator, escapeSQL(perm.Values[0]))
			conditions = append(conditions, cond)
		} else if len(perm.Values) > 1 {
			quoted := make([]string, len(perm.Values))
			for i, v := range perm.Values {
				quoted[i] = "'" + escapeSQL(v) + "'"
			}
			cond := fmt.Sprintf("%s.%s IN (%s)",
				ent.Alias, safeSQLIdent(field.Column),
				strings.Join(quoted, ","))
			conditions = append(conditions, cond)
		}
	}

	// 1.5. 默认业务过滤（如"排除未确认订单"，系统强制执行）
	for _, df := range ctx.Model.DefaultFilters {
		ent, ok := ctx.Model.FindEntity(df.Entity)
		if !ok {
			continue
		}
		field, ok := ent.FindField(df.Field)
		if !ok {
			continue
		}
		cond := fmt.Sprintf("%s.%s %s '%s'",
			ent.Alias, safeSQLIdent(field.Column),
			df.Operator, escapeSQL(df.Value))
		conditions = append(conditions, cond)
	}

	// 2. 解析后的策略条件
	for _, pol := range ctx.Policies {
		ent, ok := ctx.Model.FindEntity(pol.TargetEntity)
		if !ok {
			continue
		}
		field, ok := ent.FindField(pol.TargetField)
		if !ok {
			continue
		}
		for _, val := range pol.Values {
			cond := fmt.Sprintf("%s.%s %s '%s'",
				ent.Alias, safeSQLIdent(field.Column),
				pol.Operator, escapeSQL(val))
			conditions = append(conditions, cond)
		}
	}
	for _, f := range plan.Filters {
		ent, ok := ctx.Model.FindEntity(f.Entity)
		if !ok {
			return "", fmt.Errorf("过滤条件引用的实体 %s 不存在", f.Entity)
		}
		field, ok := ent.FindField(f.Field)
		if !ok {
			return "", fmt.Errorf("过滤条件引用的字段 %s.%s 不存在", f.Entity, f.Field)
		}
		op := f.Op
		if op == "" {
			op = "="
		}
		cond := fmt.Sprintf("%s.%s %s '%s'",
			ent.Alias, safeSQLIdent(field.Column),
			strings.ToUpper(op), escapeSQL(f.Value))
		conditions = append(conditions, cond)
	}
	if len(conditions) > 0 {
		sql.WriteString("WHERE " + strings.Join(conditions, " AND ") + "\n")
	}

	// GROUP BY
	if len(plan.GroupBy) > 0 {
		groupParts := make([]string, 0, len(plan.GroupBy))
		for _, g := range plan.GroupBy {
			ent, _ := ctx.Model.FindEntity(g.Entity)
			field, _ := ent.FindField(g.Field)
			groupParts = append(groupParts, fmt.Sprintf("%s.%s", ent.Alias, safeSQLIdent(field.Column)))
		}
		sql.WriteString("GROUP BY " + strings.Join(groupParts, ", ") + "\n")
	}

	// ORDER BY
	if len(plan.OrderBy) > 0 {
		orderParts := make([]string, 0, len(plan.OrderBy))
		for _, o := range plan.OrderBy {
			ent, _ := ctx.Model.FindEntity(o.Entity)
			field, _ := ent.FindField(o.Field)
			dir := "ASC"
			if o.Desc {
				dir = "DESC"
			}
			orderParts = append(orderParts, fmt.Sprintf("%s.%s %s", ent.Alias, safeSQLIdent(field.Column), dir))
		}
		sql.WriteString("ORDER BY " + strings.Join(orderParts, ", ") + "\n")
	}

	// LIMIT
	if plan.Limit > 0 {
		sql.WriteString(fmt.Sprintf("LIMIT %d\n", plan.Limit))
	} else {
		sql.WriteString("LIMIT 1000\n")
	}

	return strings.TrimSpace(sql.String()), nil
}

func validateAndEnrich(plan *QueryPlan, ctx *SecureSemanticContext) error {
	if strings.TrimSpace(plan.From) == "" {
		return fmt.Errorf("QueryPlan 缺少 from 字段")
	}
	_, ok := ctx.Model.FindEntity(plan.From)
	if !ok {
		return fmt.Errorf("主实体 %s 在语义模型中不存在", plan.From)
	}
	if len(plan.Select) == 0 {
		return fmt.Errorf("QueryPlan 缺少 select 字段")
	}
	for _, sel := range plan.Select {
		ent, ok := ctx.Model.FindEntity(sel.Entity)
		if !ok {
			return fmt.Errorf("SELECT 引用的实体 %s 在语义模型中不存在", sel.Entity)
		}
		_, ok = ent.FindField(sel.Field)
		if !ok {
			return fmt.Errorf("字段不存在: %s.%s", sel.Entity, sel.Field)
		}
		agg := strings.ToUpper(sel.Agg)
		if agg != "" && agg != "COUNT" && agg != "SUM" && agg != "AVG" && agg != "MIN" && agg != "MAX" {
			return fmt.Errorf("不支持的聚合函数: %s", sel.Agg)
		}
	}

	// Auto-inject required joins from policies
	requiredEntities := make(map[string]bool)
	for _, pol := range ctx.Policies {
		requiredEntities[pol.TargetEntity] = true
	}
	for _, pair := range ctx.RequiredJoinPairs {
		requiredEntities[pair.ToEntity] = true
	}

	mainEntity := plan.From
	for _, targetEntity := range sortedKeys(requiredEntities) {
		if targetEntity == mainEntity {
			continue
		}
		rel, ok := ctx.Model.FindRelation(mainEntity, targetEntity)
		if !ok {
			return fmt.Errorf("权限策略要求 join 实体 %s，但语义模型中未找到从 %s 到 %s 的关系",
				targetEntity, mainEntity, targetEntity)
		}
		fromEnt, _ := ctx.Model.FindEntity(rel.FromEntity)
		toEnt, _ := ctx.Model.FindEntity(rel.ToEntity)
		// If the policy entity becomes the "from" side in the relation, we need to adjust.
		var pair joinPair
		if fromEnt.Name == mainEntity && toEnt.Name == targetEntity {
			pair = joinPair{FromEntity: fromEnt.Name, ToEntity: toEnt.Name}
		} else if toEnt.Name == mainEntity && fromEnt.Name == targetEntity {
			pair = joinPair{FromEntity: toEnt.Name, ToEntity: fromEnt.Name}
		} else {
			return fmt.Errorf("关系 %s -> %s 不直接连接主实体 %s", rel.FromEntity, rel.ToEntity, mainEntity)
		}
		ctx.RequiredJoinPairs = append(ctx.RequiredJoinPairs, pair)
	}

	return nil
}

func escapeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// SafeSemanticEntity access from model by name
func (m SemanticModel) FindRelationEntity(name string, model SemanticModel) SemanticEntity {
	ent, _ := model.FindEntity(name)
	return ent
}

// SemanticEntity helper methods.
func (e SemanticEntity) FindFieldColumn(name string) string {
	f, ok := e.FindField(name)
	if !ok {
		return name
	}
	return f.Column
}

// RenderPrompt generates a safe LLM prompt from the semantic context.
func (ctx SecureSemanticContext) RenderPrompt() string {
	if !ctx.Enabled {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("你只能使用以下安全语义对象生成 JSON 查询计划，不要生成 SQL，不要使用未列出的实体或字段:\n\n")

	for _, entity := range ctx.Model.Entities {
		sb.WriteString(fmt.Sprintf("实体: %s (%s)\n", entity.Name, entity.Label))
		for _, field := range entity.Fields {
			sb.WriteString(fmt.Sprintf("  - %s (%s)\n", field.Name, field.Label))
		}
		sb.WriteString("\n")
	}

	if len(ctx.Model.Relations) > 0 {
		sb.WriteString("实体关系:\n")
		for _, rel := range ctx.Model.Relations {
			sb.WriteString(fmt.Sprintf("  %s.%s -> %s.%s\n",
				rel.FromEntity, rel.FromField, rel.ToEntity, rel.ToField))
		}
		sb.WriteString("\n")
	}

	if len(ctx.Policies) > 0 {
		sb.WriteString("强制数据边界（系统会自动注入，你无需在查询计划中重复添加）:\n")
		for _, pol := range ctx.Policies {
			required := ""
			if pol.Required {
				required = "[必须]"
			}
			sb.WriteString(fmt.Sprintf("  - %s: %s.%s = %v %s\n",
				pol.Name, pol.TargetEntity, pol.TargetField, pol.Values, required))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("输出格式（必须是合法的 JSON）:\n")
	sb.WriteString(`{
  "select": [{"entity":"实体名","field":"字段名","agg":"SUM|COUNT|AVG|MIN|MAX(可选)","alias":"别名(可选)"}],
  "from": "主实体名",
  "joins": [{"entity":"目标实体","from_entity":"源实体","from_field":"源字段","to_field":"目标字段"}],
  "filters": [{"entity":"实体名","field":"字段名","op":"=|!=|>|<|>=|<=","value":"比较值"}],
  "group_by": [{"entity":"实体名","field":"字段名"}],
  "order_by": [{"entity":"实体名","field":"字段名","desc":true|false}],
  "limit": 100
}
`)
	sb.WriteString("请只输出 JSON，不要包含任何解释或 markdown 标记。")

	return sb.String()
}

// ParseQueryPlan parses LLM response into QueryPlan.
func ParseQueryPlan(content string) (*QueryPlan, error) {
	cleaned := extractJSONBlock(content)
	var plan QueryPlan
	if err := json.Unmarshal([]byte(cleaned), &plan); err != nil {
		return nil, fmt.Errorf("解析 QueryPlan 失败: %w", err)
	}
	if plan.Limit == 0 {
		plan.Limit = 100
	}
	return &plan, nil
}

func extractJSONBlock(content string) string {
	content = strings.TrimSpace(content)
	if start := strings.Index(content, "```"); start >= 0 {
		content = content[start+3:]
		if nl := strings.IndexByte(content, '\n'); nl >= 0 {
			content = content[nl+1:]
		}
		if end := strings.LastIndex(content, "```"); end >= 0 {
			content = content[:end]
		}
	}
	return strings.TrimSpace(content)
}
