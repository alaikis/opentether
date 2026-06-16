package text2sql

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SemanticModel 业务语义模型，将业务对象映射到物理表字段.
type SemanticModel struct {
	Entities        []SemanticEntity   `json:"entities"`
	Relations       []SemanticRelation `json:"relations"`
	Metrics         []SemanticMetric   `json:"metrics,omitempty"`
	DefaultFilters  []DefaultFilter    `json:"default_filters,omitempty"`
	IdentityMapping *IdentityMapping   `json:"identity_mapping,omitempty"` // 显式身份映射
}

func (m SemanticModel) FindEntity(name string) (SemanticEntity, bool) {
	for _, e := range m.Entities {
		if e.Name == name {
			return e, true
		}
	}
	return SemanticEntity{}, false
}

func (m SemanticModel) FindRelation(fromEntity, toEntity string) (SemanticRelation, bool) {
	for _, r := range m.Relations {
		if r.FromEntity == fromEntity && r.ToEntity == toEntity {
			return r, true
		}
	}
	for _, r := range m.Relations {
		if r.FromEntity == toEntity && r.ToEntity == fromEntity {
			return r.Reverse(), true
		}
	}
	return SemanticRelation{}, false
}

// FindIdentityMapping 查找显式身份映射配置。
func (m SemanticModel) FindIdentityMapping(entityName string) (IdentityMapping, bool) {
	if m.IdentityMapping != nil && m.IdentityMapping.Entity == entityName {
		return *m.IdentityMapping, true
	}
	return IdentityMapping{}, false
}

// SemanticEntity 业务实体定义，映射到一张物理表.
type SemanticEntity struct {
	Name   string          `json:"name"`
	Label  string          `json:"label"`
	Table  string          `json:"table"`
	Alias  string          `json:"alias"`
	Fields []SemanticField `json:"fields"`
}

func (e SemanticEntity) FindField(name string) (SemanticField, bool) {
	for _, f := range e.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return SemanticField{}, false
}

// SemanticField 业务字段，映射到底表字段与类型.
type SemanticField struct {
	Name   string `json:"name"`
	Label  string `json:"label"`
	Column string `json:"column"`
	Type   string `json:"type"`
}

// SemanticRelation 实体关系.
type SemanticRelation struct {
	FromEntity string `json:"from_entity"`
	FromField  string `json:"from_field"`
	ToEntity   string `json:"to_entity"`
	ToField    string `json:"to_field"`
	Type       string `json:"type"`
}

func (r SemanticRelation) Reverse() SemanticRelation {
	return SemanticRelation{
		FromEntity: r.ToEntity,
		FromField:  r.ToField,
		ToEntity:   r.FromEntity,
		ToField:    r.FromField,
		Type:       r.Type,
	}
}

// SemanticMetric 业务指标定义，如"订单数 = COUNT(t_order.id)".
type SemanticMetric struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Entity      string `json:"entity"`
	Aggregation string `json:"aggregation"` // COUNT, SUM, AVG, MIN, MAX
	Field       string `json:"field"`       // field name or "*" for COUNT
}

// DataPermission 数据权限定义，描述用户能看哪些数据.
type DataPermission struct {
	Name         string             `json:"name"`
	TargetEntity string             `json:"target_entity"`
	TargetField  string             `json:"target_field"`
	Operator     string             `json:"operator"`            // =, IN
	Values       []string           `json:"values"`              // 具体的值（从用户属性解析）
	Required     bool               `json:"required"`            // 缺少时是否拒绝查询
	JoinPath     []SemanticRelation `json:"join_path,omitempty"` // 从主实体到权限实体的关系路径
}

// DefaultFilter 默认业务过滤规则，在每次查询时自动注入，不由 LLM 决定。
type DefaultFilter struct {
	Entity      string `json:"entity"`      // 实体名，如 "order"
	Field       string `json:"field"`       // 字段名，如 "order_status"
	Operator    string `json:"operator"`    // !=, =, IN, >, <
	Value       string `json:"value"`       // 过滤值，如 "0"
	Description string `json:"description"` // 业务说明，如 "排除未确认订单"
}

