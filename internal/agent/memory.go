package agent

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/embedding"
	"github.com/alaikis/opentether/internal/models"
	"github.com/alaikis/opentether/internal/vectorstore"
	"gorm.io/gorm"
)

// ============================================
// Letta-inspired Memory System
//
// Core Memory:  用户/公司 Soul (UserProfile / CompanyProfile)
// Archival Memory: 长期记忆 + 语义召回 (UserMemory / GroupMemory)
// Conversation Memory: 对话窗口管理 (内置在 ExecuteLoop)
// ============================================

// ── Aliases ──────────────────────────────────
type (
	UserMemory     = models.UserMemory
	GroupMemory    = models.GroupMemory
	UserProfile    = models.UserProfile
	CompanyProfile = models.CompanyProfile
)

// ── LettaMemory 统一记忆管理器 ──────────────
type LettaMemory struct {
	db *gorm.DB
}

func NewLettaMemory(db *gorm.DB) *LettaMemory {
	return &LettaMemory{db: db}
}

// ════════════════════════════════════════════
// Core Memory (Soul / Profile)
// ════════════════════════════════════════════

// GetUserSoul 获取用户 Soul（含默认值），如果仍为默认值则自动从记忆中进化。
func (m *LettaMemory) GetUserSoul(userID string) *UserProfile {
	var p UserProfile
	if err := m.db.Where("user_id = ?", userID).First(&p).Error; err != nil {
		return &UserProfile{
			Persona:            "你是 OpenTether AI 助手，专业且友好。",
			Human:              "用户是公司员工。",
			LanguagePreference: "zh-CN",
		}
	}
	// 动态增强：如果还是默认值，从记忆中进化
	if strings.Contains(p.Persona, "你是 OpenTether AI 助手") && p.Human == "用户是公司员工。" {
		m.evolveSoulFromMemories(userID, &p)
	}
	return &p
}

// evolveSoulFromMemories 从用户的累积记忆中自动进化 Persona 和 Human 描述。
func (m *LettaMemory) evolveSoulFromMemories(userID string, p *UserProfile) {
	if m == nil || m.db == nil {
		return
	}

	var totalQueries int64
	m.db.Model(&UserMemory{}).Where("user_id = ? AND type = ?", userID, "query_pattern").Count(&totalQueries)
	if totalQueries < 5 {
		return
	}

	// 统计常用指标
	type metricFreq struct {
		Key   string
		Count int
	}
	var metricCounts []metricFreq
	m.db.Model(&UserMemory{}).
		Select("key, COUNT(*) as count").
		Where("user_id = ? AND type = ?", userID, "preferred_metric").
		Group("key").Order("count DESC").Limit(5).
		Find(&metricCounts)

	// 统计语言
	cnCount := 0
	enCount := 0
	var contents []string
	m.db.Model(&UserMemory{}).
		Where("user_id = ? AND type = ?", userID, "query_pattern").
		Pluck("content", &contents)
	for _, c := range contents {
		if inferLanguage("", c) == "zh-CN" {
			cnCount++
		} else {
			enCount++
		}
	}

	// 生成新的 Persona
	persona := "你是 OpenTether AI 助手"
	if len(metricCounts) > 0 {
		var names []string
		for _, m := range metricCounts {
			names = append(names, m.Key)
		}
		persona += "，擅长" + strings.Join(names, "、") + "等数据分析"
	}
	persona += "。"
	if enCount > cnCount {
		persona += "优先用英文回复。"
		p.LanguagePreference = "en"
	} else if cnCount > 0 {
		p.LanguagePreference = "zh-CN"
	}

	// 生成新的 Human
	human := "用户是公司员工"
	if len(metricCounts) > 0 {
		n := len(metricCounts)
		if n > 3 {
			n = 3
		}
		var topNames []string
		for i := 0; i < n; i++ {
			topNames = append(topNames, metricCounts[i].Key)
		}
		human += "，常查询" + strings.Join(topNames, "、")
	}
	human += fmt.Sprintf("。累计查询 %d 次。", totalQueries)

	p.Persona = persona
	p.Human = human
	_ = m.db.Save(p).Error
	log.Printf("[Memory] 用户 %s Soul 自动进化: %s | %s", userID, persona, human)

	// 老化：清理 90 天前的低优先级记忆
	cutoff := time.Now().AddDate(0, 0, -90)
	m.db.Where("user_id = ? AND updated_at < ? AND priority < 3", userID, cutoff).Delete(&UserMemory{})
}

