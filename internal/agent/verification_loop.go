package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/alaikis/opentether/internal/llm"
	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

type VerificationLoop struct {
	db          *gorm.DB
	mu          sync.RWMutex
	rules       map[string]*VerifyRule
	strategies  map[string]RetryStrategy
	history     []VerificationRecord
	maxHistory  int
}

type VerifyRule struct {
	ID          string
	ToolName    string
	CheckType   string
	Pattern     string
	IsRequired  bool
	Confidence  float64
	Description string
}

type RetryStrategy struct {
	ToolName        string
	MaxRetries      int
	BaseDelayMs     int
	BackoffMultiplier float64
	TimeoutMs       int
	Conditions      []RetryCondition
}

type RetryCondition struct {
	ErrorPattern string
	Action       string
	MaxRetries   int
}

type VerificationRecord struct {
	ID          string
	ToolName    string
	Input       string
	Output      string
	Passed      bool
	Error       string
	Retries     int
	LatencyMs   int64
	Timestamp   time.Time
}

type VerificationResult struct {
	Passed      bool
	Checks      []CheckResult
	ShouldRetry bool
	RetryAdvice string
}

type CheckResult struct {
	CheckType string
	Passed    bool
	Details   string
	Confidence float64
}

func NewVerificationLoop(db *gorm.DB) *VerificationLoop {
	v := &VerificationLoop{
		db:         db,
		rules:      make(map[string]*VerifyRule),
		strategies: make(map[string]RetryStrategy),
		history:    make([]VerificationRecord, 0, 100),
		maxHistory: 100,
	}
	v.loadDefaultRules()
	v.loadDefaultStrategies()
	return v
}

func (v *VerificationLoop) loadDefaultRules() {
	defaultRules := []*VerifyRule{
		{ID: "sql_syntax", ToolName: "text2sql", CheckType: "syntax", Pattern: `(?i)(syntax|错误|error)`, IsRequired: true, Confidence: 0.9},
		{ID: "sql_empty", ToolName: "text2sql", CheckType: "empty_result", Pattern: `(无数据|未找到|no (data|results)|empty)`, IsRequired: false, Confidence: 0.7},
		{ID: "json_valid", ToolName: "*", CheckType: "json_valid", Pattern: `^\s*[\[{]`, IsRequired: true, Confidence: 0.95},
		{ID: "error_marker", ToolName: "*", CheckType: "error_marker", Pattern: `(?i)(error|failed|exception|crash)`, IsRequired: false, Confidence: 0.85},
		{ID: "truncation", ToolName: "*", CheckType: "truncation", Pattern: `(省略|截断|truncated|\.\.\.)`, IsRequired: false, Confidence: 0.8},
	}
	for _, r := range defaultRules {
		v.rules[r.ID] = r
	}
}

func (v *VerificationLoop) loadDefaultStrategies() {
	v.strategies["text2sql"] = RetryStrategy{
		ToolName:           "text2sql",
		MaxRetries:         3,
		BaseDelayMs:        500,
		BackoffMultiplier:  2.0,
		TimeoutMs:          30000,
		Conditions: []RetryCondition{
			{ErrorPattern: "timeout", Action: "increase_timeout", MaxRetries: 2},
			{ErrorPattern: "syntax", Action: "rewrite_query", MaxRetries: 2},
			{ErrorPattern: "connection", Action: "retry_later", MaxRetries: 1},
		},
	}
	v.strategies["api_caller"] = RetryStrategy{
		ToolName:           "api_caller",
		MaxRetries:         2,
		BaseDelayMs:        1000,
		BackoffMultiplier:  1.5,
		TimeoutMs:          15000,
		Conditions: []RetryCondition{
			{ErrorPattern: "429", Action: "backoff", MaxRetries: 2},
			{ErrorPattern: "5[0-9]{2}", Action: "retry", MaxRetries: 1},
		},
	}
	v.strategies["default"] = RetryStrategy{
		ToolName:           "default",
		MaxRetries:         2,
		BaseDelayMs:        500,
		BackoffMultiplier:  2.0,
		TimeoutMs:          10000,
		Conditions:         []RetryCondition{},
	}
}

