package agent

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/embedding"
	"github.com/alaikis/opentether/internal/models"
	"github.com/alaikis/opentether/internal/vectorstore"
	"gorm.io/gorm"
)

type skillRuntimeMemoryMatch struct {
	Memory models.SkillRuntimeMemory
	Score  float64
}

func (e *AgentEngine) buildText2SQLRuntimeContext(skillID, dataSourceID string) string {
	if e == nil || e.db == nil || dataSourceID == "" {
		return ""
	}
	var memories []models.SkillRuntimeMemory
	query := e.db.Where("data_source_id = ? AND confidence >= ?", dataSourceID, 0.55)
	if skillID != "" {
		query = query.Where("skill_id = ? OR skill_id = ''", skillID)
	}
	if err := query.Order("confidence DESC, use_count DESC, updated_at DESC").Limit(12).Find(&memories).Error; err != nil || len(memories) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n## Text2SQL 运行时学习上下文（高置信，可复用）\n")
	for _, mem := range memories {
		sb.WriteString(fmt.Sprintf("- [%s/%s conf=%.2f used=%d] %s\n", mem.Type, mem.Key, mem.Confidence, mem.UseCount, mem.Content))
	}
	return sb.String()
}

func (e *AgentEngine) recallSkillRuntimeMemorySemantic(dataSourceID, queryText string, limit int, threshold float64) []skillRuntimeMemoryMatch {
	if e == nil || e.db == nil || dataSourceID == "" || strings.TrimSpace(queryText) == "" {
		return nil
	}
	var memories []models.SkillRuntimeMemory
	if err := e.db.Where("data_source_id = ? AND confidence >= ? AND source <> ?", dataSourceID, 0.45, "rejected").
		Order("confidence DESC, use_count DESC, updated_at DESC").
		Limit(300).
		Find(&memories).Error; err != nil || len(memories) == 0 {
		return nil
	}
	docs := make([]string, 0, len(memories)+1)
	docs = append(docs, queryText)
	for _, mem := range memories {
		docs = append(docs, runtimeMemorySearchText(mem))
	}
	embedder, err := embedding.Create("tfidf", map[string]interface{}{"corpus": docs})
	if err != nil {
		return nil
	}
	store, err := vectorstore.CreateStore("memory", nil)
	if err != nil {
		return nil
	}
	byID := map[string]models.SkillRuntimeMemory{}
	for _, mem := range memories {
		vec, err := embedder.Embed(runtimeMemorySearchText(mem))
		if err != nil {
			continue
		}
		_ = store.Index(mem.ID, mem.Key, vec)
		byID[mem.ID] = mem
	}
	queryVec, err := embedder.Embed(queryText)
	if err != nil {
		return nil
	}
	matches, err := store.Search(queryVec, limit, threshold)
	if err != nil {
		return nil
	}
	result := make([]skillRuntimeMemoryMatch, 0, len(matches))
	for _, match := range matches {
		if mem, ok := byID[match.SkillID]; ok {
			result = append(result, skillRuntimeMemoryMatch{Memory: mem, Score: match.Score})
		}
	}
	return result
}

func runtimeMemorySearchText(mem models.SkillRuntimeMemory) string {
	return strings.TrimSpace(mem.Type + " " + mem.Key + " " + mem.Content)
}

func hasUsefulText2SQLRuntimeMemory(matches []skillRuntimeMemoryMatch) bool {
	if len(matches) == 0 {
		return false
	}
	strong := 0
	for _, match := range matches {
		if match.Score >= 0.12 || match.Memory.Confidence >= 0.75 || match.Memory.Source == "admin" {
			strong++
		}
	}
	return strong >= 2 || (len(matches) > 0 && matches[0].Memory.Source == "admin")
}

