package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/alaikis/opentether/internal/embedding"
	"github.com/alaikis/opentether/internal/llm"
	"github.com/alaikis/opentether/internal/models"
)

type SubTask struct {
	Index        int                    `json:"index"`
	Parent       int                    `json:"parent"`
	Depth        int                    `json:"depth"`
	Label        string                 `json:"label"`
	Query        string                 `json:"query"`
	Dependencies []int                  `json:"dependencies,omitempty"`
	Status       string                 `json:"status"`
	Result       string                 `json:"result,omitempty"`
	SkillUsed    string                 `json:"skill_used,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Children     []*SubTask             `json:"children,omitempty"`
}

type MultiTaskPlan struct {
	Original   string    `json:"original"`
	SubTasks   []SubTask `json:"sub_tasks"`
	TotalSteps int       `json:"total_steps"`
	IsTree     bool      `json:"is_tree"`
}

type MultiTaskResult struct {
	Plan    *MultiTaskPlan         `json:"plan"`
	Summary string                 `json:"summary"`
	Data    map[string]interface{} `json:"data"`
}

func BuildMultiTaskPlan(message string) *MultiTaskPlan {
	parts := SplitMultiPartQuestions(message)
	if len(parts) <= 1 {
		return nil
	}
	tree := detectTaskTree(message, parts)
	if tree != nil {
		plan := &MultiTaskPlan{Original: message, SubTasks: tree, IsTree: true}
		plan.TotalSteps = countTreeTasks(tree)
		return plan
	}
	deduped, _ := deduplicateSubQueries(parts)
	tasks := make([]SubTask, 0, len(deduped))
	seen := map[string]bool{}
	for i, q := range deduped {
		hash := taskHash(q)
		if seen[hash] {
			continue
		}
		seen[hash] = true
		tasks = append(tasks, SubTask{Index: i, Parent: -1, Depth: 0, Label: ExtractTaskLabel(q), Query: q, Status: "pending"})
	}
	for i := range tasks {
		tasks[i].Index = i
	}
	return &MultiTaskPlan{Original: message, SubTasks: tasks, TotalSteps: len(tasks)}
}

func isTrendMessage(message string) bool {
	lower := strings.ToLower(message)
	trendKeywords := []string{"趋势", "每月", "按月", "月份", "月度", "折线图", "柱状图", "变化图", "趋势图"}
	for _, kw := range trendKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func (e *AgentEngine) BuildMultiTaskPlanWithLLM(ctx context.Context, message string) *MultiTaskPlan {
	parts := SplitMultiPartQuestions(message)
	if len(parts) <= 1 {
		return nil
	}
	if isTrendMessage(message) {
		return nil
	}
	if e != nil && e.providers != nil {
		if provider, err := e.providers.GetActiveProvider(); err == nil && provider != nil {
			client, err := llm.NewClient(provider)
			if err == nil {
				prompt := fmt.Sprintf(`将用户的多部分问题拆分为独立的子任务。只返回 JSON 数组，不要解释。
用户问题: %s
已初步拆分为: %v
输出格式: [{"query":"子任务1","depends_on":[]},{"query":"子任务2","depends_on":[0]}]`, message, parts)
				resp, err := llm.ChatCompletionWithRetry(client, ctx, llm.ChatRequest{
					Model:          provider.Model,
					Messages:       []llm.Message{{Role: "user", Content: prompt}},
					MaxTokens:      1024,
					Temperature:    0.1,
					ResponseFormat: &llm.ResponseFormat{Type: "json_object"},
				}, 2)
				if err == nil && resp != nil && strings.TrimSpace(resp.Content) != "" {
					var llmParts []struct {
						Query     string `json:"query"`
						DependsOn []int  `json:"depends_on"`
					}
					if json.Unmarshal([]byte(resp.Content), &llmParts) == nil && len(llmParts) >= 2 {
						tasks := make([]SubTask, 0, len(llmParts))
						for i, p := range llmParts {
							if strings.TrimSpace(p.Query) == "" {
								continue
							}
							tasks = append(tasks, SubTask{Index: i, Parent: -1, Depth: 0, Label: ExtractTaskLabel(p.Query), Query: p.Query, Status: "pending", Dependencies: p.DependsOn})
						}
						if len(tasks) >= 2 {
							plan := &MultiTaskPlan{Original: message, SubTasks: tasks, IsTree: false}
							plan.TotalSteps = len(tasks)
							return plan
						}
					}
				}
			}
		}
	}
	return BuildMultiTaskPlan(message)
}

func detectTaskTree(message string, parts []string) []SubTask {
	if len(parts) < 2 {
		return nil
	}
	lowerMsg := strings.ToLower(message)
	sequentialKeywords := []string{"然后", "接着", "之后", "再", "然后分析", "然后统计"}
	parallelKeywords := []string{"同时", "分别", "各自", "以及"}
	hasSequential := false
	hasParallel := false
	for _, kw := range sequentialKeywords {
		if strings.Contains(lowerMsg, kw) {
			hasSequential = true
			break
		}
	}
	for _, kw := range parallelKeywords {
		if strings.Contains(lowerMsg, kw) {
			hasParallel = true
			break
		}
	}
	if !hasSequential && !hasParallel {
		return nil
	}
	tasks := make([]SubTask, 0, len(parts))
	for i, part := range parts {
		deps := []int{}
		if hasSequential && !hasParallel && i > 0 {
			deps = []int{i - 1}
		}
		tasks = append(tasks, SubTask{Index: i, Parent: -1, Depth: 0, Label: ExtractTaskLabel(part), Query: part, Status: "pending", Dependencies: deps})
	}
	return tasks
}

func countTreeTasks(tasks []SubTask) int {
	count := len(tasks)
	for i := range tasks {
		count += len(tasks[i].Children)
	}
	return count
}

func (e *AgentEngine) ExecuteMultiTaskPlan(ctx context.Context, user *UserContext, plan *MultiTaskPlan, conv *models.Conversation, progressFn func(int, int, string, string)) (*MultiTaskResult, error) {
	if plan == nil || len(plan.SubTasks) == 0 {
		return nil, nil
	}
	total := plan.TotalSteps
	maxParallel := e.maxParallelism
	if maxParallel <= 0 {
		maxParallel = 5
	}
	sem := make(chan struct{}, maxParallel)
	if plan.IsTree {
		return e.executeTaskTree(ctx, user, plan, conv, progressFn)
	}
	data := map[string]interface{}{"type": "multi_task", "plan": plan, "steps": []map[string]interface{}{}}
	results := make([]string, total)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	completed := make([]bool, total)
	ready := func(deps []int) bool {
		for _, d := range deps {
			if d >= 0 && d < total && !completed[d] {
				return false
			}
		}
		return true
	}

	allDone := false
	for !allDone {
		allDone = true
		running := 0
		for i := 0; i < total; i++ {
			task := &plan.SubTasks[i]
			if task.Status != "pending" || !ready(task.Dependencies) {
				allDone = false
				continue
			}
			if running >= maxParallel {
				allDone = false
				break
			}
			running++
			wg.Add(1)
			go func(t *SubTask) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				t.Status = "running"
				if progressFn != nil {
					progressFn(t.Index, total, t.Label, "running")
				}
				resp, err := e.ExecuteLoop(ctx, user, t.Query, conv.ID)
				mu.Lock()
				defer mu.Unlock()
				completed[t.Index] = true
				if err != nil {
					t.Status = "failed"
					t.Error = err.Error()
					results[t.Index] = fmt.Sprintf("❌ [%s] %s: %v", t.Label, t.Query, err)
				} else {
					t.Status = "completed"
					t.Result = resp.Message
					t.SkillUsed = resp.SkillUsed
					t.Data = resp.Data
					results[t.Index] = fmt.Sprintf("✅ [%s] %s", t.Label, resp.Message)
					if e.validateSubTaskQuality(t.Query, plan.Original) {
						e.dispatchNestedTasks(ctx, user, t, plan, conv, progressFn)
					}
				}
				if progressFn != nil {
					progressFn(t.Index, total, t.Label, t.Status)
				}
			}(task)
		}
		wg.Wait()
	}

	for _, t := range plan.SubTasks {
		data["steps"] = append(data["steps"].([]map[string]interface{}), map[string]interface{}{
			"label": t.Label, "query": t.Query, "status": t.Status, "result": t.Result, "skill_used": t.SkillUsed, "error": t.Error,
		})
	}
	summary := buildMultiTaskSummary(plan, results)
	return &MultiTaskResult{Plan: plan, Summary: summary, Data: data}, nil
}

func (e *AgentEngine) executeTaskTree(ctx context.Context, user *UserContext, plan *MultiTaskPlan, conv *models.Conversation, progressFn func(int, int, string, string)) (*MultiTaskResult, error) {
	data := map[string]interface{}{"type": "multi_task", "plan": plan, "steps": []map[string]interface{}{}}
	results := make([]string, len(plan.SubTasks))
	completed := make([]bool, len(plan.SubTasks))

	for i := 0; i < len(plan.SubTasks); i++ {
		task := &plan.SubTasks[i]
		if task.Status != "pending" {
			continue
		}
		task.Status = "running"
		if progressFn != nil {
			progressFn(i, len(plan.SubTasks), task.Label, "running")
		}
		resp, err := e.ExecuteLoop(ctx, user, task.Query, conv.ID)
		completed[i] = true
		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			results[i] = fmt.Sprintf("❌ [%s] %s: %v", task.Label, task.Query, err)
		} else {
			task.Status = "completed"
			task.Result = resp.Message
			task.SkillUsed = resp.SkillUsed
			task.Data = resp.Data
			results[i] = fmt.Sprintf("✅ [%s] %s", task.Label, resp.Message)
			e.dispatchNestedTasks(ctx, user, task, plan, conv, progressFn)
		}
		if progressFn != nil {
			progressFn(i, len(plan.SubTasks), task.Label, task.Status)
		}
	}

	for _, t := range plan.SubTasks {
		data["steps"] = append(data["steps"].([]map[string]interface{}), map[string]interface{}{
			"label": t.Label, "query": t.Query, "status": t.Status, "result": t.Result, "skill_used": t.SkillUsed, "error": t.Error,
		})
	}
	summary := buildMultiTaskSummary(plan, results)
	return &MultiTaskResult{Plan: plan, Summary: summary, Data: data}, nil
}

func (e *AgentEngine) dispatchNestedTasks(ctx context.Context, user *UserContext, parent *SubTask, plan *MultiTaskPlan, conv *models.Conversation, progressFn func(int, int, string, string)) {
	if parent == nil || parent.Result == "" {
		return
	}
	items := extractListItems(parent.Result)
	if len(items) <= 1 || len(items) > 5 {
		return
	}
	contextWords := extractContextWords(parent.Query)
	for i, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || len(item) > 30 {
			continue
		}
		nestedQuery := fmt.Sprintf("%s%s%s", strings.Join(contextWords, ""), item, strings.Join(extractMetricFromQuery(parent.Query), ""))
		child := SubTask{
			Index: len(plan.SubTasks), Parent: parent.Index, Depth: parent.Depth + 1,
			Label: "子任务: " + item, Query: nestedQuery, Status: "running",
			Dependencies: []int{parent.Index},
		}
		plan.SubTasks = append(plan.SubTasks, child)
		plan.TotalSteps++
		if progressFn != nil {
			progressFn(child.Index-child.Index+i, plan.TotalSteps, child.Label, "running")
		}
		resp, err := e.ExecuteLoop(ctx, user, child.Query, conv.ID)
		idx := len(plan.SubTasks) - 1
		if err != nil {
			plan.SubTasks[idx].Status = "failed"
			plan.SubTasks[idx].Error = err.Error()
		} else {
			plan.SubTasks[idx].Status = "completed"
			plan.SubTasks[idx].Result = resp.Message
			plan.SubTasks[idx].SkillUsed = resp.SkillUsed
			plan.SubTasks[idx].Data = resp.Data
		}
		if progressFn != nil {
			progressFn(idx, plan.TotalSteps, child.Label, plan.SubTasks[idx].Status)
		}
	}
}

func extractListItems(result string) []string {
	lines := strings.Split(result, "\n")
	items := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, value, ok := splitKV(line); ok {
			items = append(items, value)
		} else if strings.Count(line, " ") <= 3 && len([]rune(line)) <= 20 {
			items = append(items, line)
		}
	}
	return items
}

func extractContextWords(query string) []string {
	words := []string{}
	categories := []string{"北京", "上海", "广州", "深圳", "杭州", "成都", "武汉", "西安", "南京", "重庆",
		"电子产品", "服装", "食品", "家居", "美妆", "母婴", "运动", "图书", "汽车", "医药",
		"线上", "线下", "门店", "电商", "批发", "零售", "部门", "团队", "区域", "分公司"}
	for _, cat := range categories {
		if strings.Contains(query, cat) {
			words = append(words, cat)
		}
	}
	if len(words) == 0 {
		fields := strings.Fields(query)
		for _, f := range fields {
			runes := []rune(f)
			if len(runes) >= 2 && len(runes) <= 8 {
				words = append(words, f)
			}
		}
	}
	return words
}

func extractMetricFromQuery(query string) []string {
	metrics := []string{"销售额", "销量", "订单量", "利润", "成本", "收入", "增长", "趋势", "占比", "排名", "用户数", "访问量", "转化率", "客单价", "复购率"}
	found := []string{}
	for _, m := range metrics {
		if strings.Contains(query, m) {
			found = append(found, m)
		}
	}
	if len(found) == 0 {
		return nil
	}
	return found
}

func splitKV(line string) (string, string, bool) {
	sep := []string{": ", "：", " - ", "|\t"}
	for _, s := range sep {
		parts := strings.SplitN(line, s, 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" && value != "" {
				return key, value, true
			}
		}
	}
	return "", "", false
}

func deduplicateSubQueries(parts []string) ([]string, []string) {
	result := make([]string, 0, len(parts))
	labels := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		isDup := false
		for j, existing := range result {
			if querySimilarity(p, existing) > 0.8 {
				result[j] = mergeQueries(existing, p)
				isDup = true
				break
			}
		}
		if !isDup {
			result = append(result, p)
			labels = append(labels, "")
		}
	}
	return result, labels
}

func querySimilarity(a, b string) float64 {
	wordsA := tokenize(a)
	wordsB := tokenize(b)
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0
	}
	intersection := 0
	for _, w := range wordsA {
		if len(w) < 2 {
			continue
		}
		if stringInSlice(w, wordsB) {
			intersection++
		}
	}
	return float64(intersection*2) / float64(len(wordsA)+len(wordsB))
}

func tokenize(s string) []string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r >= 0x4e00 {
			return r
		}
		return ' '
	}, s)
	return strings.Fields(s)
}

func stringInSlice(s string, slice []string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func mergeQueries(a, b string) string {
	if len(b) > len(a) {
		return b
	}
	return a
}

func taskHash(q string) string {
	q = strings.TrimSpace(strings.ToLower(q))
	if q == "" {
		return ""
	}
	fields := strings.Fields(q)
	if len(fields) > 6 {
		fields = fields[:6]
	}
	return strings.Join(fields, " ")
}

func ExtractTaskLabel(q string) string {
	q = strings.TrimSpace(q)
	if len([]rune(q)) > 12 {
		return string([]rune(q)[:12]) + "..."
	}
	return q
}

func buildMultiTaskSummary(plan *MultiTaskPlan, results []string) string {
	sb := strings.Builder{}
	sb.WriteString("📋 多任务分析完成 (" + fmt.Sprintf("%d/%d", completedCount(plan), plan.TotalSteps) + ")\n\n")
	for i, t := range plan.SubTasks {
		indent := strings.Repeat("  ", t.Depth)
		if i < len(results) && results[i] != "" {
			skillInfo := ""
			if t.SkillUsed != "" {
				skillInfo = fmt.Sprintf(" [Skill: %s]", t.SkillUsed)
			}
			sb.WriteString(indent + results[i] + skillInfo)
		} else {
			skillInfo := ""
			if t.SkillUsed != "" {
				skillInfo = fmt.Sprintf(" [Skill: %s]", t.SkillUsed)
			}
			sb.WriteString(fmt.Sprintf("%s⬜ [%s] %s%s", indent, t.Label, t.Status, skillInfo))
		}
		sb.WriteString("\n")
	}
	if insight := buildAggregatedInsight(plan); insight != "" {
		sb.WriteString("\n---\n")
		sb.WriteString(insight)
	}
	return sb.String()
}

func completedCount(plan *MultiTaskPlan) int {
	count := 0
	for _, t := range plan.SubTasks {
		if t.Status == "completed" {
			count++
		}
	}
	return count
}

func buildAggregatedInsight(plan *MultiTaskPlan) string {
	metrics := map[string]string{}
	for _, t := range plan.SubTasks {
		if t.Status != "completed" || t.Result == "" {
			continue
		}
		if strings.Contains(t.Label, "订单") || strings.Contains(t.Label, "销售") || strings.Contains(t.Label, "利润") || strings.Contains(t.Label, "成本") {
			metrics[t.Label] = extractNumberFromResult(t.Result)
		}
	}
	metricKeys := make([]string, 0, len(metrics))
	for k := range metrics {
		metricKeys = append(metricKeys, k)
	}
	sort.Strings(metricKeys)
	if len(metricKeys) > 0 {
		parts := make([]string, 0, len(metricKeys))
		for _, k := range metricKeys {
			parts = append(parts, fmt.Sprintf("%s: %s", k, metrics[k]))
		}
		return "📊 关键指标汇总：\n" + strings.Join(parts, "；")
	}
	return ""
}

func extractNumberFromResult(result string) string {
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		key, value, ok := splitKV(line)
		if ok && len(value) > 0 && (value[0] >= '0' && value[0] <= '9') {
			return key + ": " + value
		}
	}
	return result
}

func (e *AgentEngine) validateSubTaskQuality(subQuery, originalQuery string) bool {
	if e == nil || e.db == nil {
		return true
	}
	emb, err := embedding.Create("", nil)
	if err != nil {
		return true
	}
	subVec, err := emb.Embed(strings.ToLower(subQuery))
	if err != nil {
		return true
	}
	origVec, err := emb.Embed(strings.ToLower(originalQuery))
	if err != nil {
		return true
	}
	score := cosineSimilarity(subVec, origVec)
	return score >= 0.3
}
