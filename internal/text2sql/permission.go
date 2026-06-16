package text2sql

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ResolveUserPermissions traverses the entity graph to find the correct filter path.
// mainEntity is the entity the user is querying (e.g. "order").
// dataScope dictates the permission mode: "self", "department", or "all".
func ResolveUserPermissions(userCtx map[string]interface{}, dataScope string, model SemanticModel, mainEntity string) ([]DataPermission, error) {
	if len(userCtx) == 0 || dataScope == "all" || mainEntity == "" {
		return nil, nil
	}
	if _, ok := model.FindEntity(mainEntity); !ok {
		return nil, fmt.Errorf("主实体 %s 在语义模型中不存在", mainEntity)
	}
	switch dataScope {
	case "self":
		return resolveSelfPermission(userCtx, model, mainEntity)
	case "department":
		return resolveDepartmentPermission(userCtx, model, mainEntity)
	case "group":
		return resolveGroupPermission(userCtx, model, mainEntity)
	default:
		return nil, nil
	}
}

// ---- graph traversal --------------------------------------------------------

type entityGraph struct {
	adj    map[string][]graphEdge
	entIdx map[string]int
	model  *SemanticModel
}

type graphEdge struct {
	ToEntity    string
	FromField   string
	ToField     string
	RelationIdx int
	Reversed    bool
}

func buildGraph(model *SemanticModel) entityGraph {
	g := entityGraph{adj: map[string][]graphEdge{}, entIdx: map[string]int{}, model: model}
	for i, e := range model.Entities {
		g.entIdx[e.Name] = i
		g.adj[e.Name] = []graphEdge{}
	}
	for i, rel := range model.Relations {
		g.adj[rel.FromEntity] = append(g.adj[rel.FromEntity], graphEdge{
			ToEntity: rel.ToEntity, FromField: rel.FromField, ToField: rel.ToField, RelationIdx: i, Reversed: false,
		})
		g.adj[rel.ToEntity] = append(g.adj[rel.ToEntity], graphEdge{
			ToEntity: rel.FromEntity, FromField: rel.ToField, ToField: rel.FromField, RelationIdx: i, Reversed: true,
		})
	}
	return g
}

func (g entityGraph) bfsUserField(startEntity string, matchField func(SemanticField) bool, maxHops int) ([]graphEdge, string, string, bool) {
	if maxHops <= 0 {
		maxHops = 2
	}
	type bfsNode struct {
		entity string
		path   []graphEdge
	}
	visited := map[string]bool{startEntity: true}
	queue := []bfsNode{{entity: startEntity, path: nil}}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		if len(node.path) > 0 {
			ent := g.model.Entities[g.entIdx[node.entity]]
			for _, f := range ent.Fields {
				if matchField(f) {
					return node.path, node.entity, f.Name, true
				}
			}
		}
		if len(node.path) >= maxHops {
			continue
		}
		for _, edge := range g.adj[node.entity] {
			if visited[edge.ToEntity] {
				continue
			}
			visited[edge.ToEntity] = true
			newPath := make([]graphEdge, len(node.path)+1)
			copy(newPath, node.path)
			newPath[len(node.path)] = edge
			queue = append(queue, bfsNode{entity: edge.ToEntity, path: newPath})
		}
	}
	return nil, "", "", false
}

func (g entityGraph) edgesToRelations(path []graphEdge) []SemanticRelation {
	rels := make([]SemanticRelation, len(path))
	for i, e := range path {
		r := g.model.Relations[e.RelationIdx]
		if e.Reversed {
			r = r.Reverse()
		}
		rels[i] = r
	}
	return rels
}

// ---- self permission -------------------------------------------------------

func resolveSelfPermission(userCtx map[string]interface{}, model SemanticModel, mainEntity string) ([]DataPermission, error) {
	uid := getUserIdentifier(userCtx)
	if uid == "" {
		return nil, fmt.Errorf("缺少当前用户标识，无法执行本人数据隔离")
	}

	// 优先使用显式配置的身份映射路径（不依赖 BFS 猜测）
	if identityMapping, ok := model.FindIdentityMapping(mainEntity); ok {
		return buildExplicitPermission(uid, identityMapping, model)
	}

	// 回退：BFS 自动发现
	graph := buildGraph(&model)
	mainEnt, _ := model.FindEntity(mainEntity)
	for _, f := range mainEnt.Fields {
		if isUserIdentifierField(f) {
			return []DataPermission{{Name: "本人" + mainEnt.Label + "数据", TargetEntity: mainEntity, TargetField: f.Name, Operator: "=", Values: []string{uid}, Required: true}}, nil
		}
	}
	path, targetEntity, targetField, ok := graph.bfsUserField(mainEntity, isUserIdentifierField, 2)
	if !ok {
		return nil, fmt.Errorf("无法为实体 %s 找到本人数据过滤路径（请确认语义模型中配置了员工/用户实体及其关联关系）", mainEntity)
	}
	return []DataPermission{{Name: "本人" + targetEntity + "数据", TargetEntity: targetEntity, TargetField: targetField, Operator: "=", Values: []string{uid}, Required: true, JoinPath: graph.edgesToRelations(path)}}, nil
}