func (v *VerificationLoop) VerifyResult(toolName, input, output string) *VerificationResult {
	result := &VerificationResult{
		Passed: true,
		Checks: make([]CheckResult, 0),
	}

	for _, rule := range v.rules {
		if !v.matchesTool(rule.ToolName, toolName) {
			continue
		}

		check := v.executeCheck(rule, output)
		result.Checks = append(result.Checks, check)

		if rule.IsRequired && !check.Passed {
			result.Passed = false
		}
	}

	result.ShouldRetry = !result.Passed
	result.RetryAdvice = v.getRetryAdvice(toolName, result)

	v.recordResult(toolName, input, output, result)

	return result
}

func (v *VerificationLoop) VerifyWithLLM(toolName, input, output string, llmClient LLMJudgeClient) *VerificationResult {
	result := v.VerifyResult(toolName, input, output)

	if llmClient == nil || output == "" || len(output) < 50 {
		return result
	}

	judgePrompt := v.buildJudgePrompt(toolName, input, output)
	resp, err := llmClient.ChatCompletion(context.Background(), llm.ChatRequest{
		Model:       "default",
		Messages:    []llm.Message{{Role: "user", Content: judgePrompt}},
		MaxTokens:   500,
		Temperature: 0.1,
	})

	if err != nil {
		result.Checks = append(result.Checks, CheckResult{
			CheckType: "llm_judge",
			Passed:    true,
			Details:   "LLM judge unavailable, fallback to rules: " + err.Error(),
		})
		return result
	}

	judgment := v.parseLLMJudgment(resp.Content)
	result.Checks = append(result.Checks, CheckResult{
		CheckType:  "llm_judge",
		Passed:     judgment.Passed,
		Details:    judgment.Reason,
		Confidence: judgment.Confidence,
	})

	if !judgment.Passed && judgment.Confidence > 0.7 {
		result.Passed = false
		result.ShouldRetry = true
		result.RetryAdvice = judgment.Suggestion
	}

	return result
}

type LLMJudgment struct {
	Passed      bool
	Confidence  float64
	Reason      string
	Suggestion  string
	Quality     string
}

func (v *VerificationLoop) buildJudgePrompt(toolName, input, output string) string {
	return fmt.Sprintf(`你是一个严格的结果质量评审专家。请评审以下工具执行结果的质量。

工具类型: %s
用户输入: %s

执行结果:
%s

请从以下维度评审:
1. 结果是否正确回答了用户问题
2. 结果格式是否正确
3. 是否有明显的错误或遗漏
4. 结果是否完整

请以JSON格式返回评审结果:
{
  "passed": true/false,
  "confidence": 0.0-1.0,
  "reason": "评审理由",
  "suggestion": "改进建议(如果失败)",
  "quality": "excellent/good/acceptable/poor"
}

只返回JSON，不要其他内容。`, toolName, input, output)
}

func (v *VerificationLoop) parseLLMJudgment(content string) *LLMJudgment {
	content = strings.TrimSpace(content)

	jsonMatch := regexp.MustCompile(`\{[\s\S]*\}`).FindString(content)
	if jsonMatch == "" {
		return &LLMJudgment{Passed: true, Confidence: 0.5, Reason: "无法解析LLM响应"}
	}

	var result struct {
		Passed      bool    `json:"passed"`
		Confidence  float64 `json:"confidence"`
		Reason      string  `json:"reason"`
		Suggestion  string  `json:"suggestion"`
		Quality     string  `json:"quality"`
	}

	if err := json.Unmarshal([]byte(jsonMatch), &result); err != nil {
		return &LLMJudgment{Passed: true, Confidence: 0.5, Reason: "JSON解析失败"}
	}

	return &LLMJudgment{
		Passed:      result.Passed,
		Confidence:  result.Confidence,
		Reason:      result.Reason,
		Suggestion:  result.Suggestion,
		Quality:     result.Quality,
	}
}

type LLMJudgeClient interface {
	ChatCompletion(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error)
}

func (v *VerificationLoop) matchesTool(pattern, toolName string) bool {
	if pattern == "*" {
		return true
	}
	return pattern == toolName
}

