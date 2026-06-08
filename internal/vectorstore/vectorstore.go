package vectorstore

import "math"

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

	// Search 按余弦相似度搜索 TopK 最相似的 Skill
	Search(queryVector []float64, topK int, threshold float64) ([]Match, error)

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
