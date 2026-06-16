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
	db *gorm.DB
}

func NewPlatformCoreService(db *gorm.DB) *PlatformCoreService {
	return &PlatformCoreService{db: db}
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
	run := &models.EvalRun{CaseID: c.ID, SkillID: c.SkillID, Status: "completed", Passed: false, Duration: time.Since(start).Milliseconds(), Output: "eval execution placeholder"}
	if strings.TrimSpace(c.ExpectedContains) == "" {
		run.Passed = true
	} else {
		run.Error = "Eval runner MVP 尚未执行真实 Agent 调用，仅保存用例和运行记录"
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
