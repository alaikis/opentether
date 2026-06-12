package text2sql

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode"
)

const maxCandidateTables = 6

const (
	queryComplexityBasic = iota + 1
	queryComplexityDetail
	queryComplexityDimension
)

var schemaSelectionCache sync.Map

type cachedSchemaSelection struct {
	Schema  string
	Version string
}

type schemaTable struct {
	Name    string
	Columns []schemaColumn
	Raw     string
}

type schemaColumn struct {
	Name    string
	Type    string
	Comment string
	Raw     string
}

func (t *Text2SQL) selectRelevantSchema(question, dataSourceID, fullSchema string) string {
	if strings.TrimSpace(question) == "" || strings.TrimSpace(fullSchema) == "" {
		return fullSchema
	}

	cacheKey := fmt.Sprintf("%s:%d:%s", dataSourceID, len(fullSchema), normalizeQuestionForSchema(question))
	if cached, ok := schemaSelectionCache.Load(cacheKey); ok {
		if item, ok := cached.(cachedSchemaSelection); ok && item.Version == dataSourceID {
			return item.Schema
		}
	}

	tables := parseSchemaTables(fullSchema)
	if len(tables) == 0 || len(tables) <= maxCandidateTables {
		return fullSchema
	}

	relations := t.loadTableRelations(dataSourceID)
	complexity := detectQueryComplexity(question)
	limit := maxCandidateTables
	if complexity == queryComplexityDetail {
		limit = 8
	} else if complexity == queryComplexityDimension {
		limit = 10
	}
	selected := selectCandidateTables(question, tables, relations, limit, complexity)
	if len(selected) == 0 {
		// 没有任何中文关键词匹配到表，尝试用启发式规则选择常见业务表
		selected = selectFallbackTables(tables, limit, complexity)
		if len(selected) == 0 {
			// 如果连常见表都选不出来，截取前几个表（避免把大 schema 全送入 LLM）
			if len(tables) > maxCandidateTables {
				for i := 0; i < maxCandidateTables && i < len(tables); i++ {
					selected = append(selected, tables[i].Name)
				}
			} else {
				return fullSchema
			}
		}
	}

	selectedSchema := renderSelectedSchema(question, tables, selected, relations)
	schemaSelectionCache.Store(cacheKey, cachedSchemaSelection{Schema: selectedSchema, Version: dataSourceID})
	return selectedSchema
}

func parseSchemaTables(schema string) []schemaTable {
	if tables := parseSchemaTablesFromJSON(schema); len(tables) > 0 {
		return tables
	}
	lines := strings.Split(schema, "\n")
	var tables []schemaTable
	var current *schemaTable
	var raw []string

	flush := func() {
		if current != nil {
			current.Raw = strings.Join(raw, "\n")
			tables = append(tables, *current)
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "表:") {
			flush()
			name := strings.TrimSpace(strings.TrimPrefix(trimmed, "表:"))
			current = &schemaTable{Name: name}
			raw = []string{line}
			continue
		}
		if current == nil {
			continue
		}
		raw = append(raw, line)
		if strings.HasPrefix(trimmed, "-") {
			colRaw := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			parts := strings.SplitN(colRaw, ":", 2)
			col := schemaColumn{Name: strings.TrimSpace(parts[0]), Raw: line}
			if len(parts) == 2 {
				col.Type = strings.TrimSpace(parts[1])
			}
			if col.Name != "" {
				current.Columns = append(current.Columns, col)
			}
		}
	}
	flush()
	return tables
}

func parseSchemaTablesFromJSON(schema string) []schemaTable {
	trimmed := strings.TrimSpace(schema)
	if !strings.HasPrefix(trimmed, "[") {
		return nil
	}
	var rawTables []struct {
		Name    string `json:"name"`
		Columns []struct {
			Name     string `json:"name"`
			Type     string `json:"type"`
			Nullable bool   `json:"nullable"`
			KeyType  string `json:"key_type"`
			Comment  string `json:"comment"`
		} `json:"columns"`
	}
	if err := json.Unmarshal([]byte(trimmed), &rawTables); err != nil {
		return nil
	}
	tables := make([]schemaTable, 0, len(rawTables))
	for _, raw := range rawTables {
		if raw.Name == "" {
			continue
		}
		table := schemaTable{Name: raw.Name}
		var rawLines []string
		rawLines = append(rawLines, "表: "+raw.Name)
		for _, col := range raw.Columns {
			if col.Name == "" {
				continue
			}
			colType := strings.TrimSpace(col.Type)
			if col.KeyType != "" {
				colType += " [" + col.KeyType + "]"
			}
			comment := strings.TrimSpace(col.Comment)
			if comment != "" {
				colType += " -- " + comment
			}
			rawLine := fmt.Sprintf("  - %s: %s", col.Name, colType)
			table.Columns = append(table.Columns, schemaColumn{Name: col.Name, Type: colType, Comment: comment, Raw: rawLine})
			rawLines = append(rawLines, rawLine)
		}
		table.Raw = strings.Join(rawLines, "\n")
		tables = append(tables, table)
	}
	return tables
}