// IdentityMapping 显式定义用户身份到业务实体的映射路径。
// 这是企业级安全的核心配置——不再依赖 BFS 自动猜测。
type IdentityMapping struct {
	Entity    string             `json:"entity"`     // 主实体，如 "order"
	UserField string             `json:"user_field"` // 用户标识字段，如 "global_user_id"
	Path      []IdentityPathStep `json:"path"`       // 从主实体到用户标识实体的关系路径
}

type IdentityPathStep struct {
	FromEntity string `json:"from_entity"`
	FromField  string `json:"from_field"`
	ToEntity   string `json:"to_entity"`
	ToField    string `json:"to_field"`
	MatchTo    string `json:"match_to,omitempty"` // "user" 表示最后一步和用户标识比对
}

// AccessPolicy 数据权限策略，声明谁对哪个实体的哪个字段有访问限制.
type AccessPolicy struct {
	Name             string   `json:"name"`
	Scope            string   `json:"scope"` // self / department / tenant / custom
	TargetEntity     string   `json:"target_entity"`
	TargetField      string   `json:"target_field"`
	PrincipalField   string   `json:"principal_field"`
	ApplyToGroups    []string `json:"apply_to_groups"`
	ExcludeGroups    []string `json:"exclude_groups"`
	ApplyToUsers     []string `json:"apply_to_users"`
	ExcludeUsers     []string `json:"exclude_users"`
	Operator         string   `json:"operator"`          // 仅 custom scope 使用，默认 "="
	MissingPrincipal string   `json:"missing_principal"` // deny / skip
}

// ResolvedPolicy 解析后的数据权限策略，包含具体值.
type ResolvedPolicy struct {
	Name         string
	TargetEntity string
	TargetField  string
	Operator     string
	Values       []string
	Required     bool
}

// SecureSemanticContext 安全查询上下文，包含语义模型和解析后的权限策略.
type SecureSemanticContext struct {
	Enabled           bool
	Model             SemanticModel
	Policies          []ResolvedPolicy
	RequiredJoinPairs []joinPair
}

type joinPair struct {
	FromEntity string
	ToEntity   string
}

// --------------------------------------------------------------------------
// 构造安全上下文
// --------------------------------------------------------------------------

// BuildSecureSemanticContext parses Skill.Config JSON and resolves access policies.
func BuildSecureSemanticContext(skillConfigJSON string, userCtx map[string]interface{}) (*SecureSemanticContext, error) {
	if strings.TrimSpace(skillConfigJSON) == "" {
		return &SecureSemanticContext{Enabled: false}, nil
	}
	var cfg struct {
		Text2SQLMode   string         `json:"text2sql_mode"`
		SemanticModel  *SemanticModel `json:"semantic_model"`
		AccessPolicies []AccessPolicy `json:"access_policies"`
	}
	if err := json.Unmarshal([]byte(skillConfigJSON), &cfg); err != nil {
		return &SecureSemanticContext{Enabled: false}, nil
	}
	if cfg.Text2SQLMode != "query_plan" || cfg.SemanticModel == nil || len(cfg.AccessPolicies) == 0 {
		return &SecureSemanticContext{Enabled: false}, nil
	}

	policies, joinPairs, err := resolveAccessPolicies(cfg.AccessPolicies, cfg.SemanticModel, userCtx)
	if err != nil {
		return nil, err
	}
	return &SecureSemanticContext{
		Enabled:           true,
		Model:             *cfg.SemanticModel,
		Policies:          policies,
		RequiredJoinPairs: joinPairs,
	}, nil
}

