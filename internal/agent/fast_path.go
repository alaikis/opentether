package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"github.com/alaikis/opentether/internal/templating"
)

const defaultFastRouterJinja = `你是轻量路由模型。判断用户问题是否可走快路径。
只返回 JSON，不要解释。route 只能是 fast_chat、fast_text2sql、agent_loop。
- fast_chat: 简单知识/解释/闲聊，不需要工具。
- fast_text2sql: 简单数据查询，尤其是订单数、销售额、员工维度统计。
- agent_loop: 多步骤、文件、报表、MCP、写操作、复杂分析。

用户问题：{{ message }}
`

type fastPathResult struct {
	Response *ChatResponse
	Hit      bool
}

func (e *AgentEngine) TryFastPath(ctx context.Context, user *UserContext, message, conversationID string) (*ChatResponse, bool, error) {
	if shouldSkipFastPath(message) {
		return nil, false, nil
	}
	if resp, ok := tryLocalFastAnswer(message, conversationID); ok {
		return resp, true, nil
	}
	route := e.routeFastPath(message)
	if route.Route == "" {
		route = e.routeFastPathWithSmallModel(ctx, user, message)
	}
	if route.Route == "fast_text2sql" || route.Intent == "text2sql" {
		if resp, ok := e.tryText2SQLApprovedTemplateFastPath(ctx, user, message, conversationID); ok {
			return resp, true, nil
		}
		if resp, ok := e.tryText2SQLTemplateFastPath(ctx, user, message, conversationID); ok {
			return resp, true, nil
		}
	}
	if route.Route == "fast_chat" || (route.Route == "" && isSimpleChat(message)) {
		resp, err := e.executeFastChat(ctx, user, message, conversationID)
		if err == nil && resp != nil {
			return resp, true, nil
		}
	}
	return nil, false, nil
}

