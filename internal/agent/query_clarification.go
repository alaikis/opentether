package agent

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/alaikis/opentether/internal/models"
)

type querySlots struct {
	Metric  string
	Subject string
	Time    string
}

type QueryClarificationResult struct {
	Message       string
	Response      *ChatResponse
	Rewritten     bool
	NeedsClarify  bool
	ResolvedSlots querySlots
}

func ResolveQueryClarification(message string, recent []models.Message, conversationID string) QueryClarificationResult {
	if isMultiPartQuestion(message) {
		return QueryClarificationResult{Message: message}
	}
	if !isShortOrVagueQuery(message) {
		return QueryClarificationResult{Message: message}
	}
	hasRecentSalesContext := false
	for i := len(recent) - 1; i >= 0; i-- {
		if recent[i].Role == "user" && utf8.RuneCountInString(recent[i].Content) > 6 {
			hasRecentSalesContext = strings.Contains(recent[i].Content, "多少") ||
				strings.Contains(recent[i].Content, "查询") ||
				strings.Contains(recent[i].Content, "销售") ||
				strings.Contains(recent[i].Content, "订单")
			break
		}
	}
	if hasRecentSalesContext {
		return QueryClarificationResult{Message: message}
	}
	return QueryClarificationResult{
		Message:      message,
		NeedsClarify: true,
		Response: &ChatResponse{
			Message:        "请补充您的查询需求，例如：查询对象、时间范围（本月/今年）、指标（订单数/销售额/利润）等。",
			ConversationID: conversationID,
			SkillUsed:      "query_clarification",
			Data: map[string]interface{}{
				"needs_clarification": true,
			},
		},
	}
}

func EmptyQueryClarification(query string) string {
	if isShortOrVagueQuery(query) {
		return "请补充您的查询需求，例如：查询对象、时间范围（本月/今年）、指标（订单数/销售额/利润）等。"
	}
	return ""
}

func isMultiPartQuestion(message string) bool {
	return strings.Count(message, "？")+strings.Count(message, "?") >= 2
}

func isShortOrVagueQuery(message string) bool {
	m := strings.TrimSpace(message)
	runes := utf8.RuneCountInString(m)
	if runes > 15 {
		return false
	}
	if runes <= 5 {
		return looksLikeBusinessQuery(m)
	}
	if looksLikeBusinessQuery(m) {
		return false
	}
	return false
}

func looksLikeBusinessQuery(message string) bool {
	return strings.Contains(message, "卖") ||
		strings.Contains(message, "多少") ||
		strings.Contains(message, "查询") ||
		strings.Contains(message, "订单") ||
		strings.Contains(message, "销售") ||
		strings.Contains(message, "利润") ||
		strings.Contains(message, "库存") ||
		strings.Contains(message, "采购") ||
		strings.Contains(message, "客户") ||
		strings.Contains(message, "产品") ||
		strings.Contains(message, "业绩") ||
		strings.Contains(message, "成本")
}

func SplitMultiPartQuestions(message string) []string {
	message = strings.TrimSpace(message)
	parts := strings.Split(message, "？")
	stripped := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			stripped = append(stripped, p)
		}
	}
	if len(stripped) <= 1 {
		parts = strings.Split(message, "?")
		stripped = stripped[:0]
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				stripped = append(stripped, p)
			}
		}
	}
	return stripped
}

func rewriteSalesQuery(subject, time, metric string) string {
	parts := make([]string, 0, 3)
	if subject != "" && subject != "我" {
		parts = append(parts, subject)
	}
	if time != "" {
		parts = append(parts, time)
	}
	if metric != "" {
		parts = append(parts, metric)
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("查询%s", strings.Join(parts, ""))
}