func (e *AgentEngine) learnText2SQLRuntime(skillID, dataSourceID, question, sql string, columns []string, rowCount int) {
	if e == nil || e.db == nil || dataSourceID == "" || strings.TrimSpace(sql) == "" {
		return
	}
	if strings.Contains(strings.ToLower(sql), "unable to connect") || strings.Contains(strings.ToLower(sql), "connection refused") || strings.Contains(strings.ToLower(sql), "192.168") || strings.Contains(strings.ToLower(sql), "10.") || strings.Contains(strings.ToLower(question), "192.168") || strings.Contains(strings.ToLower(question), "无法连接") || strings.Contains(strings.ToLower(question), "连接失败") || strings.Contains(strings.ToLower(question), "不可用") {
		return
	}
	upper := strings.ToUpper(strings.TrimSpace(sql))
	if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "WITH") {
		return
	}
	facts := extractSQLRuntimeFacts(sql, question)
	now := time.Now()
	for _, table := range facts.Tables {
		content := fmt.Sprintf("表 %s 在问题“%s”中被成功用于查询。", table, truncateForMemory(question, 80))
		e.upsertSkillRuntimeMemory(skillID, dataSourceID, "table_usage", table, content, 0.58)
		_ = e.db.Model(&models.SkillRuntimeMemory{}).Where("data_source_id = ? AND type = ? AND key = ? AND confidence < ? AND (content LIKE ? OR content LIKE ?)", dataSourceID, "table_usage", table, 0.55, "%无法连接%", "%192.168%").Update("source", "corrected").Error
	}
	for _, rel := range facts.Relations {
		_ = e.db.Model(&models.SkillRuntimeMemory{}).Where("data_source_id = ? AND type = ? AND key = ? AND confidence < ? AND (content LIKE ? OR content LIKE ?)", dataSourceID, "table_relation", rel, 0.55, "%无法连接%", "%192.168%").Update("source", "corrected").Error
		e.upsertSkillRuntimeMemory(skillID, dataSourceID, "table_relation", rel, "成功 SQL 使用关系: "+rel, 0.62)
	}
	for _, metric := range facts.Metrics {
		e.upsertSkillRuntimeMemory(skillID, dataSourceID, "metric_rule", metric.Key, metric.Content, 0.6)
	}
	if len(facts.Tables) > 0 {
		patternKey := strings.Join(facts.Tables, "+")
		normalized := normalizeSQLForMemory(sql)
		if sanitized := sanitizeContentForMemory(normalized); sanitized != "" {
			e.upsertSkillRuntimeMemory(skillID, dataSourceID, "sql_pattern", patternKey, fmt.Sprintf("问题模式: %s\nSQL 模板参考: %s", truncateForMemory(question, 120), truncateForMemory(sanitized, 500)), 0.56)
			if tpl, ok := buildParameterizedText2SQLTemplate(question, sanitized); ok {
				e.upsertSkillRuntimeMemory(skillID, dataSourceID, "text2sql_template", tpl.Key, tpl.Content, 0.5)
			}
		}
	}
	_ = now
}

type text2sqlTemplateCandidate struct {
	Key     string
	Content string
}

func buildParameterizedText2SQLTemplate(question, sql string) (text2sqlTemplateCandidate, bool) {
	employee := extractEmployeeName(question)
	if employee == "" {
		return text2sqlTemplateCandidate{}, false
	}
	tpl := sql
	for _, quoted := range []string{"'" + employee + "'", "\"" + employee + "\""} {
		tpl = strings.ReplaceAll(tpl, quoted, ":employee_name")
	}
	dateRe := regexp.MustCompile(`'\d{4}-\d{2}-\d{2}(?: [^']*)?'`)
	dateIdx := 0
	tpl = dateRe.ReplaceAllStringFunc(tpl, func(s string) string {
		dateIdx++
		if dateIdx == 1 {
			return ":start_time"
		}
		if dateIdx == 2 {
			return ":end_time"
		}
		return s
	})
	if !strings.Contains(tpl, ":employee_name") {
		return text2sqlTemplateCandidate{}, false
	}
	metric := "通用指标"
	if strings.Contains(question, "销售额") || strings.Contains(strings.ToLower(sql), "sum(") {
		metric = "销售额"
	} else if strings.Contains(question, "订单") || strings.Contains(strings.ToLower(sql), "count(") {
		metric = "订单数"
	}
	payload := fmt.Sprintf(`{"intent":"employee_metric_by_time_range","metric":%q,"slots":["employee_name","time_range"],"sql_template":%q}`, metric, tpl)
	return text2sqlTemplateCandidate{Key: "employee_metric_by_time_range:" + metric, Content: payload}, true
}

type sqlRuntimeFacts struct {
	Tables    []string
	Relations []string
	Metrics   []runtimeMetric
}

type runtimeMetric struct {
	Key     string
	Content string
}

