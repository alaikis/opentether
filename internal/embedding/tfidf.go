package embedding

import (
	"math"
	"strings"
)

// TFIDFEmbedder 默认的纯 Go TF-IDF 向量化器，零外部依赖
type TFIDFEmbedder struct {
	vocabulary map[string]int
	idf        []float64
	dim        int
}

func init() {
	Register("tfidf", NewTFIDF)
}

// NewTFIDF 创建 TF-IDF Embedder
// config 可选参数: {"corpus": ["doc1", "doc2", ...]}
func NewTFIDF(config map[string]interface{}) (Embedder, error) {
	e := &TFIDFEmbedder{
		vocabulary: map[string]int{},
	}

	// 如果提供了语料库，构建词表
	if corpus, ok := config["corpus"].([]string); ok && len(corpus) > 0 {
		e.buildVocabulary(corpus)
	}

	return e, nil
}

func (e *TFIDFEmbedder) Name() string { return "tfidf" }
func (e *TFIDFEmbedder) Dims() int    { return e.dim }

// BuildVocabulary 用文档集合构建词表（用于初始化时传入 Skill 集合）
func (e *TFIDFEmbedder) BuildVocabulary(documents []string) {
	e.buildVocabulary(documents)
}

func (e *TFIDFEmbedder) buildVocabulary(documents []string) {
	e.vocabulary = map[string]int{}

	allTokens := make([][]string, len(documents))
	for i, doc := range documents {
		tokens := tokenize(doc)
		allTokens[i] = tokens
		for _, t := range tokens {
			if _, exists := e.vocabulary[t]; !exists {
				e.vocabulary[t] = len(e.vocabulary)
			}
		}
	}

	e.dim = len(e.vocabulary)
	e.idf = make([]float64, e.dim)

	docCount := float64(len(documents))
	for term, idx := range e.vocabulary {
		docFreq := 0
		for _, tokens := range allTokens {
			for _, t := range tokens {
				if t == term {
					docFreq++
					break
				}
			}
		}
		e.idf[idx] = math.Log((docCount+1)/(float64(docFreq)+1)) + 1
	}
}

func (e *TFIDFEmbedder) Embed(text string) ([]float64, error) {
	vec := make([]float64, e.dim)
	tokens := tokenize(text)
	tf := map[string]int{}
	for _, t := range tokens {
		tf[t]++
	}
	for term, count := range tf {
		if idx, exists := e.vocabulary[term]; exists {
			vec[idx] = float64(count) * e.idf[idx]
		}
	}
	return vec, nil
}

// tokenize 中文+英文分词
func tokenize(text string) []string {
	text = strings.ToLower(text)
	tokens := []string{}

	words := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == ',' || r == '，' || r == '。' ||
			r == '；' || r == '、' || r == '\n' || r == '\t' ||
			r == '(' || r == ')' || r == '（' || r == '）'
	})

	for _, w := range words {
		w = strings.TrimSpace(w)
		if len(w) == 0 {
			continue
		}
		if isCJK(w) {
			runes := []rune(w)
			for i := 0; i < len(runes); i++ {
				tokens = append(tokens, string(runes[i]))
				if i+1 < len(runes) {
					tokens = append(tokens, string(runes[i:i+2]))
				}
			}
		} else {
			parts := strings.Fields(w)
			for _, p := range parts {
				if len(p) > 1 {
					tokens = append(tokens, p)
				}
			}
		}
	}

	return tokens
}

func isCJK(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}