func (v *VerificationLoop) executeCheck(rule *VerifyRule, output string) CheckResult {
	re := regexp.MustCompile(rule.Pattern)
	matches := re.FindString(output)

	passed := true
	if rule.CheckType == "empty_result" || rule.CheckType == "error_marker" || rule.CheckType == "truncation" {
		passed = matches == ""
	} else if rule.CheckType == "syntax" || rule.CheckType == "json_valid" {
		passed = matches != ""
	}

	return CheckResult{
		CheckType:  rule.CheckType,
		Passed:     passed,
		Details:    fmt.Sprintf("pattern: %s, match: %q", rule.Pattern, matches),
		Confidence: rule.Confidence,
	}
}

func (v *VerificationLoop) getRetryAdvice(toolName string, result *VerificationResult) string {
	if result.Passed {
		return ""
	}

	var failedTypes []string
	for _, check := range result.Checks {
		if !check.Passed {
			failedTypes = append(failedTypes, check.CheckType)
		}
	}

	for _, check := range failedTypes {
		switch check {
		case "syntax":
			return "建议: 重写查询，使用更简单的 SQL 结构"
		case "empty_result":
			return "建议: 检查查询条件是否过于严格，尝试扩大搜索范围"
		case "timeout":
			return "建议: 优化查询性能或增加超时时间"
		case "error_marker":
			return "建议: 检查上游服务状态，可能需要重试"
		case "truncation":
			return "建议: 结果被截断，考虑分页获取"
		}
	}

	strategy := v.getStrategy(toolName)
	return fmt.Sprintf("建议重试，最多 %d 次", strategy.MaxRetries)
}

func (v *VerificationLoop) getStrategy(toolName string) *RetryStrategy {
	if s, ok := v.strategies[toolName]; ok {
		return &s
	}
	defaultStrat := v.strategies["default"]
	return &defaultStrat
}

func (v *VerificationLoop) GetRetryDelay(toolName, errorMsg string, attempt int) int {
	strategy := v.getStrategy(toolName)

	for _, cond := range strategy.Conditions {
		if strings.Contains(strings.ToLower(errorMsg), strings.ToLower(cond.ErrorPattern)) {
			if attempt < cond.MaxRetries {
				delay := strategy.BaseDelayMs
				for i := 0; i < attempt; i++ {
					delay = int(float64(delay) * strategy.BackoffMultiplier)
				}
				return delay
			}
		}
	}

	delay := strategy.BaseDelayMs
	for i := 0; i < attempt; i++ {
		delay = int(float64(delay) * strategy.BackoffMultiplier)
	}
	return delay
}

func (v *VerificationLoop) ShouldRetry(toolName, errorMsg string, attempts int) bool {
	strategy := v.getStrategy(toolName)
	if attempts >= strategy.MaxRetries {
		return false
	}

	for _, cond := range strategy.Conditions {
		if strings.Contains(strings.ToLower(errorMsg), strings.ToLower(cond.ErrorPattern)) {
			return attempts < cond.MaxRetries
		}
	}

	return attempts < strategy.MaxRetries
}

func (v *VerificationLoop) recordResult(toolName, input, output string, result *VerificationResult) {
	record := VerificationRecord{
		ID:        fmt.Sprintf("%d_%s", time.Now().UnixNano(), toolName),
		ToolName:  toolName,
		Input:     input,
		Output:    output,
		Passed:    result.Passed,
		Timestamp: time.Now(),
	}

	v.mu.Lock()
	v.history = append(v.history, record)
	if len(v.history) > v.maxHistory {
		v.history = v.history[1:]
	}
	v.mu.Unlock()
}

func (v *VerificationLoop) GetHistory(toolName string, limit int) []VerificationResult {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var records []VerificationRecord
	for i := len(v.history) - 1; i >= 0 && len(records) < limit; i-- {
		if toolName == "" || v.history[i].ToolName == toolName {
			records = append(records, v.history[i])
		}
	}

	results := make([]VerificationResult, 0, len(records))
	for _, r := range records {
		results = append(results, VerificationResult{
			Passed: r.Passed,
			RetryAdvice: fmt.Sprintf("retries: %d, latency: %dms", r.Retries, r.LatencyMs),
		})
	}
	return results
}

