package embedding

func init() {
	Register("onnx", NewONNX)
}

// ONNXEmbedder ONNX embedder stub，使用固定 384 维向量
type ONNXEmbedder struct {
	dim int
}

func NewONNX(config map[string]interface{}) (Embedder, error) {
	return &ONNXEmbedder{dim: 384}, nil
}

func (e *ONNXEmbedder) Name() string { return "onnx" }
func (e *ONNXEmbedder) Dims() int    { return e.dim }

func (e *ONNXEmbedder) Embed(text string) ([]float64, error) {
	vec := make([]float64, e.dim)
	for i := range vec {
		vec[i] = float64(int(text[i%len(text)]) % 100) / 100.0
	}
	return vec, nil
}
