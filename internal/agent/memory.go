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

// GetUserSoul 获取用户 Soul（含默认值）
func (m *LettaMemory) GetUserSoul(userID string) *UserProfile {
	var p UserProfile
	if err := m.db.Where("user_id = ?", userID).First(&p).Error; err != nil {
		return &UserProfile{
			Persona:            "你是 OpenTether AI 助手，专业且友好。",
			Human:              "用户是公司员工。",
			LanguagePreference: "zh-CN",
		}
	}
	return &p
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
	for _, kw := range []string{
		"订单", "销售", "库存", "员工", "成本", "客户",
		"采购", "报表", "产品", "利润", "业绩",
		"部门", "仓库", "物流", "广告", "价格",
	} {
		if strings.Contains(query, kw) {
			return kw
		}
	}
	return ""
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
	sb.WriteString("/" + "* 用户画像 (Soul) */")
	sb.WriteString(fmt.Sprintf("\n- AI 人格 (Persona): %s\n", soul.Persona))
	sb.WriteString(fmt.Sprintf("- 用户描述 (Human): %s\n", soul.Human))
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