func (v *VerificationLoop) GetStats() map[string]interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()

	total := len(v.history)
	passed := 0
	byTool := make(map[string]int)

	for _, r := range v.history {
		if r.Passed {
			passed++
		}
		byTool[r.ToolName]++
	}

	passRate := 0.0
	if total > 0 {
		passRate = float64(passed) / float64(total)
	}

	return map[string]interface{}{
		"total_verifications": total,
		"passed":              passed,
		"failed":              total - passed,
		"pass_rate":           passRate,
		"by_tool":             byTool,
		"rules_count":         len(v.rules),
		"strategies_count":    len(v.strategies),
	}
}

func (v *VerificationLoop) AddRule(rule *VerifyRule) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.rules[rule.ID] = rule
	log.Printf("[VerificationLoop] Added rule: %s for tool %s", rule.ID, rule.ToolName)
}

func (v *VerificationLoop) UpdateStrategy(strategy *RetryStrategy) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.strategies[strategy.ToolName] = *strategy
	log.Printf("[VerificationLoop] Updated strategy for tool: %s", strategy.ToolName)
}

func (v *VerificationLoop) AnalyzeFailurePatterns() map[string]interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()

	patterns := make(map[string]int)
	toolFailures := make(map[string]map[string]int)

	for _, r := range v.history {
		if !r.Passed {
			patterns["total_failures"]++
			if toolFailures[r.ToolName] == nil {
				toolFailures[r.ToolName] = make(map[string]int)
			}

			errorType := v.classifyError(r.Error)
			toolFailures[r.ToolName][errorType]++
		}
	}

	return map[string]interface{}{
		"failure_patterns": patterns,
		"by_tool":          toolFailures,
	}
}

func (v *VerificationLoop) classifyError(errorMsg string) string {
	if errorMsg == "" {
		return "unknown"
	}
	lower := strings.ToLower(errorMsg)
	switch {
	case strings.Contains(lower, "timeout"):
		return "timeout"
	case strings.Contains(lower, "syntax"):
		return "syntax_error"
	case strings.Contains(lower, "permission"):
		return "permission_denied"
	case strings.Contains(lower, "connection"):
		return "connection_error"
	case strings.Contains(lower, "memory"):
		return "memory_error"
	default:
		return "other"
	}
}

func (v *VerificationLoop) SuggestOptimization(toolName string) string {
	analysis := v.AnalyzeFailurePatterns()
	byTool, ok := analysis["by_tool"].(map[string]map[string]int)
	if !ok {
		return "暂无足够数据分析"
	}

	toolStats, ok := byTool[toolName]
	if !ok {
		return "该工具暂无失败记录"
	}

	var maxError string
	var maxCount int
	for error, count := range toolStats {
		if count > maxCount {
			maxCount = count
			maxError = error
		}
	}

	switch maxError {
	case "timeout":
		return fmt.Sprintf("超时问题最多(%d次)，建议增加超时时间或优化查询", maxCount)
	case "syntax_error":
		return fmt.Sprintf("语法错误最多(%d次)，建议改进 SQL 生成模板", maxCount)
	case "connection_error":
		return fmt.Sprintf("连接错误最多(%d次)，建议添加重试和降级策略", maxCount)
	default:
		return fmt.Sprintf("主要错误类型: %s (%d次)", maxError, maxCount)
	}
}

var _ = models.SkillRuntimeMemory{}

type VerificationAPI struct {
	loop *VerificationLoop
}

func NewVerificationAPI(loop *VerificationLoop) *VerificationAPI {
	return &VerificationAPI{loop: loop}
}

func (a *VerificationAPI) Verify(toolName, input, output string) *VerificationResult {
	return a.loop.VerifyResult(toolName, input, output)
}

func (a *VerificationAPI) GetStats() map[string]interface{} {
	return a.loop.GetStats()
}

func (a *VerificationAPI) GetHistory(toolName string, limit int) []VerificationResult {
	return a.loop.GetHistory(toolName, limit)
}

func (a *VerificationAPI) GetRetryDelay(toolName, errorMsg string, attempt int) int {
	return a.loop.GetRetryDelay(toolName, errorMsg, attempt)
}