package vectorstore

import (
	"math"
	"strings"
)

// Match 向量匹配结果
type Match struct {
	SkillID  string
	SkillName string
	Score    float64 // 余弦相似度 [0, 1]
	Vector   []float64
}

// Store 向量存储接口——可插拔实现
type Store interface {
	// Index 为一个 Skill 建立向量索引
	Index(skillID, skillName string, vector []float64) error

	// Update 更新已有索引
	Update(skillID, skillName string, vector []float64) error

	// Search 按余弦相似度搜索 TopK 最相似的 Skill
	Search(queryVector []float64, topK int, threshold float64) ([]Match, error)

	// HybridSearch 混合搜索（语义 + BM25）
	HybridSearch(queryVector []float64, topK int, threshold float64, alpha float64) ([]Match, error)

	// SearchWithMetadata 带元数据过滤的搜索
	SearchWithMetadata(queryVector []float64, topK int, threshold float64, filters map[string]string) ([]Match, error)

	// Remove 删除一个 Skill 的索引
	Remove(skillID string) error

	// Count 返回索引数量
	Count() int

	// Clear 清空所有索引
	Clear()

	// Name 返回实现名称
	Name() string
}

// StoreFactory 创建 Store 的工厂函数
type StoreFactory func(config map[string]interface{}) (Store, error)

var storeRegistry = map[string]StoreFactory{}

// RegisterStore 注册一个 Store 工厂
func RegisterStore(name string, factory StoreFactory) {
	storeRegistry[name] = factory
}

// CreateStore 根据配置创建 Store，无配置时返回默认内存实现
func CreateStore(provider string, config map[string]interface{}) (Store, error) {
	if provider == "" {
		provider = "memory"
	}

	factory, exists := storeRegistry[provider]
	if !exists {
		factory = storeRegistry["memory"]
	}

	return factory(config)
}

// CosineSimilarity 计算余弦相似度（公开工具函数）
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
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

// BM25Score 简化版 BM25 评分（词频 + 长度归一化）
func BM25Score(queryVector, docVector []float64) float64 {
	if len(queryVector) == 0 || len(docVector) == 0 || len(queryVector) != len(docVector) {
		return 0
	}
	avgLen := 1.0
	k1 := 1.2
	b := 0.75
	docLen := 0.0
	for _, v := range docVector {
		docLen += v
	}
	if docLen == 0 {
		return 0
	}
	var score float64
	for i, qv := range queryVector {
		if qv == 0 {
			continue
		}
		tf := docVector[i]
		if tf == 0 {
			continue
		}
		score += qv * ((tf * (k1 + 1)) / (tf + k1*(1-b+b*(docLen/avgLen))))
	}
	return score
}

func parseMetadata(s string) map[string]string {
	m := map[string]string{}
	if s == "" {
		return m
	}
	parts := strings.Split(s, ";")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			m[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return m
}

func serializeMetadata(m map[string]string) string {
	var parts []string
	for k, v := range m {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ";")
}