func selectCandidateTables(question string, tables []schemaTable, relations []map[string]string, limit int, complexity int) []string {
	tokens := schemaTokens(question)
	tableScores := map[string]int{}

	for _, table := range tables {
		score := scoreTable(question, tokens, table)
		if score > 0 {
			tableScores[table.Name] = score
		}
	}

	applyBusinessHints(question, tables, tableScores)
	applyComplexityScores(tables, tableScores, complexity)

	type candidate struct {
		Name  string
		Score int
	}
	candidates := make([]candidate, 0, len(tableScores))
	for name, score := range tableScores {
		candidates = append(candidates, candidate{Name: name, Score: score})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Name < candidates[j].Name
		}
		return candidates[i].Score > candidates[j].Score
	})

	selected := map[string]bool{}
	var result []string
	for _, c := range candidates {
		if len(result) >= limit {
			break
		}
		selected[c.Name] = true
		result = append(result, c.Name)
	}

	// Include directly related bridge tables when space allows.
	for _, rel := range relations {
		if len(result) >= limit {
			break
		}
		from := rel["from_table"]
		to := rel["to_table"]
		if selected[from] && !selected[to] {
			selected[to] = true
			result = append(result, to)
		} else if selected[to] && !selected[from] {
			selected[from] = true
			result = append(result, from)
		}
	}

	return result
}

func scoreTable(question string, tokens []string, table schemaTable) int {
	score := 0
	name := strings.ToLower(table.Name)
	q := strings.ToLower(question)
	if strings.Contains(q, name) {
		score += 20
	}
	for _, token := range tokens {
		if token == "" {
			continue
		}
		if strings.Contains(name, token) {
			score += 8
		}
	}
	for _, col := range table.Columns {
		colName := strings.ToLower(col.Name)
		if strings.Contains(q, colName) {
			score += 10
		}
		for _, token := range tokens {
			if strings.Contains(colName, token) {
				score += 3
			}
		}
	}
	return score
}

func detectQueryComplexity(question string) int {
	dimensionTokens := []string{"分类", "品类", "类目", "商品", "产品", "sku", "SKU", "变体", "型号"}
	for _, token := range dimensionTokens {
		if strings.Contains(question, token) {
			return queryComplexityDimension
		}
	}
	detailTokens := []string{"明细", "详情", "列表", "订单号", "客户", "地址", "手机号"}
	for _, token := range detailTokens {
		if strings.Contains(question, token) {
			return queryComplexityDetail
		}
	}
	return queryComplexityBasic
}

func applyComplexityScores(tables []schemaTable, scores map[string]int, complexity int) {
	for _, table := range tables {
		name := strings.ToLower(table.Name)
		switch complexity {
		case queryComplexityBasic:
			if strings.Contains(name, "product") || strings.Contains(name, "goods") || strings.Contains(name, "sku") || strings.Contains(name, "category") || strings.Contains(name, "variant") || strings.Contains(name, "item") {
				scores[table.Name] -= 12
			}
		case queryComplexityDetail:
			if strings.Contains(name, "item") || strings.Contains(name, "detail") || strings.Contains(name, "customer") || strings.Contains(name, "address") {
				scores[table.Name] += 8
			}
		case queryComplexityDimension:
			if strings.Contains(name, "product") || strings.Contains(name, "goods") || strings.Contains(name, "sku") || strings.Contains(name, "category") || strings.Contains(name, "variant") || strings.Contains(name, "item") {
				scores[table.Name] += 12
			}
		}
	}
}

func applyBusinessHints(question string, tables []schemaTable, scores map[string]int) {
	hints := map[string][]string{
		"订单":   {"order", "sale", "sales"},
		"出单":   {"order", "sale", "sales"},
		"多少单":  {"order", "sale", "sales"},
		"订单数":  {"order", "sale", "sales"},
		"订单数量": {"order", "sale", "sales"},
		"销量":   {"order", "sale", "sales"},
		"下单":   {"order", "sale", "sales"},
		"销售":   {"order", "sale", "sales"},
		"销售额":  {"order", "sale", "sales", "amount", "pay", "price"},
		"金额":   {"amount", "pay", "price", "total"},
		"员工":   {"profile", "staff", "employee", "user"},
		"姓名":   {"profile", "staff", "employee", "user"},
		"林烽":   {"profile", "staff", "employee", "user", "user_id"},
		"客户":   {"customer", "client"},
		"产品":   {"product", "goods", "sku"},
		"库存":   {"stock", "inventory", "warehouse"},
		"时间":   {"time", "date", "created", "create"},
		"月份":   {"time", "date", "created", "create"},
	}
	for keyword, aliases := range hints {
		if !strings.Contains(question, keyword) {
			continue
		}
		for _, table := range tables {
			searchText := strings.ToLower(table.Name)
			for _, col := range table.Columns {
				searchText += " " + strings.ToLower(col.Name)
			}
			for _, alias := range aliases {
				if strings.Contains(searchText, alias) {
					scores[table.Name] += 12
				}
			}
		}
	}
}