// UpsertUserSoul 创建或更新用户 Soul
func (m *LettaMemory) UpsertUserSoul(userID string, persona, human, preferences string) error {
	var p UserProfile
	err := m.db.Where("user_id = ?", userID).First(&p).Error
	if err == gorm.ErrRecordNotFound {
		p = UserProfile{
			UserID:             userID,
			Persona:            persona,
			Human:              human,
			Preferences:        preferences,
			LanguagePreference: "zh-CN",
		}
		return m.db.Create(&p).Error
	}
	if err != nil {
		return err
	}
	p.Persona = persona
	p.Human = human
	p.Preferences = preferences
	return m.db.Save(&p).Error
}

// GetCompanySoul 获取公司 Soul
func (m *LettaMemory) GetCompanySoul() *CompanyProfile {
	var p CompanyProfile
	if err := m.db.First(&p).Error; err != nil {
		return nil
	}
	return &p
}

// UpsertCompanySoul 创建或更新公司 Soul
func (m *LettaMemory) UpsertCompanySoul(name, persona, brandTone, industry string) error {
	var p CompanyProfile
	err := m.db.First(&p).Error
	if err == gorm.ErrRecordNotFound {
		p = CompanyProfile{
			Name:      name,
			Persona:   persona,
			BrandTone: brandTone,
			Industry:  industry,
		}
		return m.db.Create(&p).Error
	}
	if err != nil {
		return err
	}
	p.Name = name
	p.Persona = persona
	p.BrandTone = brandTone
	p.Industry = industry
	return m.db.Save(&p).Error
}

// ════════════════════════════════════════════
// Archival Memory (长期记忆 + 召回)
// ════════════════════════════════════════════

// SaveArchivalMemory 用户存档记忆
func (m *LettaMemory) SaveArchivalMemory(userID, memType, key, content string, priority int) error {
	return m.UpsertUserMemory(userID, memType, key, content, priority)
}

// UpsertUserMemory 创建或更新用户长期记忆
func (m *LettaMemory) UpsertUserMemory(userID, memType, key, content string, priority int) error {
	var existing UserMemory
	err := m.db.Where("user_id = ? AND type = ? AND key = ?", userID, memType, key).First(&existing).Error
	if err == nil {
		existing.Content = content
		existing.Priority = priority
		return m.db.Save(&existing).Error
	}
	mem := &UserMemory{
		UserID:   userID,
		Type:     memType,
		Key:      key,
		Content:  content,
		Priority: priority,
	}
	return m.db.Create(mem).Error
}

