package agent

import (
	"log"

	"github.com/alaikis/opentether/internal/embedding"
	"github.com/alaikis/opentether/internal/vectorstore"
	"github.com/alaikis/opentether/internal/config"
)

// VectorConfig 从应用配置中解析向量相关配置
type VectorConfig struct {
	EmbeddingProvider  string
	EmbeddingConfig    map[string]interface{}
	VectorStoreProvider string
	VectorStoreConfig  map[string]interface{}
}

// LoadVectorConfig 从 config.yaml 加载向量配置，未配置时使用默认值
func LoadVectorConfig(cfg *config.Config) VectorConfig {
	vc := VectorConfig{
		EmbeddingProvider:  "tfidf",
		VectorStoreProvider: "memory",
		EmbeddingConfig:    map[string]interface{}{},
		VectorStoreConfig:  map[string]interface{}{},
	}

	if cfg != nil && cfg.Embedding.Model != "" {
		vc.EmbeddingProvider = "tfidf" // 默认仍用 tfidf，可通过配置切换
		vc.EmbeddingConfig["model"] = cfg.Embedding.Model
		vc.EmbeddingConfig["dimension"] = cfg.Embedding.Dimension
	}

	return vc
}

// NewEmbedder 根据配置创建 Embedder
func NewEmbedder(cfg VectorConfig, corpus []string) (embedding.Embedder, error) {
	if cfg.EmbeddingProvider == "" {
		cfg.EmbeddingProvider = "tfidf"
	}

	cfg.EmbeddingConfig["corpus"] = corpus
	emb, err := embedding.Create(cfg.EmbeddingProvider, cfg.EmbeddingConfig)
	if err != nil {
		return nil, err
	}
	log.Printf("[Vector] Embedder: %s (dim=%d)", emb.Name(), emb.Dims())
	return emb, nil
}

// NewVectorStore 根据配置创建 VectorStore
func NewVectorStore(cfg VectorConfig) (vectorstore.Store, error) {
	if cfg.VectorStoreProvider == "" {
		cfg.VectorStoreProvider = "memory"
	}

	store, err := vectorstore.CreateStore(cfg.VectorStoreProvider, cfg.VectorStoreConfig)
	if err != nil {
		return nil, err
	}
	log.Printf("[Vector] Store: %s (count=%d)", store.Name(), store.Count())
	return store, nil
}