func extractSQLRuntimeFacts(sql, question string) sqlRuntimeFacts {
	facts := sqlRuntimeFacts{}
	facts.Tables = extractSQLTables(sql)
	facts.Relations = extractSQLJoinRelations(sql)
	facts.Metrics = extractSQLMetrics(sql, question)
	return facts
}

func extractSQLTables(sql string) []string {
	re := regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+([a-zA-Z0-9_\.]+)`)
	matches := re.FindAllStringSubmatch(sql, -1)
	seen := map[string]bool{}
	var tables []string
	for _, m := range matches {
		name := strings.Trim(m[1], "`\" ")
		if idx := strings.LastIndex(name, "."); idx >= 0 {
			name = name[idx+1:]
		}
		if name != "" && !seen[name] {
			seen[name] = true
			tables = append(tables, name)
		}
	}
	sort.Strings(tables)
	return tables
}

func extractSQLJoinRelations(sql string) []string {
	re := regexp.MustCompile(`(?i)\bON\s+([a-zA-Z0-9_\.]+)\s*=\s*([a-zA-Z0-9_\.]+)`)
	matches := re.FindAllStringSubmatch(sql, -1)
	seen := map[string]bool{}
	var relations []string
	for _, m := range matches {
		rel := strings.TrimSpace(m[1]) + " = " + strings.TrimSpace(m[2])
		if !seen[rel] {
			seen[rel] = true
			relations = append(relations, rel)
		}
	}
	sort.Strings(relations)
	return relations
}

func extractSQLMetrics(sql, question string) []runtimeMetric {
	lowerSQL := strings.ToLower(sql)
	var metrics []runtimeMetric
	if strings.Contains(lowerSQL, "count(") || strings.Contains(question, "多少单") || strings.Contains(question, "订单数") {
		metrics = append(metrics, runtimeMetric{Key: "订单数", Content: "订单数类问题通常使用 COUNT(...) 聚合统计，时间字段按问题时间范围过滤。"})
	}
	if strings.Contains(lowerSQL, "sum(") || strings.Contains(question, "销售额") || strings.Contains(question, "金额") {
		metrics = append(metrics, runtimeMetric{Key: "销售额", Content: "销售额/金额类问题通常使用 SUM(金额字段) 聚合；若结果为空应使用 COALESCE 兜底为 0。"})
	}
	if strings.Contains(lowerSQL, "create_time") || strings.Contains(lowerSQL, "created_at") {
		metrics = append(metrics, runtimeMetric{Key: "时间过滤", Content: "时间范围类问题优先使用 create_time/created_at 等创建时间字段进行过滤。"})
	}
	return metrics
}

func (e *AgentEngine) upsertSkillRuntimeMemory(skillID, dataSourceID, memType, key, content string, confidence float64) {
	if key == "" || content == "" {
		return
	}
	var existing models.SkillRuntimeMemory
	now := time.Now()
	err := e.db.Where("skill_id = ? AND data_source_id = ? AND type = ? AND key = ?", skillID, dataSourceID, memType, key).First(&existing).Error
	if err == nil {
		existing.Content = content
		existing.UseCount++
		existing.LastUsedAt = now
		if existing.Confidence < 0.95 {
			existing.Confidence += 0.05
		}
		_ = e.db.Save(&existing).Error
		return
	}
	if err != gorm.ErrRecordNotFound {
		return
	}
	mem := models.SkillRuntimeMemory{
		SkillID:      skillID,
		DataSourceID: dataSourceID,
		Type:         memType,
		Key:          key,
		Content:      content,
		Confidence:   confidence,
		UseCount:     1,
		Source:       "runtime",
		LastUsedAt:   now,
	}
	_ = e.db.Create(&mem).Error
}

func normalizeSQLForMemory(sql string) string {
	return strings.Join(strings.Fields(sql), " ")
}

func sanitizeContentForMemory(s string) string {
	if strings.Contains(s, "192.168") || strings.Contains(s, "10.") || strings.Contains(s, "无法连接") || strings.Contains(s, "connection refused") || strings.Contains(s, "unable to connect") || strings.Contains(s, "连接失败") || strings.Contains(s, "不可用") || strings.Contains(s, "丢包") {
		return ""
	}
	return s
}

func truncateForMemory(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if max <= 0 || len(r) <= max {
		return strings.TrimSpace(s)
	}
	return string(r[:max]) + "..."
}