// buildExplicitPermission 从显式 IdentityMapping 生成权限
func buildExplicitPermission(uid string, im IdentityMapping, model SemanticModel) ([]DataPermission, error) {
	var joinPath []SemanticRelation
	var targetEntity string
	var targetField string

	for _, step := range im.Path {
		rel := SemanticRelation{
			FromEntity: step.FromEntity,
			FromField:  step.FromField,
			ToEntity:   step.ToEntity,
			ToField:    step.ToField,
			Type:       "many_to_one",
		}
		joinPath = append(joinPath, rel)
		if step.MatchTo == "user" {
			targetEntity = step.ToEntity
			targetField = step.ToField
		}
	}

	if targetEntity == "" {
		// 最后一步没标记 matchTo，用最后一个 step 的 toField
		last := joinPath[len(joinPath)-1]
		targetEntity = last.ToEntity
		targetField = last.ToField
	}

	return []DataPermission{{
		Name:         "本人" + targetEntity + "数据（显式映射）",
		TargetEntity: targetEntity,
		TargetField:  targetField,
		Operator:     "=",
		Values:       []string{uid},
		Required:     true,
		JoinPath:     joinPath,
	}}, nil
}

// ---- department permission -------------------------------------------------

func resolveDepartmentPermission(userCtx map[string]interface{}, model SemanticModel, mainEntity string) ([]DataPermission, error) {
	dept := getUserDepartment(userCtx)
	if dept == "" {
		return nil, fmt.Errorf("缺少当前用户部门信息，无法执行部门级数据隔离")
	}
	graph := buildGraph(&model)

	mainEnt, _ := model.FindEntity(mainEntity)
	for _, f := range mainEnt.Fields {
		if isDepartmentField(f) {
			return []DataPermission{{Name: "本部门" + mainEnt.Label + "数据", TargetEntity: mainEntity, TargetField: f.Name, Operator: "=", Values: []string{dept}, Required: true}}, nil
		}
	}

	path, targetEntity, targetField, ok := graph.bfsUserField(mainEntity, isDepartmentField, 2)
	if !ok {
		return nil, fmt.Errorf("无法为实体 %s 找到部门数据过滤路径（请确认语义模型中配置了部门实体及其关联关系）", mainEntity)
	}
	return []DataPermission{{Name: "本部门" + targetEntity + "数据", TargetEntity: targetEntity, TargetField: targetField, Operator: "=", Values: []string{dept}, Required: true, JoinPath: graph.edgesToRelations(path)}}, nil
}

// ---- group permission ------------------------------------------------------

func resolveGroupPermission(userCtx map[string]interface{}, model SemanticModel, mainEntity string) ([]DataPermission, error) {
	groups := getUserGroups(userCtx)
	if len(groups) == 0 {
		return nil, fmt.Errorf("当前用户不属于任何组，无法执行组级数据隔离")
	}
	graph := buildGraph(&model)

	mainEnt, _ := model.FindEntity(mainEntity)
	for _, f := range mainEnt.Fields {
		if isGroupField(f) {
			return []DataPermission{{Name: "本组" + mainEnt.Label + "数据", TargetEntity: mainEntity, TargetField: f.Name, Operator: "IN", Values: groups, Required: true}}, nil
		}
	}

	path, targetEntity, targetField, ok := graph.bfsUserField(mainEntity, isGroupField, 2)
	if !ok {
		return nil, fmt.Errorf("无法为实体 %s 找到组级数据过滤路径（请确认语义模型中配置了组/团队实体及其关联关系）", mainEntity)
	}
	return []DataPermission{{Name: "本组" + targetEntity + "数据", TargetEntity: targetEntity, TargetField: targetField, Operator: "IN", Values: groups, Required: true, JoinPath: graph.edgesToRelations(path)}}, nil
}

// ---- helpers ---------------------------------------------------------------

