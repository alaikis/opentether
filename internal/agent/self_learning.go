package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

type FailureType string

const (
	FailureTypeSyntax     FailureType = "syntax"
	FailureTypeSemantic   FailureType = "semantic"
	FailureTypeTimeout    FailureType = "timeout"
	FailureTypePermission FailureType = "permission"
	FailureTypeUnknown    FailureType = "unknown"
)

type SelfLearning struct {
	db       *gorm.DB
	mu       sync.Mutex
	patterns map[string]failurePattern

	llmQualityTracker *LLMQualityTracker
}

type failurePattern struct {
	Query           string
	FailureType     FailureType
	Count           int
	SuccessCount    int
	LastError       string
	FirstSeen       time.Time
	LastSeen        time.Time
	Confidence      float64
	SuggestionGiven bool
}

type LLMQualityTracker struct {
	mu           sync.Mutex
	recentLatencies []int64
	recentErrors   int
	recentCalls    int
	windowSize     int
	baselineLatency int64
	alertThreshold float64
}

func NewLLMQualityTracker(windowSize int, baselineLatencyMs int64) *LLMQualityTracker {
	return &LLMQualityTracker{
		windowSize:       windowSize,
		baselineLatency:  baselineLatencyMs,
		alertThreshold:   2.0,
		recentLatencies:  make([]int64, 0, windowSize),
	}
}

func (t *LLMQualityTracker) RecordCall(latencyMs int64, success bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.recentCalls++
	if !success {
		t.recentErrors++
	}

	t.recentLatencies = append(t.recentLatencies, latencyMs)
	if len(t.recentLatencies) > t.windowSize {
		t.recentLatencies = t.recentLatencies[1:]
	}
}

func (t *LLMQualityTracker) GetErrorRate() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.recentCalls == 0 {
		return 0
	}
	return float64(t.recentErrors) / float64(t.recentCalls)
}

func (t *LLMQualityTracker) GetAvgLatency() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.recentLatencies) == 0 {
		return 0
	}
	var sum int64
	for _, l := range t.recentLatencies {
		sum += l
	}
	return sum / int64(len(t.recentLatencies))
}

func (t *LLMQualityTracker) IsDegraded() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	avgLatency := t.GetAvgLatencyUnlocked()
	if t.baselineLatency > 0 && avgLatency > int64(float64(t.baselineLatency)*t.alertThreshold) {
		return true
	}
	errorRate := float64(t.recentErrors) / float64(max(1, t.recentCalls))
	if errorRate > 0.2 {
		return true
	}
	return false
}

func (t *LLMQualityTracker) GetAvgLatencyUnlocked() int64 {
	if len(t.recentLatencies) == 0 {
		return 0
	}
	var sum int64
	for _, l := range t.recentLatencies {
		sum += l
	}
	return sum / int64(len(t.recentLatencies))
}