type fastPathRoute struct {
	Route      string  `json:"route"`
	Intent     string  `json:"intent"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

func (e *AgentEngine) routeFastPath(message string) fastPathRoute {
	prediction := e.routeByEmbeddedClassifier(message)
	if prediction.Route == "" {
		return fastPathRoute{}
	}
	return fastPathRoute{Route: prediction.Route, Intent: prediction.Intent, Confidence: prediction.Confidence, Reason: "内置 TF-IDF FastPathClassifier 命中: " + prediction.MatchedText}
}

func (e *AgentEngine) routeFastPathWithSmallModel(ctx context.Context, user *UserContext, message string) fastPathRoute {
	dataSourceID := ""
	if user != nil && user.Context != nil {
		dataSourceID, _ = user.Context["data_source_id"].(string)
	}
	if matches := e.recallSkillRuntimeMemorySemantic(dataSourceID, message, 5, 0.08); hasUsefulText2SQLRuntimeMemory(matches) {
		return fastPathRoute{Route: "fast_text2sql", Intent: "text2sql", Confidence: 0.86, Reason: "命中高置信 Text2SQL 运行时记忆"}
	}
	provider, err := e.providers.GetProviderByRole("fast_router")
	if err != nil || provider == nil {
		provider, err = e.providers.GetActiveProvider()
		if err != nil || provider == nil {
			return fastPathRoute{}
		}
	}
	prompt := templating.SafeRender(defaultFastRouterJinja, map[string]interface{}{"message": message}, fmt.Sprintf("用户问题：%s", message))
	answer, err := e.providers.CallLLM(ctx, provider, prompt)
	if err != nil {
		return fastPathRoute{}
	}
	var route fastPathRoute
	candidate := extractJSONFromText(answer)
	if json.Unmarshal([]byte(candidate), &route) != nil {
		return fastPathRoute{}
	}
	if route.Confidence < 0.78 {
		return fastPathRoute{}
	}
	return route
}

func extractJSONFromText(text string) string {
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, "{"); idx >= 0 {
		text = text[idx:]
		if end := strings.LastIndex(text, "}"); end >= 0 {
			return text[:end+1]
		}
	}
	return text
}

func tryLocalFastAnswer(message, conversationID string) (*ChatResponse, bool) {
	m := strings.TrimSpace(strings.ToLower(message))
	now := time.Now()
	switch {
	case m == "你好" || m == "hello" || m == "hi" || m == "嗨":
		return &ChatResponse{Message: "你好，我是 OpenTether AI 助手。", ConversationID: conversationID, SkillUsed: "fast_local"}, true
	case strings.Contains(m, "你是谁") || strings.Contains(m, "介绍一下你"):
		return &ChatResponse{Message: "我是 OpenTether 企业级 AI Agent，可以帮助你进行数据查询、报表生成、任务执行和业务分析。", ConversationID: conversationID, SkillUsed: "fast_local"}, true
	case strings.Contains(m, "今天") && (strings.Contains(m, "日期") || strings.Contains(m, "几号") || strings.Contains(m, "星期")):
		return &ChatResponse{Message: fmt.Sprintf("今天是 %s。", now.Format("2006-01-02 Monday")), ConversationID: conversationID, SkillUsed: "fast_local"}, true
	case m == "帮助" || m == "help":
		return &ChatResponse{Message: "你可以直接问我：查询订单数据、统计销售额、生成报表、分析员工或调用已配置的 MCP/Skills。", ConversationID: conversationID, SkillUsed: "fast_local"}, true
	default:
		return nil, false
	}
}

func (e *AgentEngine) executeFastChat(ctx context.Context, user *UserContext, message, conversationID string) (*ChatResponse, error) {
	provider, err := e.providers.GetActiveProvider()
	if err != nil || provider == nil {
		return nil, err
	}
	prompt := fmt.Sprintf("请用中文简洁回答，不要使用工具。用户问题：%s", message)
	answer, err := e.providers.CallLLM(ctx, provider, prompt)
	if err != nil {
		return nil, err
	}
	return &ChatResponse{Message: answer, ConversationID: conversationID, SkillUsed: "fast_chat"}, nil
}

func isSimpleChat(message string) bool {
	m := strings.TrimSpace(message)
	if m == "" || len([]rune(m)) > 120 {
		return false
	}
	blocked := []string{"查询", "订单", "销售额", "多少单", "库存", "报表", "pdf", "excel", "执行", "脚本", "生成", "下载", "数据库", "sql", "员工"}
	lower := strings.ToLower(m)
	for _, kw := range blocked {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return false
		}
	}
	return strings.Contains(m, "什么") || strings.Contains(m, "为什么") || strings.Contains(m, "如何") || strings.Contains(m, "怎么") || strings.Contains(m, "解释")
}

type text2sqlTemplateIntent struct {
	Employee  string
	Metric    string
	TimeRange string
}

func (e *AgentEngine) tryText2SQLApprovedTemplateFastPath(ctx context.Context, user *UserContext, message, conversationID string) (*ChatResponse, bool) {
	intent, ok := parseText2SQLTemplateIntent(message)
	if !ok || user == nil || e == nil || e.db == nil || e.externalDBPool == nil {
		return nil, false
	}
	dataSourceID := ""
	skillID := ""
	if user.Context != nil {
		dataSourceID, _ = user.Context["data_source_id"].(string)
		skillID, _ = user.Context["selected_skill_id"].(string)
	}
	if dataSourceID == "" {
		return nil, false
	}
	var mems []struct{ Content string }
	q := e.db.Model(&models.SkillRuntimeMemory{}).
		Select("content").
		Where("data_source_id = ? AND type = ? AND key = ? AND (source = ? OR confidence >= ?)", dataSourceID, "text2sql_template", "employee_metric_by_time_range:"+intent.Metric, "admin", 0.9)
	if skillID != "" {
		q = q.Where("skill_id = ? OR skill_id = ''", skillID)
	}
	if err := q.Order("confidence DESC, updated_at DESC").Limit(1).Find(&mems).Error; err != nil || len(mems) == 0 {
		return nil, false
	}
	var tpl struct {
		SQLTemplate string `json:"sql_template"`
	}
	if json.Unmarshal([]byte(mems[0].Content), &tpl) != nil || tpl.SQLTemplate == "" {
		return nil, false
	}
	start, end, ok := resolveChineseTimeRange(intent.TimeRange)
	if !ok {
		return nil, false
	}
	db, err := e.externalDBPool.Get(ctx, dataSourceID)
	if err != nil {
		return nil, false
	}
	sqlQuery, args := renderText2SQLTemplate(tpl.SQLTemplate, intent.Employee, start, end)
	if !isReadOnlyTemplateSQL(sqlQuery) {
		return nil, false
	}
	rows, err := db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var value interface{}
	if rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if rows.Scan(ptrs...) == nil && len(vals) > 0 {
			value = vals[0]
		}
	}
	answer := fmt.Sprintf("%s%s的%s为 %v。", intent.Employee, intent.TimeRange, intent.Metric, normalizeDBValue(value))
	return &ChatResponse{Message: answer, ConversationID: conversationID, SkillUsed: "fast_text2sql_approved_template", Data: map[string]interface{}{"sql": sqlQuery, "args": args, "data_source_id": dataSourceID}}, true
}

func (e *AgentEngine) tryText2SQLTemplateFastPath(ctx context.Context, user *UserContext, message, conversationID string) (*ChatResponse, bool) {
	intent, ok := parseText2SQLTemplateIntent(message)
	if !ok || user == nil {
		return nil, false
	}
	question := fmt.Sprintf("%s%s的%s", intent.Employee, intent.TimeRange, intent.Metric)
	resp, err := e.executeText2SQL(question, user)
	if err != nil || resp == nil {
		return nil, false
	}
	// 如果结果包含错误（如数据源无法生成查询），视为快路径失败，回退到 agent loop
	if resp.Data != nil {
		if errMsg, ok := resp.Data["error"].(string); ok && errMsg != "" {
			return nil, false
		}
		if status, ok := resp.Data["status"].(string); ok && status == "failed" {
			return nil, false
		}
	}
	if resp.Message == "" {
		return nil, false
	}
	resp.ConversationID = conversationID
	resp.SkillUsed = "fast_text2sql_template"
	return resp, true
}

var chineseMonthMap = map[string]time.Month{
	"一月":  time.January,
	"二月":  time.February,
	"三月":  time.March,
	"四月":  time.April,
	"五月":  time.May,
	"六月":  time.June,
	"七月":  time.July,
	"八月":  time.August,
	"九月":  time.September,
	"十月":  time.October,
	"十一月": time.November,
	"十二月": time.December,
	"1月":  time.January,
	"2月":  time.February,
	"3月":  time.March,
	"4月":  time.April,
	"5月":  time.May,
	"6月":  time.June,
	"7月":  time.July,
	"8月":  time.August,
	"9月":  time.September,
	"10月": time.October,
	"11月": time.November,
	"12月": time.December,
}

func shouldSkipFastPath(message string) bool {
	complexTokens := []string{"并且", "然后", "同时", "生成", "导出", "报表", "pdf", "PDF", "分析原因", "对比", "明细", "下载", "总结"}
	for _, token := range complexTokens {
		if strings.Contains(message, token) {
			return true
		}
	}
	return false
}

func resolveChineseTimeRange(label string) (time.Time, time.Time, bool) {
	now := time.Now()
	loc := now.Location()
	y, m, _ := now.Date()

	// 先检查具体月份（如 "五月" / "5月" / "五月份" / "五月份"）
	monthLabel := strings.TrimSuffix(label, "份")
	if targetMonth, ok := chineseMonthMap[monthLabel]; ok {
		// 默认当前年。如果指定的月份已过去，可能是上一年
		year := y
		if targetMonth > m {
			year = y - 1
		}
		start := time.Date(year, targetMonth, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 1, 0), true
	}

	switch label {
	case "今天":
		start := time.Date(y, m, now.Day(), 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 0, 1), true
	case "本月", "当前", "现在":
		start := time.Date(y, m, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 1, 0), true
	case "上个月", "上月":
		start := time.Date(y, m, 1, 0, 0, 0, 0, loc).AddDate(0, -1, 0)
		return start, start.AddDate(0, 1, 0), true
	case "本季度", "这个季度":
		qm := time.Month(((int(m)-1)/3)*3 + 1)
		start := time.Date(y, qm, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 3, 0), true
	case "上个季度", "上季度":
		qm := time.Month(((int(m)-1)/3)*3 + 1)
		start := time.Date(y, qm, 1, 0, 0, 0, 0, loc).AddDate(0, -3, 0)
		return start, start.AddDate(0, 3, 0), true
	case "今年":
		start := time.Date(y, 1, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(1, 0, 0), true
	case "去年":
		start := time.Date(y-1, 1, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(1, 0, 0), true
	}
	return time.Time{}, time.Time{}, false
}

func renderText2SQLTemplate(tpl, employee string, start, end time.Time) (string, []interface{}) {
	args := []interface{}{}
	replacements := []struct {
		key string
		val interface{}
	}{
		{":employee_name", employee},
		{":start_time", start.Format("2006-01-02 15:04:05")},
		{":end_time", end.Format("2006-01-02 15:04:05")},
	}
	query := tpl
	for _, r := range replacements {
		if strings.Contains(query, r.key) {
			query = strings.ReplaceAll(query, r.key, "?")
			args = append(args, r.val)
		}
	}
	return query, args
}

func isReadOnlyTemplateSQL(sql string) bool {
	upper := strings.ToUpper(strings.TrimSpace(sql))
	if !(strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "WITH")) {
		return false
	}
	for _, kw := range []string{"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE", "TRUNCATE", "REPLACE", "GRANT", "REVOKE"} {
		if strings.Contains(upper, kw) {
			return false
		}
	}
	return true
}

func normalizeDBValue(v interface{}) interface{} {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	if v == nil {
		return 0
	}
	return v
}

func parseText2SQLTemplateIntent(message string) (text2sqlTemplateIntent, bool) {
	m := strings.TrimSpace(message)
	metric := ""
	if strings.Contains(m, "销售额") || strings.Contains(m, "金额") {
		metric = "销售额"
	} else if strings.Contains(m, "多少单") || strings.Contains(m, "订单数") || strings.Contains(m, "订单数量") || strings.Contains(m, "订单量") || strings.Contains(m, "卖了") || strings.Contains(m, "出了") || strings.Contains(m, "出单") || strings.Contains(m, "销量") || strings.Contains(m, "下单") {
		metric = "订单数"
	}
	if metric == "" {
		return text2sqlTemplateIntent{}, false
	}

	// 从消息中提取时间范围（支持具体月份如 "五月"、"五月份"、"5月"）
	timeRange := ""
	timeTokens := []string{"上个季度", "上季度", "本季度", "这个季度", "上个月", "上月", "本月", "当前", "现在", "今天", "今年", "去年"}
	// 添加所有月份（一月~十二月 + 1月~12月 + 一月份~十二月份）
	for monthName := range chineseMonthMap {
		timeTokens = append(timeTokens, monthName, monthName+"份")
	}
	// 去重后匹配
	seen := map[string]bool{}
	for _, token := range timeTokens {
		if seen[token] {
			continue
		}
		seen[token] = true
		if strings.Contains(m, token) {
			timeRange = token
			break
		}
	}
	if timeRange == "" {
		timeRange = "当前"
	}

	// 用月份模式动态构建正则中间的可选分组（按长度降序，避免短匹配吃掉长匹配）
	var monthNames []string
	for monthName := range chineseMonthMap {
		monthNames = append(monthNames, monthName)
	}
	sort.Slice(monthNames, func(i, j int) bool {
		return len([]rune(monthNames[i])) > len([]rune(monthNames[j]))
	})
	monthPattern := ""
	seenMonth := map[string]bool{}
	for _, monthName := range monthNames {
		// 每个月份生成两个变体：原词 和 原词+"份"，长变体优先
		variants := []string{monthName + "份", monthName}
		for _, v := range variants {
			if seenMonth[v] {
				continue
			}
			seenMonth[v] = true
			if monthPattern != "" {
				monthPattern += "|"
			}
			monthPattern += regexp.QuoteMeta(v)
		}
	}
	// 时间范围后可选 "的"，再跟动作词
	re := regexp.MustCompile(fmt.Sprintf(`([\p{Han}]{2,4})(?:%s|当前|现在|本月|上月|上个季度|上季度|这个季度)(?:的)?(?:卖了|卖|出了|出|订单|下单|销售额|业绩|多少单|出单|订单数|订单数量|销量)`, monthPattern))
	match := re.FindStringSubmatch(m)
	if len(match) < 2 {
		return text2sqlTemplateIntent{}, false
	}
	return text2sqlTemplateIntent{Employee: match[1], Metric: metric, TimeRange: timeRange}, true
}