func isUserIdentifierField(f SemanticField) bool {
	lower := strings.ToLower(f.Name)
	for _, p := range []string{"user_id", "global_user_id", "company_user_id", "owner", "staff_id", "sale_staff"} {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func isDepartmentField(f SemanticField) bool {
	lower := strings.ToLower(f.Name)
	return strings.Contains(lower, "department") || strings.Contains(lower, "depart") || strings.Contains(lower, "dept")
}

func getUserIdentifier(userCtx map[string]interface{}) string {
	for _, key := range []string{"global_user_id", "company_user_id", "user_id"} {
		if v, ok := userCtx[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func getUserDepartment(userCtx map[string]interface{}) string {
	if v, ok := userCtx["department"].(string); ok && v != "" {
		return v
	}
	return ""
}

func getUserGroups(userCtx map[string]interface{}) []string {
	var result []string
	for _, key := range []string{"group_ids", "group_codes", "group_names"} {
		if v, ok := userCtx[key]; ok {
			switch vals := v.(type) {
			case []string:
				result = append(result, vals...)
			case []interface{}:
				for _, item := range vals {
					if s, ok := item.(string); ok {
						result = append(result, s)
					}
				}
			case string:
				if vals != "" {
					result = append(result, vals)
				}
			}
		}
	}
	return result
}

func isGroupField(f SemanticField) bool {
	lower := strings.ToLower(f.Name)
	for _, p := range []string{"team_id", "team", "group_id", "group", "department_id"} {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// ---- auto discovery --------------------------------------------------------

func AutoDiscoverModel(schemaJSON string) SemanticModel {
	var rawTables []struct {
		Name    string `json:"name"`
		Columns []struct {
			Name    string `json:"name"`
			Type    string `json:"type"`
			KeyType string `json:"key_type"`
			Comment string `json:"comment"`
		} `json:"columns"`
	}
	if json.Unmarshal([]byte(schemaJSON), &rawTables) != nil {
		return SemanticModel{}
	}
	model := SemanticModel{}
	entityNames := map[string]bool{}
	for _, table := range rawTables {
		entity := SemanticEntity{
			Name:  suggestEntityName(table.Name),
			Label: suggestEntityLabel(table.Name),
			Table: table.Name,
			Alias: suggestEntityAlias(table.Name),
		}
		for _, col := range table.Columns {
			entity.Fields = append(entity.Fields, SemanticField{
				Name: suggestFieldName(col.Name), Label: suggestFieldLabel(col.Name, col.Comment),
				Column: col.Name, Type: col.Type,
			})
		}
		if entityNames[entity.Name] {
			entity.Name = entity.Name + "_" + table.Name
		}
		entityNames[entity.Name] = true
		model.Entities = append(model.Entities, entity)
	}
	model.Relations = discoverRelations(model.Entities)
	return model
}

func suggestEntityName(tableName string) string {
	name := strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(tableName, "t_"), "tb_"), "tbl_")
	return strings.ToLower(name)
}

func suggestEntityLabel(tableName string) string {
	return suggestEntityName(tableName)
}

func suggestEntityAlias(tableName string) string {
	name := suggestEntityName(tableName)
	if len(name) > 1 {
		return string(name[0])
	}
	return "x"
}

func suggestFieldName(columnName string) string {
	return columnName
}

func suggestFieldLabel(columnName, comment string) string {
	if comment != "" {
		return comment
	}
	return columnName
}

func discoverRelations(entities []SemanticEntity) []SemanticRelation {
	var rels []SemanticRelation

	for i := range entities {
		for j := range entities {
			if i == j {
				continue
			}

			// Case 1: FK column → primary key (e.g., order.sale_staff_id → employee.id)
			for _, fi := range entities[i].Fields {
				if !strings.HasSuffix(fi.Column, "_id") {
					continue
				}
				for _, fj := range entities[j].Fields {
					if fj.Column != "id" {
						continue
					}
					// fi.Column="sale_staff_id" references entities[j].id
					// Check if fi.Column hints at entities[j]'s table name
					fkBare := strings.TrimSuffix(fi.Column, "_id") // "sale_staff"
					_ = fkBare
					if isRelatedFK(fi.Column, entities[j].Table) {
						addRel(&rels, entities[i].Name, fi.Name, entities[j].Name, fj.Name)
					}
				}
			}

			// Case 2: Same column name (shared value) in different tables
			for _, fi := range entities[i].Fields {
				for _, fj := range entities[j].Fields {
					if fi.Column == fj.Column && fi.Column != "" && fi.Column != "id" &&
						strings.HasSuffix(fi.Column, "_id") {
						addRel(&rels, entities[i].Name, fi.Name, entities[j].Name, fj.Name)
					}
				}
			}
		}
	}
	return rels
}

func isRelatedFK(fkColumn, targetTable string) bool {
	bare := strings.TrimSuffix(strings.TrimPrefix(fkColumn, "t_"), "_id")
	targetBare := strings.TrimPrefix(strings.TrimPrefix(targetTable, "t_"), "tb_")
	return strings.EqualFold(bare, targetBare)
}

func addRel(rels *[]SemanticRelation, fromEntity, fromField, toEntity, toField string) {
	r := SemanticRelation{FromEntity: fromEntity, FromField: fromField, ToEntity: toEntity, ToField: toField, Type: "many_to_one"}
	if !hasRelation(*rels, r) {
		*rels = append(*rels, r)
	}
}

func hasRelation(rels []SemanticRelation, rel SemanticRelation) bool {
	for _, r := range rels {
		if r.FromEntity == rel.FromEntity && r.ToEntity == rel.ToEntity && r.FromField == rel.FromField {
			return true
		}
	}
	return false
}