// RecallUserMemory 按用户召回记忆（关键词 + 时间衰减）
func (m *LettaMemory) RecallUserMemory(userID, query string, limit int) ([]UserMemory, error) {
	if query == "" {
		return m.GetRecentUserMemories(userID, limit)
	}

	keywords := strings.Fields(query)
	var memories []UserMemory

	for _, kw := range keywords {
		var matched []UserMemory
		pattern := "%" + kw + "%"
		m.db.Where("user_id = ? AND (content LIKE ? OR key LIKE ?)", userID, pattern, pattern).
			Order("priority DESC, updated_at DESC").
			Limit(5).Find(&matched)
		memories = append(memories, matched...)
	}

	// 去重
	seen := make(map[string]bool)
	var result []UserMemory
	for _, m := range memories {
		if !seen[m.ID] {
			seen[m.ID] = true
			result = append(result, m)
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// GetRecentUserMemories 获取用户最近记忆
func (m *LettaMemory) GetRecentUserMemories(userID string, limit int) ([]UserMemory, error) {
	var memories []UserMemory
	query := m.db.Where("user_id = ?", userID).Order("updated_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&memories).Error
	return memories, err
}

// RecallUserMemorySemantic 使用向量语义相似度召回用户长期记忆。
// 当前实现复用内置 TF-IDF + memory vectorstore，按请求临时构建小索引，避免引入额外外部依赖。
func (m *LettaMemory) RecallUserMemorySemantic(userID, query string, limit int) ([]UserMemory, error) {
	if strings.TrimSpace(query) == "" {
		return m.GetRecentUserMemories(userID, limit)
	}

	var memories []UserMemory
	if err := m.db.Where("user_id = ?", userID).
		Order("priority DESC, updated_at DESC").
		Limit(200).
		Find(&memories).Error; err != nil {
		return nil, err
	}
	if len(memories) == 0 {
		return nil, nil
	}

	docs := make([]string, 0, len(memories)+1)
	docs = append(docs, query)
	for _, mem := range memories {
		docs = append(docs, memorySearchText(mem.Type, mem.Key, mem.Content))
	}

	embedder, err := embedding.Create("tfidf", map[string]interface{}{"corpus": docs})
	if err != nil {
		return nil, err
	}
	store, err := vectorstore.CreateStore("memory", nil)
	if err != nil {
		return nil, err
	}

	memoryByID := make(map[string]UserMemory, len(memories))
	for _, mem := range memories {
		id := mem.ID
		vec, err := embedder.Embed(memorySearchText(mem.Type, mem.Key, mem.Content))
		if err != nil {
			continue
		}
		_ = store.Index(id, mem.Key, vec)
		memoryByID[id] = mem
	}
	queryVec, err := embedder.Embed(query)
	if err != nil {
		return nil, err
	}
	matches, err := store.Search(queryVec, limit, 0.08)
	if err != nil {
		return nil, err
	}

	result := make([]UserMemory, 0, len(matches))
	seen := map[string]bool{}
	for _, match := range matches {
		if mem, ok := memoryByID[match.SkillID]; ok && !seen[mem.ID] {
			result = append(result, mem)
			seen[mem.ID] = true
		}
	}
	return result, nil
}

// SaveGroupMemory 保存组共享记忆
func (m *LettaMemory) SaveGroupMemory(groupID, memType, key, content string, priority int) error {
	var existing GroupMemory
	err := m.db.Where("group_id = ? AND type = ? AND key = ?", groupID, memType, key).First(&existing).Error
	if err == nil {
		existing.Content = content
		existing.Priority = priority
		return m.db.Save(&existing).Error
	}
	mem := &GroupMemory{
		GroupID:  groupID,
		Type:     memType,
		Key:      key,
		Content:  content,
		Priority: priority,
	}
	return m.db.Create(mem).Error
}

// RecallGroupMemories 召回组共享记忆
func (m *LettaMemory) RecallGroupMemories(groupIDs []string, query string, limit int) ([]GroupMemory, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}

	var memories []GroupMemory
	if query != "" {
		keywords := strings.Fields(query)
		for _, kw := range keywords {
			pattern := "%" + kw + "%"
			var matched []GroupMemory
			m.db.Where("group_id IN ? AND (content LIKE ? OR key LIKE ?)", groupIDs, pattern, pattern).
				Order("priority DESC, updated_at DESC").
				Limit(5).Find(&matched)
			memories = append(memories, matched...)
		}
	} else {
		m.db.Where("group_id IN ?", groupIDs).Order("priority DESC").
			Limit(limit).Find(&memories)
	}

	seen := make(map[string]bool)
	var result []GroupMemory
	for _, m := range memories {
		if !seen[m.ID] {
			seen[m.ID] = true
			result = append(result, m)
		}
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// RecallGroupMemoriesSemantic 使用向量语义相似度召回团队/部门共享记忆。
func (m *LettaMemory) RecallGroupMemoriesSemantic(groupIDs []string, query string, limit int) ([]GroupMemory, error) {
	if len(groupIDs) == 0 || strings.TrimSpace(query) == "" {
		return m.RecallGroupMemories(groupIDs, query, limit)
	}

	var memories []GroupMemory
	if err := m.db.Where("group_id IN ?", groupIDs).
		Order("priority DESC, updated_at DESC").
		Limit(200).
		Find(&memories).Error; err != nil {
		return nil, err
	}
	if len(memories) == 0 {
		return nil, nil
	}

	docs := make([]string, 0, len(memories)+1)
	docs = append(docs, query)
	for _, mem := range memories {
		docs = append(docs, memorySearchText(mem.Type, mem.Key, mem.Content))
	}

	embedder, err := embedding.Create("tfidf", map[string]interface{}{"corpus": docs})
	if err != nil {
		return nil, err
	}
	store, err := vectorstore.CreateStore("memory", nil)
	if err != nil {
		return nil, err
	}

	memoryByID := make(map[string]GroupMemory, len(memories))
	for _, mem := range memories {
		id := mem.ID
		vec, err := embedder.Embed(memorySearchText(mem.Type, mem.Key, mem.Content))
		if err != nil {
			continue
		}
		_ = store.Index(id, mem.Key, vec)
		memoryByID[id] = mem
	}
	queryVec, err := embedder.Embed(query)
	if err != nil {
		return nil, err
	}
	matches, err := store.Search(queryVec, limit, 0.08)
	if err != nil {
		return nil, err
	}

	result := make([]GroupMemory, 0, len(matches))
	seen := map[string]bool{}
	for _, match := range matches {
		if mem, ok := memoryByID[match.SkillID]; ok && !seen[mem.ID] {
			result = append(result, mem)
			seen[mem.ID] = true
		}
	}
	return result, nil
}

func memorySearchText(memType, key, content string) string {
	return strings.TrimSpace(memType + " " + key + " " + content)
}

// ════════════════════════════════════════════
// Conversation Memory (对话摘要归档)
// ════════════════════════════════════════════

// SaveConversationSummary 对话结束后摘要入库
func (m *LettaMemory) SaveConversationSummary(userID, userQuery, assistantReply string, groupIDs []string) {
	summary := m.buildSummary(userQuery, assistantReply)
	timestamp := time.Now().Format("2006-01-02 15:04")
	key := fmt.Sprintf("对话 %s", timestamp)
	m.UpsertUserMemory(userID, "conversation", key, summary, 1)

	for _, gid := range groupIDs {
		topic := m.extractTopic(userQuery)
		if topic != "" {
			m.SaveGroupMemory(gid, "topic", topic, summary, 5)
		}
	}
}

func (m *LettaMemory) buildSummary(query, reply string) string {
	q := truncateStr(query, 200)
	r := truncateStr(reply, 300)
	return fmt.Sprintf("用户问题: %s\nAI 回复: %s", q, r)
}

func (m *LettaMemory) extractTopic(query string) string {
	return ""
}

// ═══════════════════════════════════════════=
// Enhanced Soul Evolution: 隐式反馈 + 多维度 Persona
// ═══════════════════════════════════════════=

type EnhancedSoul struct {
	UserID             string
	ProfessionalLevel  string
	ReplyStyle         string
	PreferredMetrics   []string
	Confidence         map[string]float64
	AvgSessionLength   int
	FollowUpRate       float64
	QueryComplexity    float64
	LockedDimensions   map[string]bool
	LastEvolvedAt      time.Time
}

const (
	ProfessionalBeginner    = "beginner"
	ProfessionalIntermediate = "intermediate"
	ProfessionalExpert      = "expert"

	ReplyStyleConcise = "concise"
	ReplyStyleDetailed = "detailed"
	ReplyStyleTechnical = "technical"
)

func (m *LettaMemory) GetEnhancedSoul(userID string) *EnhancedSoul {
	var existing EnhancedSoulData
	if err := m.db.Where("user_id = ?", userID).First(&existing).Error; err != nil {
		return &EnhancedSoul{
			UserID:           userID,
			PreferredMetrics: []string{},
			Confidence:       make(map[string]float64),
			LockedDimensions: make(map[string]bool),
		}
	}

	locked := make(map[string]bool)
	if existing.LockedDimensions != "" {
		for _, d := range strings.Split(existing.LockedDimensions, ",") {
			if d = strings.TrimSpace(d); d != "" {
				locked[d] = true
			}
		}
	}

	metrics := []string{}
	if existing.PreferredMetrics != "" {
		metrics = strings.Split(existing.PreferredMetrics, ",")
	}

	return &EnhancedSoul{
		UserID:            userID,
		ProfessionalLevel: existing.ProfessionalLevel,
		ReplyStyle:        existing.ReplyStyle,
		PreferredMetrics:  metrics,
		Confidence:        map[string]float64{"professional": existing.ProfessionalConfidence, "style": existing.StyleConfidence},
		AvgSessionLength:  existing.AvgSessionLength,
		FollowUpRate:      existing.FollowUpRate,
		QueryComplexity:   existing.QueryComplexity,
		LockedDimensions:  locked,
		LastEvolvedAt:     existing.LastEvolvedAt,
	}
}

func (m *LettaMemory) RecordImplicitFeedback(userID string, sessionLength int, isFollowUp bool, queryComplexity float64) {
	if m == nil || m.db == nil {
		return
	}

	var existing EnhancedSoulData
	err := m.db.Where("user_id = ?", userID).First(&existing).Error
	now := time.Now()

	if err == gorm.ErrRecordNotFound {
		existing = EnhancedSoulData{
			UserID:           userID,
			TotalSessions:    1,
			TotalQueries:     sessionLength,
			FollowUpCount:    0,
			ComplexitySum:    queryComplexity,
			AvgSessionLength: sessionLength,
			FollowUpRate:     0,
			QueryComplexity:  queryComplexity,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if isFollowUp {
			existing.FollowUpCount = 1
		}
		m.db.Create(&existing)
	} else {
		existing.TotalSessions++
		existing.TotalQueries += sessionLength
		existing.ComplexitySum += queryComplexity
		existing.AvgSessionLength = existing.TotalQueries / existing.TotalSessions

		if isFollowUp {
			existing.FollowUpCount++
		}
		existing.FollowUpRate = float64(existing.FollowUpCount) / float64(existing.TotalQueries)
		existing.QueryComplexity = existing.ComplexitySum / float64(existing.TotalQueries)
		existing.UpdatedAt = now

		m.db.Save(&existing)
	}

	m.triggerEnhancedEvolution(userID)
}

func (m *LettaMemory) triggerEnhancedEvolution(userID string) {
	soul := m.GetEnhancedSoul(userID)

	if soul.TotalQueries() >= 5 {
		m.evolveAdvancedSoul(userID, soul)
	}
}

func (s *EnhancedSoul) TotalQueries() int {
	return s.AvgSessionLength * 10
}

func (m *LettaMemory) evolveAdvancedSoul(userID string, soul *EnhancedSoul) {
	if m == nil || m.db == nil {
		return
	}

	var existing EnhancedSoulData
	if err := m.db.Where("user_id = ?", userID).First(&existing).Error; err != nil {
		return
	}

	changes := make(map[string]interface{})

	if !soul.LockedDimensions["professional"] {
		newLevel := calculateProfessionalLevel(existing.QueryComplexity, existing.TotalQueries)
		if newLevel != existing.ProfessionalLevel {
			changes["professional_level"] = newLevel
			changes["professional_confidence"] = min(0.95, existing.ProfessionalConfidence+0.15)
		}
	}

	if !soul.LockedDimensions["style"] {
		newStyle := calculateReplyStyle(existing.FollowUpRate, existing.AvgSessionLength)
		if newStyle != existing.ReplyStyle {
			changes["reply_style"] = newStyle
			changes["style_confidence"] = min(0.95, existing.StyleConfidence+0.15)
		}
	}

	if len(changes) > 0 {
		changes["updated_at"] = time.Now()
		m.db.Model(&existing).Updates(changes)
		log.Printf("[Memory] Enhanced soul evolved for user %s: %+v", userID, changes)
	}
}

func calculateProfessionalLevel(complexity float64, totalQueries int) string {
	if totalQueries < 5 {
		return ProfessionalBeginner
	}

	if complexity > 0.7 || totalQueries >= 20 {
		return ProfessionalExpert
	} else if complexity > 0.4 || totalQueries >= 10 {
		return ProfessionalIntermediate
	}

	return ProfessionalBeginner
}

func calculateReplyStyle(followUpRate float64, avgSessionLength int) string {
	if avgSessionLength > 5 && followUpRate < 0.2 {
		return ReplyStyleConcise
	} else if avgSessionLength <= 2 && followUpRate > 0.5 {
		return ReplyStyleDetailed
	} else if avgSessionLength > 8 {
		return ReplyStyleTechnical
	}

	return ReplyStyleDetailed
}

func (m *LettaMemory) LockSoulDimension(userID, dimension string, locked bool) error {
	var existing EnhancedSoulData
	if err := m.db.Where("user_id = ?", userID).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			existing = EnhancedSoulData{UserID: userID}
			m.db.Create(&existing)
		} else {
			return err
		}
	}

	lockedDims := make(map[string]bool)
	if existing.LockedDimensions != "" {
		for _, d := range strings.Split(existing.LockedDimensions, ",") {
			if d = strings.TrimSpace(d); d != "" {
				lockedDims[d] = true
			}
		}
	}

	if locked {
		lockedDims[dimension] = true
	} else {
		delete(lockedDims, dimension)
	}

	var dims []string
	for d := range lockedDims {
		dims = append(dims, d)
	}

	return m.db.Model(&existing).Update("locked_dimensions", strings.Join(dims, ",")).Error
}

func (m *LettaMemory) GetSoulEvolutionHistory(userID string) []EvolutionEvent {
	var events []EnhancedSoulEvolution
	m.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(20).Find(&events)

	result := make([]EvolutionEvent, 0, len(events))
	for _, e := range events {
		result = append(result, EvolutionEvent{
			ID:         fmt.Sprintf("%d", e.ID),
			UserID:     e.UserID,
			Dimension:  e.Dimension,
			OldValue:   e.OldValue,
			NewValue:   e.NewValue,
			Trigger:    e.Trigger,
			Confidence: e.Confidence,
			CreatedAt:  e.CreatedAt,
		})
	}
	return result
}

type EnhancedSoulData struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	UserID               string    `gorm:"uniqueIndex" json:"user_id"`
	ProfessionalLevel    string    `json:"professional_level"`
	ProfessionalConfidence float64 `json:"professional_confidence"`
	ReplyStyle           string    `json:"reply_style"`
	StyleConfidence      float64   `json:"style_confidence"`
	PreferredMetrics     string    `json:"preferred_metrics"`
	TotalSessions        int       `json:"total_sessions"`
	TotalQueries         int       `json:"total_queries"`
	FollowUpCount        int       `json:"follow_up_count"`
	AvgSessionLength     int       `json:"avg_session_length"`
	FollowUpRate         float64   `json:"follow_up_rate"`
	QueryComplexity      float64   `json:"query_complexity"`
	ComplexitySum        float64   `json:"complexity_sum"`
	LockedDimensions     string    `json:"locked_dimensions"`
	LastEvolvedAt        time.Time `json:"last_evolved_at"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type EvolutionEvent struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Dimension  string    `json:"dimension"`
	OldValue   string    `json:"old_value"`
	NewValue   string    `json:"new_value"`
	Trigger    string    `json:"trigger"`
	Confidence float64   `json:"confidence"`
	CreatedAt  time.Time `json:"created_at"`
}

func (e EvolutionEvent) GetID() string {
	return fmt.Sprintf("%d_%s", e.CreatedAt.Unix(), e.Dimension)
}

type EnhancedSoulEvolution struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     string    `gorm:"index" json:"user_id"`
	Dimension  string    `json:"dimension"`
	OldValue   string    `json:"old_value"`
	NewValue   string    `json:"new_value"`
	Trigger    string    `json:"trigger"`
	Confidence float64   `json:"confidence"`
	CreatedAt  time.Time `json:"created_at"`
}

func (m *LettaMemory) AutoCreateEnhancedSoulTable() {
	if m.db.Migrator().HasTable(&EnhancedSoulData{}) {
		return
	}
	if err := m.db.AutoMigrate(&EnhancedSoulData{}); err != nil {
		log.Printf("[Memory] Failed to create enhanced_soul_data table: %v", err)
		return
	}
	if err := m.db.AutoMigrate(&EnhancedSoulEvolution{}); err != nil {
		log.Printf("[Memory] Failed to create enhanced_soul_evolution table: %v", err)
		return
	}
	log.Printf("[Memory] Enhanced soul tables created successfully")
}

func (m *LettaMemory) BuildEnhancedSoulPrompt(userID string, basePrompt string) string {
	soul := m.GetEnhancedSoul(userID)
	profile := m.GetUserSoul(userID)

	var sb strings.Builder

	sb.WriteString("## 用户画像 (增强版)\n")

	if profile.Persona != "" {
		sb.WriteString(fmt.Sprintf("- 基础人格: %s\n", profile.Persona))
	}

	if soul.ProfessionalLevel != "" {
		sb.WriteString(fmt.Sprintf("- 专业级别: %s\n", soul.ProfessionalLevel))
	}
	if soul.ReplyStyle != "" {
		sb.WriteString(fmt.Sprintf("- 回复风格: %s\n", soul.ReplyStyle))
	}
	if len(soul.PreferredMetrics) > 0 {
		sb.WriteString(fmt.Sprintf("- 常用指标: %s\n", strings.Join(soul.PreferredMetrics, ", ")))
	}
	if soul.FollowUpRate > 0 {
		sb.WriteString(fmt.Sprintf("- 追问率: %.1f%%\n", soul.FollowUpRate*100))
	}

	sb.WriteString("---\n\n")
	sb.WriteString(basePrompt)

	return sb.String()
}

// ════════════════════════════════════════════
// Auto-Learning: 从用户行为中自动学习偏好
// ════════════════════════════════════════════

// LearnUserHabits 从用户查询中自动学习并保存偏好。
// 在每次成功查询后调用，提取语言、常用指标、时间范围等习惯。
func (m *LettaMemory) LearnUserHabits(userID, query, detectedLang string) {
	if m == nil || m.db == nil || userID == "" {
		return
	}

	// 1. 语言偏好：如果检测到的语言与当前不同，更新
	soul := m.GetUserSoul(userID)
	lang := inferLanguage(detectedLang, query)
	if lang != "" && lang != soul.LanguagePreference {
		soul.LanguagePreference = lang
		_ = m.db.Save(soul).Error
		log.Printf("[Memory] 用户 %s 语言偏好更新为: %s", userID, lang)
	}

	// 2. 保存查询模式为长期记忆（用于语义召回）
	key := query
	if len([]rune(key)) > 30 {
		key = string([]rune(key)[:30])
	}
	m.SaveArchivalMemory(userID, "query_pattern", key,
		fmt.Sprintf("用户查询: %s", query), 1)

	// 3. 提取并保存常用指标偏好
	metrics := extractMetricHints(query)
	for _, metric := range metrics {
		m.SaveArchivalMemory(userID, "preferred_metric", metric,
			fmt.Sprintf("用户常用指标: %s", metric), 3)
	}
}

func inferLanguage(detectedLang, query string) string {
	if detectedLang != "" {
		return detectedLang
	}
	// 简单启发式：如果包含中文字符，判断为中文
	for _, r := range query {
		if r >= 0x4E00 && r <= 0x9FFF {
			return "zh-CN"
		}
	}
	// 否则判断为英文
	return "en"
}

func extractMetricHints(query string) []string {
	return nil
}

func formatMemoryKey(input string) string {
	// 截取前30个字符作为 key
	runes := []rune(input)
	if len(runes) > 30 {
		return string(runes[:30])
	}
	return input
}

// ════════════════════════════════════════════
// Prompt Assembly（组装 Letta-style prompt）
// ════════════════════════════════════════════

// BuildSoulPrompt 构建含 Soul + Memory 的 system prompt
func (m *LettaMemory) BuildSoulPrompt(userID string, groupIDs []string, query string, basePrompt string) string {
	var sb strings.Builder

	// 公司级 Soul
	company := m.GetCompanySoul()
	if company != nil {
		sb.WriteString(fmt.Sprintf("## 公司信息\n- 名称: %s\n- 行业: %s\n", company.Name, company.Industry))
		if company.BrandTone != "" {
			sb.WriteString(fmt.Sprintf("- 语调规则: %s\n", company.BrandTone))
		}
		sb.WriteString("\n")
	}

	// 用户级 Soul
	soul := m.GetUserSoul(userID)
	sb.WriteString("## 用户画像 (Soul)\n")
	sb.WriteString(fmt.Sprintf("- 人格: %s\n", soul.Persona))
	sb.WriteString(fmt.Sprintf("- 描述: %s\n", soul.Human))
	if soul.LanguagePreference != "" && soul.LanguagePreference != "zh-CN" {
		sb.WriteString(fmt.Sprintf("- 首选语言: %s（请用此语言回复）\n", soul.LanguagePreference))
	}
	if soul.Preferences != "" {
		sb.WriteString(fmt.Sprintf("- 偏好设置: %s\n", truncateStr(soul.Preferences, 200)))
	}
	sb.WriteString("\n")

	// 相关记忆召回：优先向量语义召回，失败时回退关键词召回
	if query != "" {
		memories, _ := m.RecallUserMemorySemantic(userID, query, 5)
		if len(memories) == 0 {
			memories, _ = m.RecallUserMemory(userID, query, 5)
		}
		if len(memories) > 0 {
			sb.WriteString("## 相关个人长期记忆（语义召回）\n")
			for _, mem := range memories {
				sb.WriteString(fmt.Sprintf("- [%s/%s] %s\n", mem.Type, mem.Key, truncateStr(mem.Content, 150)))
			}
			sb.WriteString("\n")
		}

		// 组记忆召回：优先向量语义召回，失败时回退关键词召回
		groupMems, _ := m.RecallGroupMemoriesSemantic(groupIDs, query, 3)
		if len(groupMems) == 0 {
			groupMems, _ = m.RecallGroupMemories(groupIDs, query, 3)
		}
		if len(groupMems) > 0 {
			sb.WriteString("## 团队/部门共享记忆（语义召回）\n")
			for _, mem := range groupMems {
				sb.WriteString(fmt.Sprintf("- [%s/%s] %s\n", mem.Type, mem.Key, truncateStr(mem.Content, 150)))
			}
			sb.WriteString("\n")
		}
	}

	// 基础 system prompt
	sb.WriteString("---\n\n")
	sb.WriteString(basePrompt)

	return sb.String()
}

// ── Helpers ──────────────────────────────────

func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// ── Deprecated wrappers (keep compatibility) ──
// 这些包装器保留旧接口，内部委托给 LettaMemory

type LongTermMemory = LettaMemory

func NewLongTermMemory(db *gorm.DB) *LongTermMemory {
	log.Println("[Memory] Letta-style memory system initialized")
	return NewLettaMemory(db)
}

func (m *LettaMemory) SaveUserMemory(userID, memType, key, content string, priority int) error {
	return m.UpsertUserMemory(userID, memType, key, content, priority)
}

func (m *LettaMemory) GetUserMemory(userID, memType string) ([]UserMemory, error) {
	var memories []UserMemory
	query := m.db.Where("user_id = ?", userID)
	if memType != "" {
		query = query.Where("type = ?", memType)
	}
	err := query.Order("priority DESC").Find(&memories).Error
	return memories, err
}

func (m *LettaMemory) GetUserMemoryByKey(userID, key string) (*UserMemory, error) {
	var mem UserMemory
	err := m.db.Where("user_id = ? AND key = ?", userID, key).First(&mem).Error
	if err != nil {
		return nil, err
	}
	return &mem, nil
}

func (m *LettaMemory) GetGroupMemories(groupIDs []string) ([]GroupMemory, error) {
	return m.RecallGroupMemories(groupIDs, "", 20)
}

func (m *LettaMemory) InjectMemoryIntoPrompt(userID string, groupIDs []string) string {
	return m.BuildSoulPrompt(userID, groupIDs, "", "")
}
