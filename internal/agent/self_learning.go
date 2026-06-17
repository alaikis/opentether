package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

type SelfLearning struct {
	db       *gorm.DB
	mu       sync.Mutex
	patterns map[string]failurePattern
}

type failurePattern struct {
	Query     string
	Count     int
	LastError string
	FirstSeen time.Time
	LastSeen  time.Time
}

func NewSelfLearning(db *gorm.DB) *SelfLearning {
	return &SelfLearning{db: db, patterns: make(map[string]failurePattern)}
}

func (s *SelfLearning) RecordFailure(query, errorMsg string) {
	normalized := normalizeForLearning(query)
	s.mu.Lock()
	pattern := s.patterns[normalized]
	pattern.Count++
	pattern.LastError = errorMsg
	pattern.LastSeen = time.Now()
	if pattern.FirstSeen.IsZero() {
		pattern.FirstSeen = time.Now()
		pattern.Query = query
	}
	s.patterns[normalized] = pattern
	s.mu.Unlock()
	if pattern.Count >= 3 {
		s.generateImprovementTemplate(normalized, pattern)
	}
}

func (s *SelfLearning) RecordFeedback(query, correction string) {
	normalized := normalizeForLearning(query)
	s.mu.Lock()
	delete(s.patterns, normalized)
	s.mu.Unlock()
	mem := models.SkillRuntimeMemory{
		Type:       "user_feedback",
		Key:        "feedback_" + normalized,
		Content:    fmt.Sprintf("用户对查询的纠正: 原始=%s 纠正=%s", query, correction),
		Confidence: 0.8,
		Source:     "feedback",
		Status:     "active",
		LastUsedAt: time.Now(),
	}
	s.db.Create(&mem)
}

func (s *SelfLearning) generateImprovementTemplate(normalized string, pattern failurePattern) {
	content := fmt.Sprintf("失败查询: %s, 错误: %s, 次数: %d", pattern.Query, pattern.LastError, pattern.Count)
	mem := models.SkillRuntimeMemory{
		Type:       "improvement_hint",
		Key:        "improve_" + normalized,
		Content:    content,
		Confidence: 0.5,
		Source:     "bootstrap",
		Status:     "pending",
		LastUsedAt: time.Now(),
	}
	var existing models.SkillRuntimeMemory
	if err := s.db.Where("type = ? AND key = ?", mem.Type, mem.Key).First(&existing).Error; err == nil {
		s.db.Model(&existing).Updates(map[string]interface{}{"content": content, "confidence": 0.5, "status": "pending", "updated_at": time.Now(), "last_used_at": time.Now()})
		return
	}
	s.db.Create(&mem)
}

func (s *SelfLearning) AutoPublishSkill(skillID string) error {
	var skill models.Skill
	if err := s.db.First(&skill, "id = ?", skillID).Error; err != nil {
		return err
	}
	return s.db.Model(&skill).Update("enabled", true).Error
}

func (s *SelfLearning) PeriodicBootstrap(skillID string) {
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			s.db.Where("type = ? AND source = ? AND status = ?", "sql_pattern", "runtime", "active").Where("confidence >= ?", 0.7).Find(&[]models.SkillRuntimeMemory{})
		}
	}()
}

func normalizeForLearning(query string) string {
	query = strings.ReplaceAll(query, " ", "")
	query = strings.ReplaceAll(query, "？", "?")
	if len([]rune(query)) > 30 {
		query = string([]rune(query)[:30])
	}
	return query
}

func (e *AgentEngine) RecordFailure(query, errorMsg string) {
	if e.selfLearning == nil {
		return
	}
	e.selfLearning.RecordFailure(query, errorMsg)
}

func (e *AgentEngine) RecordFeedback(query, correction string) {
	if e.selfLearning == nil {
		return
	}
	e.selfLearning.RecordFeedback(query, correction)
}

func (e *AgentEngine) AutoPublishSkill(skillID string) error {
	if e.selfLearning == nil {
		return nil
	}
	return e.selfLearning.AutoPublishSkill(skillID)
}

func (e *AgentEngine) GetImprovementHints() []map[string]interface{} {
	if e.selfLearning == nil {
		return nil
	}
	var mems []models.SkillRuntimeMemory
	e.db.Where("type = ? AND source = ?", "improvement_hint", "bootstrap").Order("created_at DESC").Limit(20).Find(&mems)
	var hints []map[string]interface{}
	for _, m := range mems {
		hints = append(hints, map[string]interface{}{"key": m.Key, "content": m.Content, "confidence": m.Confidence, "status": m.Status})
	}
	return hints
}

func (e *AgentEngine) GetFeedbackRecords() []map[string]interface{} {
	if e.selfLearning == nil {
		return nil
	}
	var mems []models.SkillRuntimeMemory
	e.db.Where("type = ? AND source = ?", "user_feedback", "feedback").Order("created_at DESC").Limit(20).Find(&mems)
	var records []map[string]interface{}
	for _, m := range mems {
		records = append(records, map[string]interface{}{"key": m.Key, "content": m.Content, "confidence": m.Confidence})
	}
	return records
}

var _ = json.Marshal
