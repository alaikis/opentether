package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"github.com/alaikis/opentether/internal/templating"
	"github.com/alaikis/opentether/internal/text2sql"
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

func (e *AgentEngine) fastPathCacheKey(user *UserContext, message string) string {
	userID := ""
	dataSourceID := ""
	if user != nil {
		userID = user.UserID
		if user.Context != nil {
			dataSourceID, _ = user.Context["data_source_id"].(string)
		}
	}
	return userID + "|" + dataSourceID + "|" + strings.TrimSpace(message)
}

func (e *AgentEngine) getFastPathCache(key, conversationID string) (*ChatResponse, bool) {
	if e == nil || key == "" {
		return nil, false
	}
	e.fastCacheMu.Lock()
	entry, ok := e.fastCache[key]
	if ok && time.Now().Before(entry.ExpiresAt) && entry.Resp != nil {
		resp := *entry.Resp
		resp.ConversationID = conversationID
		e.fastCacheMu.Unlock()
		return &resp, true
	}
	if ok {
		delete(e.fastCache, key)
	}
	e.fastCacheMu.Unlock()
	return nil, false
}

func (e *AgentEngine) setFastPathCache(key string, resp *ChatResponse, ttl time.Duration) {
	if e == nil || key == "" || resp == nil || ttl <= 0 {
		return
	}
	copyResp := *resp
	entry := fastCacheEntry{Resp: &copyResp, ExpiresAt: time.Now().Add(ttl)}
	e.fastCacheMu.Lock()
	e.fastCache[key] = entry
	e.fastCacheMu.Unlock()
	e.finishFastPathInflight(key, entry)
}

func (e *AgentEngine) beginFastPathInflight(key string) (chan fastCacheEntry, bool) {
	if e == nil || key == "" {
		return nil, true
	}
	e.fastCacheMu.Lock()
	if ch, ok := e.fastInflight[key]; ok {
		e.fastCacheMu.Unlock()
		return ch, false
	}
	ch := make(chan fastCacheEntry, 1)
	e.fastInflight[key] = ch
	e.fastCacheMu.Unlock()
	return ch, true
}

func (e *AgentEngine) finishFastPathInflight(key string, entry fastCacheEntry) {
	if e == nil || key == "" {
		return
	}
	e.fastCacheMu.Lock()
	ch := e.fastInflight[key]
	delete(e.fastInflight, key)
	e.fastCacheMu.Unlock()
	if ch != nil {
		ch <- entry
		close(ch)
	}
}

