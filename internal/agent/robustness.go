package agent

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/alaikis/opentether/internal/llm"
	"github.com/alaikis/opentether/internal/models"
)

func normalizeQueryForMatching(query string) string {
	query = strings.ToLower(query)
	for _, ch := range []string{"的", "了", "是", "在", "？", "，", ",", " ", "　", "the", "a", "an", "is", "are", "was", "were", "?", "!", ".", "of", "to", "in", "for"} {
		query = strings.ReplaceAll(query, ch, "")
	}
	return query
}

func tokenScore(query string, tokens []string) int {
	score := 0
	for _, token := range tokens {
		token = strings.TrimSpace(strings.ToLower(token))
		if token == "" {
			continue
		}
		if strings.Contains(query, token) {
			score += 5
		}
		runes := []rune(token)
		if len(runes) >= 2 {
			for i := 0; i <= len(runes)-2; i++ {
				if strings.Contains(query, string(runes[i:i+2])) {
					score++
				}
			}
		}
	}
	return score
}

type templateIntentCache struct {
	mu        sync.Mutex
	intents   []string
	expiresAt time.Time
}

var tmplCache = &templateIntentCache{}

func (e *AgentEngine) loadTemplateIntents() {
	tmplCache.mu.Lock()
	defer tmplCache.mu.Unlock()
	if time.Now().Before(tmplCache.expiresAt) && len(tmplCache.intents) > 0 {
		return
	}
	var mems []struct{ Content string }
	e.db.Model(&models.SkillRuntimeMemory{}).Select("content").Where("type = ? AND source = ? AND status = ?", "text2sql_template", "admin", "active").Limit(50).Find(&mems)
	intents := make([]string, 0, len(mems))
	for _, mem := range mems {
		var tpl struct {
			Intent string `json:"intent"`
		}
		json.Unmarshal([]byte(mem.Content), &tpl)
		if tpl.Intent != "" {
			intents = append(intents, tpl.Intent)
		}
	}
	tmplCache.intents = intents
	tmplCache.expiresAt = time.Now().Add(5 * time.Minute)
}

func (e *AgentEngine) routeByTemplateIntent(message string) fastPathRoute {
	e.loadTemplateIntents()
	tmplCache.mu.Lock()
	intents := make([]string, len(tmplCache.intents))
	copy(intents, tmplCache.intents)
	tmplCache.mu.Unlock()
	normalized := normalizeQueryForMatching(message)
	bestScore := 0
	for _, intent := range intents {
		score := tokenScore(normalized, strings.Split(intent, ","))
		if score > bestScore {
			bestScore = score
		}
	}
	if bestScore >= int(e.GetAutoThreshold()) {
		return fastPathRoute{Route: "fast_text2sql", Intent: "text2sql", Confidence: float64(bestScore) / 10.0, Reason: "模板 intent 评分匹配"}
	}
	return fastPathRoute{}
}

func (e *AgentEngine) compressPromptIfNeeded(ctx context.Context, provider *models.Provider, prompt string) string {
	if len(prompt) <= 8000 {
		return prompt
	}
	client, err := llm.NewClient(provider)
	if err != nil {
		return prompt
	}
	resp, err := llm.ChatCompletionWithRetry(client, ctx, llm.ChatRequest{
		Model:     provider.Model,
		Messages:  []llm.Message{{Role: "user", Content: "压缩以下系统提示，保留关键信息，去冗余。直接返回压缩文本。\n" + prompt}},
		MaxTokens: 256, Temperature: 0.1,
	}, 2)
	if err != nil || resp == nil || strings.TrimSpace(resp.Content) == "" {
		return prompt
	}
	compressed := resp.Content
	if len(compressed) < len(prompt)/4 {
		log.Printf("[Prompt] 压缩过度 (%d -> %d)，返回原提示", len(prompt), len(compressed))
		return prompt
	}
	return compressed
}

type semanticCache struct {
	mu    sync.Mutex
	items map[string]semanticCacheEntry
}

type semanticCacheEntry struct {
	Response  *ChatResponse
	Query     string
	ExpiresAt time.Time
}

var semCache = &semanticCache{items: make(map[string]semanticCacheEntry)}

func (e *AgentEngine) semanticCacheGet(query string) (*ChatResponse, bool) {
	normalized := normalizeQueryForMatching(query)
	semCache.mu.Lock()
	defer semCache.mu.Unlock()
	for key, entry := range semCache.items {
		if time.Now().After(entry.ExpiresAt) {
			delete(semCache.items, key)
			continue
		}
		entryNorm := normalizeQueryForMatching(entry.Query)
		if normalized == entryNorm {
			return entry.Response, true
		}
		if tokenScore(normalized, strings.Fields(entryNorm)) >= 10 {
			return entry.Response, true
		}
	}
	return nil, false
}

func (e *AgentEngine) semanticCacheSet(query string, resp *ChatResponse, ttl time.Duration) {
	normalized := normalizeQueryForMatching(query)
	semCache.mu.Lock()
	semCache.items[normalized] = semanticCacheEntry{Response: resp, Query: query, ExpiresAt: time.Now().Add(ttl)}
	semCache.mu.Unlock()
}

var _ = json.Marshal
var _ = models.SkillRuntimeMemory{}