func renderSelectedSchema(question string, tables []schemaTable, selected []string, relations []map[string]string) string {
	selectedSet := map[string]bool{}
	for _, name := range selected {
		selectedSet[name] = true
	}

	var sb strings.Builder
	sb.WriteString("数据库候选表结构（已根据问题预筛选，请优先只使用这些表和字段）：\n\n")
	sb.WriteString("用户问题: " + question + "\n\n")
	for _, table := range tables {
		if !selectedSet[table.Name] {
			continue
		}
		sb.WriteString(fmt.Sprintf("表: %s\n", table.Name))
		for _, col := range table.Columns {
			line := fmt.Sprintf("  - %s: %s", col.Name, col.Type)
			if col.Comment != "" {
				line += fmt.Sprintf(" -- %s", col.Comment)
			}
			sb.WriteString(line + "\n")
		}
		sb.WriteString("\n")
	}

	relationText := renderRelevantRelations(selectedSet, relations)
	if relationText != "" {
		sb.WriteString(relationText)
	}
	return sb.String()
}

func renderRelevantRelations(selected map[string]bool, relations []map[string]string) string {
	if len(relations) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, rel := range relations {
		from := rel["from_table"]
		to := rel["to_table"]
		if selected[from] || selected[to] {
			if sb.Len() == 0 {
				sb.WriteString("候选表关系：\n")
			}
			sb.WriteString(fmt.Sprintf("  %s.%s → %s.%s\n", from, rel["from_column"], to, rel["to_column"]))
		}
	}
	if sb.Len() > 0 {
		sb.WriteString("\n")
	}
	return sb.String()
}

func (t *Text2SQL) loadTableRelations(dataSourceID string) []map[string]string {
	if t == nil || t.db == nil || dataSourceID == "" {
		return nil
	}
	var raw struct{ TableRelations string }
	if err := t.db.Model(&struct{}{}).Table("data_sources").Select("table_relations").Where("id = ?", dataSourceID).Scan(&raw).Error; err != nil {
		return nil
	}
	if raw.TableRelations == "" || raw.TableRelations == "[]" {
		return nil
	}
	var rels []map[string]string
	if err := json.Unmarshal([]byte(raw.TableRelations), &rels); err != nil {
		return nil
	}
	return rels
}

func schemaTokens(text string) []string {
	text = strings.ToLower(text)
	var tokens []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			tokens = append(tokens, string(current))
			current = nil
		}
	}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			current = append(current, r)
			continue
		}
		flush()
	}
	flush()

	// Add CJK bigrams for matching Chinese business terms.
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		if isCJKRune(runes[i]) {
			tokens = append(tokens, string(runes[i]))
			if i+1 < len(runes) && isCJKRune(runes[i+1]) {
				tokens = append(tokens, string(runes[i:i+2]))
			}
		}
	}
	return tokens
}

func normalizeQuestionForSchema(question string) string {
	return strings.Join(schemaTokens(question), "|")
}

func isCJKRune(r rune) bool {
	return r >= 0x4E00 && r <= 0x9FFF
}

// selectFallbackTables 从大表中用启发式规则选出常见业务表（order、sale、staff、product 等）
func selectFallbackTables(tables []schemaTable, limit int, complexity int) []string {
	commonPatterns := []string{"order", "sale", "profile", "staff", "employee", "user", "pay", "bill", "invoice", "work_order"}
	if complexity >= queryComplexityDetail {
		commonPatterns = append(commonPatterns, "order_item", "item", "detail", "customer", "address")
	}
	if complexity >= queryComplexityDimension {
		commonPatterns = append(commonPatterns, "product", "goods", "sku", "category", "variant", "t_product", "t_goods")
	}
	scores := map[string]int{}
	for _, table := range tables {
		name := strings.ToLower(table.Name)
		for _, p := range commonPatterns {
			if strings.Contains(name, p) {
				scores[table.Name] += 10
			}
		}
		// 也检查列名中的业务术语
		for _, col := range table.Columns {
			colName := strings.ToLower(col.Name)
			if strings.Contains(colName, "order") || strings.Contains(colName, "sale") ||
				strings.Contains(colName, "staff") || strings.Contains(colName, "employee") ||
				strings.Contains(colName, "amount") || strings.Contains(colName, "price") {
				scores[table.Name] += 5
			}
		}
	}
	type candidate struct {
		Name  string
		Score int
	}
	var list []candidate
	for name, score := range scores {
		if score > 0 {
			list = append(list, candidate{Name: name, Score: score})
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Score > list[j].Score })
	var result []string
	for i := 0; i < limit && i < len(list); i++ {
		result = append(result, list[i].Name)
	}
	return result
}