func (e *AgentEngine) TryFastPath(ctx context.Context, user *UserContext, message, conversationID string) (*ChatResponse, bool, error) {
	e.metrics.IncFastPath()
	message = stripEnvironmentDetailsFromPrompt(message)
	start := time.Now()
	routeMS := int64(0)
	annotate := func(resp *ChatResponse, source string) *ChatResponse {
		if resp == nil {
			return resp
		}
		if resp.Data == nil {
			resp.Data = map[string]interface{}{}
		}
		resp.Data["fast_path"] = true
		resp.Data["fast_path_source"] = source
		resp.Data["duration_ms"] = time.Since(start).Milliseconds()
		resp.Data["route_ms"] = routeMS
		return resp
	}
	if shouldSkipFastPath(message) {
		return nil, false, nil
	}
	cacheKey := e.fastPathCacheKey(user, message)
	if resp, ok := e.getFastPathCache(cacheKey, conversationID); ok {
		return annotate(resp, "cache"), true, nil
	}
	if ch, owner := e.beginFastPathInflight(cacheKey); !owner {
		select {
		case entry := <-ch:
			if entry.Resp != nil {
				resp := *entry.Resp
				resp.ConversationID = conversationID
				return annotate(&resp, "singleflight"), true, entry.Err
			}
		case <-ctx.Done():
			return nil, false, ctx.Err()
		}
		return nil, false, nil
	} else {
		defer func() {
			if _, ok := e.getFastPathCache(cacheKey, conversationID); !ok {
				e.finishFastPathInflight(cacheKey, fastCacheEntry{})
			}
		}()
	}
	if resp, ok := tryLocalFastAnswer(message, conversationID); ok {
		return annotate(resp, "local"), true, nil
	}
	// 扩展时间范围：如果消息含时间表达，直接走 text2sql
	if tr, ok := parseExtendedTimeRange(message); ok && tr.IsRange {
		if resp, ok := e.tryText2SQLRangeFastPath(ctx, user, message, tr, conversationID); ok {
			resp = annotate(resp, "range")
			e.setFastPathCache(cacheKey, resp, 120*time.Second)
			return resp, true, nil
		}
	}

	routeStart := time.Now()
	route := e.routeFastPath(message)
	if route.Route == "" && isChartDataQuery(message) {
		route = fastPathRoute{Route: "fast_text2sql", Intent: "text2sql", Confidence: 0.9, Reason: "图表类数据查询优先 Text2SQL"}
	}
	if route.Route == "" {
		route = e.routeFastPathWithSmallModel(ctx, user, message)
	}
	routeMS = time.Since(routeStart).Milliseconds()
	if route.Route == "fast_text2sql" || route.Intent == "text2sql" {
		if resp, ok := e.tryText2SQLApprovedTemplateFastPath(ctx, user, message, conversationID); ok {
			resp = annotate(resp, "approved_template")
			e.setFastPathCache(cacheKey, resp, 120*time.Second)
			return resp, true, nil
		}
		if resp, ok := e.tryText2SQLTemplateFastPath(ctx, user, message, conversationID); ok {
			resp = annotate(resp, "text2sql")
			e.setFastPathCache(cacheKey, resp, 120*time.Second)
			return resp, true, nil
		}
	}
	if route.Route == "fast_chat" || (route.Route == "" && isSimpleChat(message)) {
		resp, err := e.executeFastChat(ctx, user, message, conversationID)
		if err == nil && resp != nil {
			return annotate(resp, "chat"), true, nil
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
	answer, err := e.providers.CallLLMWithFallback(ctx, provider, prompt)
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
	return nil, false
}

func (e *AgentEngine) executeFastChat(ctx context.Context, user *UserContext, message, conversationID string) (*ChatResponse, error) {
	provider, err := e.providers.GetActiveProvider()
	if err != nil || provider == nil {
		return nil, err
	}
	prompt := fmt.Sprintf("请用中文简洁回答，不要使用工具。用户问题：%s", message)
	answer, err := e.providers.CallLLMWithFallback(ctx, provider, prompt)
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
	if strings.Count(m, "？")+strings.Count(m, "?") > 0 {
		return false
	}
	return strings.Contains(m, "什么") || strings.Contains(m, "为什么") || strings.Contains(m, "如何") || strings.Contains(m, "怎么") || strings.Contains(m, "解释")
}

func (e *AgentEngine) tryText2SQLApprovedTemplateFastPath(ctx context.Context, user *UserContext, message, conversationID string) (*ChatResponse, bool) {
	if user == nil || e == nil || e.db == nil {
		return nil, false
	}
	// 检查运行时记忆中是否有已审批的模板
	dataSourceID := ""
	if user.Context != nil {
		dataSourceID, _ = user.Context["data_source_id"].(string)
	}
	if dataSourceID == "" {
		return nil, false
	}
	// 用原始问题作为 key 查询模板
	var mems []struct{ Content string }
	q := e.db.Model(&models.SkillRuntimeMemory{}).
		Select("content").
		Where("data_source_id = ? AND type = ? AND source = ?", dataSourceID, "text2sql_template", "admin").
		Order("confidence DESC, updated_at DESC").Limit(20)
	if err := q.Find(&mems).Error; err != nil || len(mems) == 0 {
		return nil, false
	}
	for _, mem := range mems {
		var tpl struct {
			SQLTemplate string `json:"SQLTemplate"`
			Intent      string `json:"intent"`
		}
		if json.Unmarshal([]byte(mem.Content), &tpl) != nil || tpl.SQLTemplate == "" {
			continue
		}
		if !templateMatchesIntent(message, tpl.Intent) {
			continue
		}
		renderedSQL, ok := renderSQLTemplate(tpl.SQLTemplate, message, time.Now())
		if !ok {
			continue
		}
		if resp, ok := e.executeRenderedSQLTemplate(ctx, user, message, renderedSQL, dataSourceID, conversationID); ok {
			return resp, true
		}
	}
	return nil, false
}

func templateMatchesIntent(message, intent string) bool {
	if strings.TrimSpace(intent) == "" {
		return true
	}
	lower := strings.ToLower(message)
	for _, token := range strings.Split(intent, ",") {
		token = strings.TrimSpace(strings.ToLower(token))
		if token == "" {
			continue
		}
		if strings.Contains(lower, token) {
			return true
		}
		if len([]rune(token)) >= 2 {
			runes := []rune(token)
			for i := 0; i <= len(runes)-2; i++ {
				sub := string(runes[i : i+2])
				if strings.Contains(lower, sub) {
					return true
				}
			}
		}
	}
	return false
}

func renderSQLTemplate(tpl string, message string, now time.Time) (string, bool) {
	tr, ok := parseExtendedTimeRange(message)
	if !ok {
		return "", false
	}
	vars := map[string]string{
		"start_date":    tr.Start.Format("2006-01-02"),
		"end_date":      tr.End.Format("2006-01-02"),
		"current_year":  fmt.Sprintf("%d", now.Year()),
		"current_month": fmt.Sprintf("%02d", int(now.Month())),
	}
	for k, v := range extractTemplateVariables(message) {
		vars[k] = v
	}
	out := tpl
	for k, v := range vars {
		v = strings.ReplaceAll(v, "'", "''")
		out = strings.ReplaceAll(out, "{{"+k+"}}", "'"+v+"'")
		out = strings.ReplaceAll(out, "{{ "+k+" }}", "'"+v+"'")
	}
	if strings.Contains(out, "{{") || strings.Contains(out, "}}") {
		return "", false
	}
	return out, true
}

func extractTemplateVariables(message string) map[string]string {
	vars := map[string]string{}
	if m := regexp.MustCompile(`^\s*([\p{Han}]{2,4})`).FindStringSubmatch(message); len(m) > 1 {
		vars["employee_name"] = m[1]
	}
	patterns := map[string]string{
		"country":     `国家[:：\s]+([^，,；;\s]+)`,
		"buyer":       `买家[:：\s]+([^，,；;\s]+)`,
		"mpn":         `(?i)mpn[:：\s]+([^，,；;\s]+)`,
		"sku":         `(?i)sku[:：\s]+([^，,；;\s]+)`,
		"title_regex": `标题(?:正则)?[:：\s]+([^，,；;]+)`,
	}
	for key, pattern := range patterns {
		if m := regexp.MustCompile(pattern).FindStringSubmatch(message); len(m) > 1 {
			vars[key] = strings.TrimSpace(m[1])
		}
	}
	return vars
}

func (e *AgentEngine) executeRenderedSQLTemplate(ctx context.Context, user *UserContext, message, sqlText, dataSourceID, conversationID string) (*ChatResponse, bool) {
	if e.externalDBPool == nil {
		return nil, false
	}
	extDB, err := e.externalDBPool.Get(ctx, dataSourceID)
	if err != nil {
		return nil, false
	}
	t2s := text2sql.NewWithExternalDB(e.db, nil, dataSourceID, extDB)
	if e.sqlAuditor != nil {
		t2s.SetAuditService(e.sqlAuditor)
	}
	result, err := t2s.ExecuteSQL(ctx, &text2sql.QueryRequest{Question: message, RawSQL: sqlText, DataSourceID: dataSourceID, UserID: user.UserID})
	if err != nil || result == nil || result.Error != "" {
		return nil, false
	}
	return &ChatResponse{Message: formatQueryResult(result), ConversationID: conversationID, SkillUsed: "fast_text2sql_approved_template", Data: map[string]interface{}{"sql": result.SQL, "columns": result.Columns, "rows": result.Rows, "row_count": result.RowCount, "template_rendered": true, "generated_at": time.Now().Format(time.RFC3339), "freshness": "live_query"}}, true
}

func (e *AgentEngine) tryText2SQLTemplateFastPath(ctx context.Context, user *UserContext, message, conversationID string) (*ChatResponse, bool) {
	if user == nil {
		return nil, false
	}
	resp, err := e.executeText2SQL(message, user)
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

// tryText2SQLRangeFastPath handles extended time range queries.
func (e *AgentEngine) tryText2SQLRangeFastPath(ctx context.Context, user *UserContext, message string, tr timeRangeSpec, conversationID string) (*ChatResponse, bool) {
	if user == nil || e == nil {
		return nil, false
	}
	// 把时间和原始问题一起传给 text2sql，让 LLM 理解
	question := fmt.Sprintf("查询%s到%s期间的数据。原始问题: %s",
		tr.Start.Format("2006-01-02"),
		tr.End.Format("2006-01-02"),
		message)
	resp, err := e.executeText2SQL(question, user)
	if err != nil || resp == nil {
		return nil, false
	}
	if resp.Data != nil {
		if errMsg, ok := resp.Data["error"].(string); ok && errMsg != "" {
			return nil, false
		}
	}
	if resp.Message == "" {
		return nil, false
	}
	resp.ConversationID = conversationID
	resp.SkillUsed = "fast_text2sql_range"
	return resp, true
}

var chineseMonthMap = map[string]time.Month{
	"一月": time.January, "二月": time.February, "三月": time.March,
	"四月": time.April, "五月": time.May, "六月": time.June,
	"七月": time.July, "八月": time.August, "九月": time.September,
	"十月": time.October, "十一月": time.November, "十二月": time.December,
	"1月": time.January, "2月": time.February, "3月": time.March,
	"4月": time.April, "5月": time.May, "6月": time.June,
	"7月": time.July, "8月": time.August, "9月": time.September,
	"10月": time.October, "11月": time.November, "12月": time.December,
}

func isChartDataQuery(message string) bool {
	m := strings.ToLower(message)
	if strings.Contains(m, "excel") || strings.Contains(m, "xlsx") || strings.Contains(message, "导出") || strings.Contains(message, "下载") {
		return false
	}
	return strings.Contains(message, "图表") || strings.Contains(message, "折线图") || strings.Contains(message, "柱状图") || strings.Contains(message, "趋势图") || strings.Contains(message, "变化图")
}

func shouldSkipFastPath(message string) bool {
	complexTokens := []string{"并且", "然后", "同时", "导出", "pdf", "PDF", "分析原因", "下载", "总结"}
	for _, token := range complexTokens {
		if strings.Contains(message, token) {
			return true
		}
	}
	return false
}
