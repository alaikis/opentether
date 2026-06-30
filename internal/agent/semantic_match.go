package agent

import (
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alaikis/opentether/internal/embedding"
	"github.com/alaikis/opentether/internal/models"
)

var (
	semanticCacheMu     sync.Mutex
	semanticToolCache   map[string][]ToolDef
	semanticToolTexts   []string
	semanticToolIndex   int
	semanticCacheQuery  string
	semanticCacheResult []ToolDef
	semanticCacheExpiry time.Time
)

func init() {
	semanticToolCache = make(map[string][]ToolDef)
}

func SemanticMatchTools(query string, tools []ToolDef, limit int) []ToolDef {
	if len(tools) == 0 {
		return nil
	}
	if limit <= 0 || len(tools) <= limit {
		return tools
	}

	emb, err := embedding.Create("", nil)
	if err != nil {
		return tools
	}

	semanticCacheMu.Lock()
	if semanticCacheQuery == query && time.Now().Before(semanticCacheExpiry) && len(semanticCacheResult) > 0 {
		result := semanticCacheResult
		semanticCacheMu.Unlock()
		return result
	}
	semanticCacheMu.Unlock()

	queryVec, err := emb.Embed(strings.ToLower(query))
	if err != nil {
		return tools
	}

	type scoredTool struct {
		tool  ToolDef
		score float64
		idx   int
	}
	scored := make([]scoredTool, 0, len(tools))
	for i, tool := range tools {
		toolText := strings.ToLower(tool.Name + " " + tool.Description)
		toolVec, err := emb.Embed(toolText)
		if err != nil {
			continue
		}
		score := cosineSimilarity(queryVec, toolVec)
		scored = append(scored, scoredTool{tool: tool, score: score, idx: i})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if math.Abs(scored[i].score-scored[j].score) < 0.001 {
			return scored[i].idx < scored[j].idx
		}
		return scored[i].score > scored[j].score
	})

	selected := make([]ToolDef, 0, limit)
	for i := 0; i < limit && i < len(scored); i++ {
		selected = append(selected, scored[i].tool)
	}

	semanticCacheMu.Lock()
	semanticCacheQuery = query
	semanticCacheResult = selected
	semanticCacheExpiry = time.Now().Add(5 * time.Minute)
	semanticCacheMu.Unlock()

	return selected
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func SemanticMatchToolsForEngine(e *AgentEngine, query string, tools []ToolDef, limit int) []ToolDef {
	if e == nil || e.db == nil {
		return SemanticMatchTools(query, tools, limit)
	}
	var skillTools []models.SkillRuntimeMemory
	e.db.Where("type = ? AND source = ?", "tool_selection_feedback", "runtime").Order("created_at DESC").Limit(200).Find(&skillTools)
	if len(skillTools) == 0 {
		return SemanticMatchTools(query, tools, limit)
	}
	scoredTools := SemanticMatchTools(query, tools, len(tools))
	if len(scoredTools) == 0 {
		return tools
	}
	feedbackScores := map[string]float64{}
	for _, fb := range skillTools {
		parts := strings.Split(fb.Content, ",")
		selected := ""
		for _, part := range parts {
			if strings.HasPrefix(part, "selected=") {
				selected = strings.TrimPrefix(part, "selected=")
				break
			}
		}
		if selected != "" {
			names := strings.Split(selected, ",")
			for _, name := range names {
				feedbackScores[name] += 0.1
			}
		}
	}
	type scoredTool struct {
		tool  ToolDef
		score float64
		idx   int
	}
	scored := make([]scoredTool, 0, len(scoredTools))
	for i, tool := range scoredTools {
		baseScore := 0.5
		if s, ok := feedbackScores[tool.Name]; ok {
			baseScore += s
		}
		scored = append(scored, scoredTool{tool: tool, score: baseScore, idx: i})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if math.Abs(scored[i].score-scored[j].score) < 0.001 {
			return scored[i].idx < scored[j].idx
		}
		return scored[i].score > scored[j].score
	})
	selected := make([]ToolDef, 0, limit)
	for i := 0; i < limit && i < len(scored); i++ {
		selected = append(selected, scored[i].tool)
	}
	return selected
}