func (t *LLMQualityTracker) GetStatus() map[string]interface{} {
	t.mu.Lock()
	defer t.mu.Unlock()
	return map[string]interface{}{
		"recent_calls":    t.recentCalls,
		"recent_errors":   t.recentErrors,
		"error_rate":      float64(t.recentErrors) / float64(max(1, t.recentCalls)),
		"avg_latency_ms":  t.GetAvgLatencyUnlocked(),
		"baseline_ms":     t.baselineLatency,
		"is_degraded":     t.IsDegraded(),
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func classifyFailureType(errorMsg string) FailureType {
	errorLower := strings.ToLower(errorMsg)

	syntaxPatterns := []string{"syntax error", "parse error", "invalid syntax", "sql syntax", "语法错误", "解析错误"}
	for _, p := range syntaxPatterns {
		if strings.Contains(errorLower, strings.ToLower(p)) {
			return FailureTypeSyntax
		}
	}

	timeoutPatterns := []string{"timeout", "超时", "deadline", "cancelled", "context deadline"}
	for _, p := range timeoutPatterns {
		if strings.Contains(errorLower, p) {
			return FailureTypeTimeout
		}
	}

	permissionPatterns := []string{"permission", "权限", "forbidden", "unauthorized", "access denied", "拒绝访问"}
	for _, p := range permissionPatterns {
		if strings.Contains(errorLower, p) {
			return FailureTypePermission
		}
	}

	semanticPatterns := []string{"not found", "no such", "不存在", "无效的", "invalid", "unknown", "无法", "无法识别"}
	for _, p := range semanticPatterns {
		if strings.Contains(errorLower, p) {
			return FailureTypeSemantic
		}
	}

	return FailureTypeUnknown
}

func NewSelfLearning(db *gorm.DB) *SelfLearning {
	return &SelfLearning{
		db:                db,
		patterns:          make(map[string]failurePattern),
		llmQualityTracker: NewLLMQualityTracker(100, 5000),
	}
}

func (s *SelfLearning) RecordFailure(query, errorMsg string) {
	normalized := normalizeForLearning(query)
	failureType := classifyFailureType(errorMsg)

	s.mu.Lock()
	pattern := s.patterns[normalized]
	pattern.Count++
	pattern.LastError = errorMsg
	pattern.FailureType = failureType
	pattern.LastSeen = time.Now()

	if pattern.FirstSeen.IsZero() {
		pattern.FirstSeen = time.Now()
		pattern.Query = query
		pattern.Confidence = 0.1
	}

	s.patterns[normalized] = pattern
	s.mu.Unlock()

	s.generateImprovementTemplateRealtime(normalized, pattern, failureType)
}

func (s *SelfLearning) RecordSuccess(query string) {
	normalized := normalizeForLearning(query)
	s.mu.Lock()
	defer s.mu.Unlock()
	if pattern, ok := s.patterns[normalized]; ok {
		pattern.SuccessCount++
		if pattern.Confidence < 0.95 {
			pattern.Confidence = min(0.95, pattern.Confidence+0.05)
		}
		if pattern.Confidence >= 0.7 && !pattern.SuggestionGiven {
			pattern.SuggestionGiven = true
			go s.markAsResolved(normalized, pattern)
		}
	}
}

func (s *SelfLearning) generateImprovementTemplateRealtime(normalized string, pattern failurePattern, failureType FailureType) {
	var hint string
	switch failureType {
	case FailureTypeSyntax:
		hint = fmt.Sprintf("查询语法问题: %s | 错误: %s | 尝试次数: %d",
			pattern.Query, pattern.LastError, pattern.Count)
	case FailureTypeSemantic:
		hint = fmt.Sprintf("查询语义问题: %s | 错误: %s | 尝试次数: %d",
			pattern.Query, pattern.LastError, pattern.Count)
	case FailureTypeTimeout:
		hint = fmt.Sprintf("查询超时: %s | 错误: %s | 尝试次数: %d",
			pattern.Query, pattern.LastError, pattern.Count)
	case FailureTypePermission:
		hint = fmt.Sprintf("权限问题: %s | 错误: %s | 尝试次数: %d",
			pattern.Query, pattern.LastError, pattern.Count)
	default:
		hint = fmt.Sprintf("查询失败: %s | 错误: %s | 尝试次数: %d",
			pattern.Query, pattern.LastError, pattern.Count)
	}

	mem := models.SkillRuntimeMemory{
		Type:       "improvement_hint",
		Key:        "improve_" + normalized,
		Content:    hint,
		Confidence: pattern.Confidence,
		Source:     "bootstrap",
		Status:     "pending",
		LastUsedAt: time.Now(),
	}

	s.mu.Lock()
	_, patternExists := s.patterns[normalized]
	s.mu.Unlock()

	if patternExists {
		var existingMem models.SkillRuntimeMemory
		if err := s.db.Where("type = ? AND key = ?", mem.Type, mem.Key).First(&existingMem).Error; err == nil {
			newConfidence := min(0.9, existingMem.Confidence+0.1)
			s.db.Model(&existingMem).Updates(map[string]interface{}{
				"content":      hint,
				"confidence":   newConfidence,
				"updated_at":   time.Now(),
				"last_used_at": time.Now(),
			})
		} else {
			mem.Confidence = pattern.Confidence
			s.db.Create(&mem)
		}
		s.mu.Lock()
		if p, ok := s.patterns[normalized]; ok {
			p.SuggestionGiven = true
		}
		s.mu.Unlock()
		return
	}

	var existingMem models.SkillRuntimeMemory
	if err := s.db.Where("type = ? AND key = ?", mem.Type, mem.Key).First(&existingMem).Error; err == nil {
		s.db.Model(&existingMem).Updates(map[string]interface{}{
			"content":      hint,
			"confidence":   pattern.Confidence,
			"updated_at":   time.Now(),
			"last_used_at": time.Now(),
		})
	} else {
		s.db.Create(&mem)
	}
}

func (s *SelfLearning) markAsResolved(normalized string, pattern failurePattern) {
	if s == nil || s.db == nil {
		return
	}
	mem := models.SkillRuntimeMemory{
		Type:       "resolved_pattern",
		Key:        "resolved_" + normalized,
		Content:    fmt.Sprintf("已解决的问题: %s | 成功次数: %d", pattern.Query, pattern.SuccessCount),
		Confidence: pattern.Confidence,
		Source:     "runtime",
		Status:     "active",
		LastUsedAt: time.Now(),
	}

	var existing models.SkillRuntimeMemory
	if err := s.db.Where("type = ? AND key = ?", mem.Type, mem.Key).First(&existing).Error; err != nil {
		s.db.Create(&mem)
	}
	log.Printf("[SelfLearning] Pattern resolved: %s (confidence: %.2f)", normalized, pattern.Confidence)
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
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

type BacktestResult struct {
	TotalPatterns    int     `json:"total_patterns"`
	TestedPatterns   int     `json:"tested_patterns"`
	ResolvedPatterns int     `json:"resolved_patterns"`
	ImprovementRate  float64 `json:"improvement_rate"`
}

func (s *SelfLearning) Backtest(query string) *BacktestResult {
	if s == nil || s.db == nil {
		return &BacktestResult{}
	}
	s.mu.Lock()
	patterns := make([]failurePattern, 0, len(s.patterns))
	for _, p := range s.patterns {
		patterns = append(patterns, p)
	}
	s.mu.Unlock()

	result := &BacktestResult{TotalPatterns: len(patterns)}
	if len(patterns) == 0 {
		return result
	}

	resolved := 0
	for _, pattern := range patterns {
		normalized := normalizeForLearning(pattern.Query)
		var feedbackCount int64
		s.db.Model(&models.SkillRuntimeMemory{}).Where("type = ? AND key = ?", "user_feedback", "feedback_"+normalized).Count(&feedbackCount)
		if feedbackCount > 0 {
			resolved++
		}
	}

	result.TestedPatterns = len(patterns)
	result.ResolvedPatterns = resolved
	if result.TestedPatterns > 0 {
		result.ImprovementRate = float64(resolved) / float64(result.TestedPatterns)
	}
	return result
}

func (e *AgentEngine) Backtest(query string) *BacktestResult {
	if e.selfLearning == nil {
		return &BacktestResult{}
	}
	return e.selfLearning.Backtest(query)
}

var _ = json.Marshal

type promptCache struct {
	mu    sync.Mutex
	items map[string]promptCacheEntry
}

type promptCacheEntry struct {
	Prompt    string
	ExpiresAt time.Time
	Hits      int
	Successes int
}

var promptStore = &promptCache{items: make(map[string]promptCacheEntry)}

func (e *AgentEngine) getPromptCache(key string) (string, bool) {
	promptStore.mu.Lock()
	defer promptStore.mu.Unlock()
	if entry, ok := promptStore.items[key]; ok && time.Now().Before(entry.ExpiresAt) {
		entry.Hits++
		promptStore.items[key] = entry
		return entry.Prompt, true
	}
	return "", false
}

func (e *AgentEngine) setPromptCache(key, prompt string, ttl time.Duration) {
	promptStore.mu.Lock()
	defer promptStore.mu.Unlock()
	promptStore.items[key] = promptCacheEntry{Prompt: prompt, ExpiresAt: time.Now().Add(ttl), Hits: 1}
}

func (e *AgentEngine) recordPromptSuccess(key string) {
	promptStore.mu.Lock()
	defer promptStore.mu.Unlock()
	if entry, ok := promptStore.items[key]; ok {
		entry.Successes++
		entry.Hits++
		promptStore.items[key] = entry
	}
}

func (e *AgentEngine) getPromptVersion(key string) string {
	promptStore.mu.Lock()
	defer promptStore.mu.Unlock()
	if entry, ok := promptStore.items[key]; ok && entry.Hits >= 10 {
		rate := float64(entry.Successes) / float64(entry.Hits)
		if rate > 0.8 {
			return "v2"
		}
	}
	return "v1"
}

func (e *AgentEngine) GetAdaptiveCacheTTL() int { return 120 }
func (e *AgentEngine) GetAutoThreshold() float64 { return 2.0 }

func (e *AgentEngine) RecordLLMCall(latencyMs int64, success bool) {
	if e.selfLearning == nil || e.selfLearning.llmQualityTracker == nil {
		return
	}
	e.selfLearning.llmQualityTracker.RecordCall(latencyMs, success)

	if e.selfLearning.llmQualityTracker.IsDegraded() {
		log.Printf("[SelfLearning] LLM quality degraded detected! Status: %+v",
			e.selfLearning.llmQualityTracker.GetStatus())
	}
}

func (e *AgentEngine) GetLLMQualityStatus() map[string]interface{} {
	if e.selfLearning == nil || e.selfLearning.llmQualityTracker == nil {
		return nil
	}
	return e.selfLearning.llmQualityTracker.GetStatus()
}

func (e *AgentEngine) RecordQuerySuccess(query string) {
	if e.selfLearning == nil {
		return
	}
	e.selfLearning.RecordSuccess(query)
}

func (s *SelfLearning) GetFailureStats() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	total := len(s.patterns)
	var byType = make(map[string]int)
	var avgConfidence float64
	var withSuggestion int

	for _, p := range s.patterns {
		byType[string(p.FailureType)]++
		avgConfidence += p.Confidence
		if p.SuggestionGiven {
			withSuggestion++
		}
	}

	if total > 0 {
		avgConfidence /= float64(total)
	}

	return map[string]interface{}{
		"total_patterns":     total,
		"by_type":            byType,
		"avg_confidence":     avgConfidence,
		"patterns_with_hint": withSuggestion,
	}
}