func resolveAccessPolicies(policies []AccessPolicy, model *SemanticModel, userCtx map[string]interface{}) ([]ResolvedPolicy, []joinPair, error) {
	var resolved []ResolvedPolicy
	var pairs []joinPair
	for _, policy := range policies {
		res, jp, err := resolveSinglePolicy(policy, model, userCtx)
		if err != nil {
			return nil, nil, err
		}
		if res != nil {
			resolved = append(resolved, *res)
			if jp != nil {
				pairs = append(pairs, *jp)
			}
		}
	}
	return resolved, pairs, nil
}

func resolveSinglePolicy(policy AccessPolicy, model *SemanticModel, userCtx map[string]interface{}) (*ResolvedPolicy, *joinPair, error) {
	// 排除判断
	if len(policy.ExcludeUsers) > 0 {
		if uid, ok := userCtx["user_id"].(string); ok && containsInSlice(policy.ExcludeUsers, uid) {
			return nil, nil, nil
		}
		if gid, ok := userCtx["global_user_id"].(string); ok && containsInSlice(policy.ExcludeUsers, gid) {
			return nil, nil, nil
		}
	}
	if len(policy.ExcludeGroups) > 0 {
		if gids, ok := userCtx["group_ids"].([]string); ok {
			for _, gid := range gids {
				if containsInSlice(policy.ExcludeGroups, gid) {
					return nil, nil, nil
				}
			}
		}
	}
	if len(policy.ApplyToUsers) > 0 {
		uid, _ := userCtx["user_id"].(string)
		gid, _ := userCtx["global_user_id"].(string)
		if !containsInSlice(policy.ApplyToUsers, uid) && !containsInSlice(policy.ApplyToUsers, gid) {
			return nil, nil, nil
		}
	}
	if len(policy.ApplyToGroups) > 0 {
		matched := false
		if gids, ok := userCtx["group_ids"].([]string); ok {
			for _, gid := range gids {
				if containsInSlice(policy.ApplyToGroups, gid) {
					matched = true
					break
				}
			}
		}
		if !matched {
			return nil, nil, nil
		}
	}

	principalVal, ok := resolvePrincipalValue(policy.PrincipalField, userCtx)
	if !ok || principalVal == "" {
		if policy.MissingPrincipal == "deny" {
			return nil, nil, fmt.Errorf("缺少当前用户字段 %s，无法执行受限查询", policy.PrincipalField)
		}
		return nil, nil, nil
	}

	op := "="
	if policy.Operator != "" {
		op = policy.Operator
	}

	resolved := &ResolvedPolicy{
		Name:         policy.Name,
		TargetEntity: policy.TargetEntity,
		TargetField:  policy.TargetField,
		Operator:     op,
		Values:       []string{principalVal},
		Required:     policy.MissingPrincipal == "deny",
	}

	var jp *joinPair
	if policy.Scope == "department" {
		_, exists := model.FindEntity(policy.TargetEntity)
		if !exists {
			return nil, nil, fmt.Errorf("权限策略引用的实体 %s 在语义模型中不存在", policy.TargetEntity)
		}
		if policy.TargetEntity == "employee" {
			// If the target entity is employee, we can directly apply department filter on it.
			// We still need to ensure it's joined if not the main entity.
			// But we don't know the main entity yet; we'll add it conditionally.
		}
	}

	return resolved, jp, nil
}

func resolvePrincipalValue(principalField string, userCtx map[string]interface{}) (string, bool) {
	aliases := map[string]string{
		"company_user_id": "global_user_id",
		"current_user_id": "user_id",
	}
	v, ok := userCtx[principalField]
	if !ok {
		if alias, exists := aliases[principalField]; exists {
			v, ok = userCtx[alias]
		}
	}
	if !ok || v == nil {
		return "", false
	}
	s, isValid := v.(string)
	if !isValid || strings.TrimSpace(s) == "" {
		return "", false
	}
	return s, true
}

func containsInSlice(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}
