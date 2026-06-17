package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

type SkillValidationIssue struct {
	Level   string `json:"level"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type SkillValidationResult struct {
	SkillID string                 `json:"skill_id"`
	Valid   bool                   `json:"valid"`
	Issues  []SkillValidationIssue `json:"issues"`
}

type PlatformCoreService struct {
	db    *gorm.DB
	agent *AgentService
}

func NewPlatformCoreService(db *gorm.DB) *PlatformCoreService {
	return &PlatformCoreService{db: db}
}

func (s *PlatformCoreService) SetAgentService(agent *AgentService) {
	s.agent = agent
}

func (s *PlatformCoreService) ValidateSkill(skillID string) (*SkillValidationResult, error) {
	var skill models.Skill
	if err := s.db.First(&skill, "id = ?", skillID).Error; err != nil {
		return nil, err
	}
	result := &SkillValidationResult{SkillID: skill.ID, Valid: true, Issues: []SkillValidationIssue{}}
	cfg := map[string]interface{}{}
	if strings.TrimSpace(skill.Config) != "" {
		if err := json.Unmarshal([]byte(skill.Config), &cfg); err != nil {
			result.add("error", "invalid_config_json", "Skill config 不是合法 JSON")
			return result, nil
		}
	}
	if skill.SkillType == "text2sql" {
		if strings.TrimSpace(asString(cfg["data_source_id"])) == "" {
			result.add("error", "missing_data_source", "Text2SQL Skill 缺少 data_source_id")
		}
		if len(asSlice(cfg["selected_tables"])) == 0 {
			result.add("warning", "missing_selected_tables", "缺少 selected_tables，LLM 可能看到过多或过少 schema")
		}
		if len(asSlice(cfg["table_relations"])) == 0 {
			result.add("warning", "missing_relations", "缺少 table_relations，JOIN 路径可能不稳定")
		}
		if len(asSlice(cfg["metric_rules"])) == 0 {
			result.add("warning", "missing_metrics", "缺少 metric_rules，指标口径可能不稳定")
		}
		if len(asSlice(cfg["entity_rules"])) == 0 {
			result.add("warning", "missing_entities", "缺少 entity_rules，自然语言实体映射可能不稳定")
		}
	}
	if strings.TrimSpace(asString(cfg["context_md"])) == "" && strings.TrimSpace(asString(cfg["context_md_path"])) == "" {
		result.add("warning", "missing_context_md", "缺少 Skill MD 调用说明/业务说明")
	}
	result.Valid = true
	for _, issue := range result.Issues {
		if issue.Level == "error" {
			result.Valid = false
		}
	}
	return result, nil
}

func (r *SkillValidationResult) add(level, code, message string) {
	r.Issues = append(r.Issues, SkillValidationIssue{Level: level, Code: code, Message: message})
}

func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asSlice(v interface{}) []interface{} {
	if s, ok := v.([]interface{}); ok {
		return s
	}
	return nil
}

func (s *PlatformCoreService) StartSkillBootstrap(skillID string) (map[string]interface{}, error) {
	started := time.Now()
	go func() {
		_ = s.bootstrapSkillRules(skillID)
	}()
	return map[string]interface{}{"started": true, "skill_id": skillID, "started_at": started}, nil
}

func (s *PlatformCoreService) bootstrapSkillRules(skillID string) error {
	var skill models.Skill
	if err := s.db.First(&skill, "id = ?", skillID).Error; err != nil {
		return err
	}
	cfg := map[string]interface{}{}
	_ = json.Unmarshal([]byte(skill.Config), &cfg)
	dataSourceID := asString(cfg["data_source_id"])
	seedFromArray := func(kind string, arr []interface{}) {
		for i, raw := range arr {
			b, _ := json.Marshal(raw)
			content := stripEnvironmentDetails(string(b))
			key := kind + "_" + fmt.Sprintf("%03d", i+1)
			s.upsertBootstrapMemory(skill.ID, dataSourceID, kind, key, content)
		}
	}
	seedFromArray("metric_rule", asSlice(cfg["metric_rules"]))
	seedFromArray("entity_rule", asSlice(cfg["entity_rules"]))
	seedFromArray("dimension_rule", asSlice(cfg["dimension_rules"]))
	var runtime []models.SkillRuntimeMemory
	s.db.Where("skill_id = ? AND source IN ?", skill.ID, []string{"runtime", "confirmed"}).Order("confidence DESC, updated_at DESC").Limit(100).Find(&runtime)
	for _, mem := range runtime {
		if mem.Type == "sql_pattern" || mem.Type == "table_relation" || mem.Type == "metric_rule" || mem.Type == "text2sql_template" {
			s.upsertBootstrapMemory(skill.ID, dataSourceID, mem.Type, "bootstrap_"+mem.Key, stripEnvironmentDetails(mem.Content))
		}
	}
	return nil
}

func (s *PlatformCoreService) upsertBootstrapMemory(skillID, dataSourceID, memType, key, content string) {
	if strings.TrimSpace(content) == "" || strings.Contains(content, "<environment_details>") {
		return
	}
	var existing models.SkillRuntimeMemory
	if err := s.db.Where("skill_id = ? AND data_source_id = ? AND type = ? AND key = ? AND source = ?", skillID, dataSourceID, memType, key, "bootstrap").First(&existing).Error; err == nil {
		s.db.Model(&existing).Updates(map[string]interface{}{"content": content, "confidence": 0.7, "status": "pending", "updated_at": time.Now()})
		return
	}
	_ = s.db.Create(&models.SkillRuntimeMemory{SkillID: skillID, DataSourceID: dataSourceID, Type: memType, Key: key, Content: content, Confidence: 0.7, Source: "bootstrap", Status: "pending", LastUsedAt: time.Now()}).Error
}

func (s *PlatformCoreService) ListPolicies() ([]models.AccessPolicy, error) {
	var rows []models.AccessPolicy
	return rows, s.db.Order("created_at DESC").Find(&rows).Error
}

func (s *PlatformCoreService) SavePolicy(row *models.AccessPolicy) error {
	return s.db.Save(row).Error
}

func (s *PlatformCoreService) DeletePolicy(id string) error {
	return s.db.Delete(&models.AccessPolicy{}, "id = ?", id).Error
}

func (s *PlatformCoreService) EvaluatePolicy(scope, resource string, ctx map[string]interface{}) (map[string]interface{}, error) {
	var policies []models.AccessPolicy
	if err := s.db.Where("enabled = ? AND scope = ? AND (resource = ? OR resource = ?)", true, scope, resource, "*").Find(&policies).Error; err != nil {
		return nil, err
	}
	decision := "allow"
	matched := []models.AccessPolicy{}
	for _, p := range policies {
		matched = append(matched, p)
		if p.Effect == "deny" {
			decision = "deny"
		}
	}
	return map[string]interface{}{"decision": decision, "matched": matched, "context": ctx}, nil
}

func (s *PlatformCoreService) ListPrecomputeJobs() ([]models.PrecomputeJob, error) {
	var rows []models.PrecomputeJob
	return rows, s.db.Order("created_at DESC").Find(&rows).Error
}

func (s *PlatformCoreService) SavePrecomputeJob(row *models.PrecomputeJob) error {
	return s.db.Save(row).Error
}

func (s *PlatformCoreService) DeletePrecomputeJob(id string) error {
	return s.db.Delete(&models.PrecomputeJob{}, "id = ?", id).Error
}

func (s *PlatformCoreService) RunPrecomputeJob(id string) (*models.PrecomputeJob, error) {
	var row models.PrecomputeJob
	if err := s.db.First(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if strings.TrimSpace(row.TargetTable) == "" || strings.TrimSpace(row.SQL) == "" {
		row.Status = "failed"
		row.LastError = "target_table/sql required"
		_ = s.db.Save(&row).Error
		return &row, nil
	}
	if !safeSQLIdentifier(row.TargetTable) || !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(row.SQL)), "SELECT") {
		row.Status = "failed"
		row.LastError = "only safe target table names and SELECT SQL are allowed"
		_ = s.db.Save(&row).Error
		return &row, nil
	}
	if err := s.db.Exec("DROP TABLE IF EXISTS `" + row.TargetTable + "`").Error; err != nil {
		row.Status = "failed"
		row.LastError = err.Error()
		_ = s.db.Save(&row).Error
		return &row, nil
	}
	if err := s.db.Exec("CREATE TABLE `" + row.TargetTable + "` AS " + row.SQL).Error; err != nil {
		row.Status = "failed"
		row.LastError = err.Error()
		_ = s.db.Save(&row).Error
		return &row, nil
	}
	now := time.Now()
	row.LastRunAt = &now
	row.Status = "completed"
	row.LastError = ""
	return &row, s.db.Save(&row).Error
}

func safeSQLIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func (s *PlatformCoreService) ListEvalCases() ([]models.EvalCase, error) {
	var rows []models.EvalCase
	return rows, s.db.Order("created_at DESC").Find(&rows).Error
}

func (s *PlatformCoreService) SaveEvalCase(row *models.EvalCase) error {
	return s.db.Save(row).Error
}

func (s *PlatformCoreService) DeleteEvalCase(id string) error {
	return s.db.Delete(&models.EvalCase{}, "id = ?", id).Error
}

func (s *PlatformCoreService) RunEvalCase(id string) (*models.EvalRun, error) {
	start := time.Now()
	var c models.EvalCase
	if err := s.db.First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	run := &models.EvalRun{CaseID: c.ID, SkillID: c.SkillID, Status: "completed", Passed: false}
	if s.agent == nil {
		run.Error = "agent service not configured"
		run.Duration = time.Since(start).Milliseconds()
		return run, s.db.Create(run).Error
	}
	var user models.User
	if err := s.db.Where("role = ?", "admin").First(&user).Error; err != nil {
		run.Error = err.Error()
		run.Duration = time.Since(start).Milliseconds()
		return run, s.db.Create(run).Error
	}
	resp, err := s.agent.Chat(user.ID, c.Question, "", c.SkillID)
	run.Duration = time.Since(start).Milliseconds()
	if err != nil {
		run.Error = err.Error()
		return run, s.db.Create(run).Error
	}
	b, _ := json.Marshal(resp)
	run.Output = string(b)
	run.Passed = true
	if strings.TrimSpace(c.ExpectedContains) != "" && !strings.Contains(run.Output, c.ExpectedContains) {
		run.Passed = false
		run.Error = "output missing expected_contains"
	}
	if strings.TrimSpace(c.ExpectedSQLContains) != "" && !strings.Contains(run.Output, c.ExpectedSQLContains) {
		run.Passed = false
		run.Error = "output missing expected_sql_contains"
	}
	return run, s.db.Create(run).Error
}

func (s *PlatformCoreService) ListEvalRuns(caseID string) ([]models.EvalRun, error) {
	var rows []models.EvalRun
	q := s.db.Order("created_at DESC")
	if caseID != "" {
		q = q.Where("case_id = ?", caseID)
	}
	return rows, q.Find(&rows).Error
}
