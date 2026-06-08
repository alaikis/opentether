package embedding

// Embedder 向量化接口——将文本转为语义向量
type Embedder interface {
	// Embed 将文本转为向量，维度由 Dims() 返回
	Embed(text string) ([]float64, error)

	// Dims 返回向量维度
	Dims() int

	// Name 返回实现名称（如 "tfidf", "bge-m3", "openai"）
	Name() string
}

// EmbedderFactory 创建 Embedder 的工厂函数
type EmbedderFactory func(config map[string]interface{}) (Embedder, error)

// 注册的工厂
var registry = map[string]EmbedderFactory{}

// Register 注册一个 Embedder 工厂
func Register(name string, factory EmbedderFactory) {
	registry[name] = factory
}

// Create 根据配置创建 Embedder，无配置时返回默认 TF-IDF
func Create(provider string, config map[string]interface{}) (Embedder, error) {
	if provider == "" {
		provider = "tfidf"
	}

	factory, exists := registry[provider]
	if !exists {
		// 回退到默认
		factory = registry["tfidf"]
	}

	return factory(config)
}

// DefaultConfig 返回默认配置（TF-IDF，零依赖）
func DefaultConfig() (string, map[string]interface{}) {
	return "tfidf", map[string]interface{}{}
}
