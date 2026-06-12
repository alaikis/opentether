package agent

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alaikis/opentether/internal/embedding"
	"github.com/alaikis/opentether/internal/models"
	"github.com/alaikis/opentether/internal/vectorstore"
	"gorm.io/gorm"
)

const fastClassifierThreshold = 0.35

type RoutePrediction struct {
	Route       string
	Intent      string
	Confidence  float64
	MatchedText string
	Source      string
}

type routeExampleSeed struct {
	Text   string
	Route  string
	Intent string
}

var builtinRouteExamples = []routeExampleSeed{
	{"你好", "fast_local", "greeting"},
	{"你是谁", "fast_local", "identity"},
	{"帮助", "fast_local", "help"},
	{"今天星期几", "fast_local", "date"},
	{"林烽上季度销售额", "fast_text2sql", "employee_sales_amount"},
	{"张三上个月订单数", "fast_text2sql", "employee_order_count"},
	{"李四本月卖了多少单", "fast_text2sql", "employee_order_count"},
	{"王五今年销售额多少", "fast_text2sql", "employee_sales_amount"},
	{"什么是销售转化率", "fast_chat", "explain_metric"},
	{"解释一下库存周转率", "fast_chat", "explain_metric"},
	{"怎么理解毛利率", "fast_chat", "explain_metric"},
	{"帮我生成销售报表", "agent_loop", "report"},
	{"导出 PDF 报表", "agent_loop", "report"},
	{"读取这个文件并总结", "agent_loop", "file_process"},
	{"调用 MCP 工具查询文件", "agent_loop", "mcp"},
	{"执行脚本处理数据", "agent_loop", "script"},
}

type FastPathClassifier struct {
	db       *gorm.DB
	mu       sync.RWMutex
	loaded   bool
	loadedAt time.Time
	examples []models.RouteExample
	embedder embedding.Embedder
	store    vectorstore.Store
}

func NewFastPathClassifier(db *gorm.DB) *FastPathClassifier {
	return &FastPathClassifier{db: db}
}

func (c *FastPathClassifier) Predict(text string) RoutePrediction {
	if c == nil || strings.TrimSpace(text) == "" {
		return RoutePrediction{}
	}
	c.ensureLoaded()
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.loaded || c.embedder == nil || c.store == nil || len(c.examples) == 0 {
		return RoutePrediction{}
	}
	vec, err := c.embedder.Embed(text)
	if err != nil {
		return RoutePrediction{}
	}
	matches, err := c.store.Search(vec, 3, fastClassifierThreshold)
	if err != nil || len(matches) == 0 {
		return RoutePrediction{}
	}
	byID := map[string]models.RouteExample{}
	for _, ex := range c.examples {
		byID[ex.ID] = ex
	}
	type routeScore struct {
		route  string
		intent string
		score  float64
		text   string
		source string
	}
	scores := map[string]routeScore{}
	for _, m := range matches {
		ex, ok := byID[m.SkillID]
		if !ok {
			continue
		}
		weighted := m.Score * (0.75 + ex.Confidence*0.25)
		cur := scores[ex.Route]
		prevScore := cur.score
		cur.route = ex.Route
		cur.score += weighted
		if weighted > prevScore || cur.text == "" {
			cur.intent = ex.Intent
			cur.text = ex.Text
			cur.source = ex.Source
		}
		scores[ex.Route] = cur
	}
	var list []routeScore
	for _, s := range scores {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].score > list[j].score })
	if len(list) == 0 || list[0].score < fastClassifierThreshold {
		return RoutePrediction{}
	}
	return RoutePrediction{Route: list[0].route, Intent: list[0].intent, Confidence: list[0].score, MatchedText: list[0].text, Source: list[0].source}
}

func (c *FastPathClassifier) Reload() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.loaded = false
	c.mu.Unlock()
	c.ensureLoaded()
}

func (c *FastPathClassifier) ensureLoaded() {
	c.mu.RLock()
	if c.loaded && time.Since(c.loadedAt) < time.Minute {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.loaded && time.Since(c.loadedAt) < time.Minute {
		return
	}
	examples := c.loadExamples()
	if len(examples) == 0 {
		c.loaded = true
		c.loadedAt = time.Now()
		return
	}
	docs := make([]string, 0, len(examples))
	for _, ex := range examples {
		docs = append(docs, ex.Text)
	}
	embedder, err := embedding.Create("tfidf", map[string]interface{}{"corpus": docs})
	if err != nil {
		c.loaded = true
		c.loadedAt = time.Now()
		return
	}
	store, err := vectorstore.CreateStore("memory", nil)
	if err != nil {
		c.loaded = true
		c.loadedAt = time.Now()
		return
	}
	for _, ex := range examples {
		vec, err := embedder.Embed(ex.Text)
		if err != nil {
			continue
		}
		_ = store.Index(ex.ID, ex.Text, vec)
	}
	c.examples = examples
	c.embedder = embedder
	c.store = store
	c.loaded = true
	c.loadedAt = time.Now()
}

func (c *FastPathClassifier) loadExamples() []models.RouteExample {
	var examples []models.RouteExample
	for i, seed := range builtinRouteExamples {
		examples = append(examples, models.RouteExample{
			ID:         "builtin_route_" + string(rune('a'+i)),
			Text:       seed.Text,
			Route:      seed.Route,
			Intent:     seed.Intent,
			Source:     "builtin",
			Status:     "active",
			Confidence: 0.9,
			UseCount:   1,
		})
	}
	if c.db == nil {
		return examples
	}
	var custom []models.RouteExample
	if err := c.db.Where("status = ? AND source <> ?", "active", "rejected").Find(&custom).Error; err == nil {
		examples = append(examples, custom...)
	}
	return examples
}

func (e *AgentEngine) routeByEmbeddedClassifier(message string) RoutePrediction {
	if e == nil || e.fastClassifier == nil {
		return RoutePrediction{}
	}
	return e.fastClassifier.Predict(message)
}

func (e *AgentEngine) learnRouteExampleCandidate(text, route, intent string, confidence float64) {
	if e == nil || e.db == nil || text == "" || route == "" {
		return
	}
	var existing models.RouteExample
	if err := e.db.Where("text = ? AND route = ?", text, route).First(&existing).Error; err == nil {
		existing.UseCount++
		if existing.Confidence < confidence {
			existing.Confidence = confidence
		}
		_ = e.db.Save(&existing).Error
		return
	}
	ex := models.RouteExample{Text: text, Route: route, Intent: intent, Source: "runtime", Status: "pending", Confidence: confidence, UseCount: 1}
	_ = e.db.Create(&ex).Error
}
