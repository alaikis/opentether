package agent

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
)

// EncodeVector 将 float64 向量编码为字节（用于 DB 存储）
func EncodeVector(vec []float64) []byte {
	data := make([]byte, len(vec)*8)
	for i, v := range vec {
		bits := math.Float64bits(v)
		binary.LittleEndian.PutUint64(data[i*8:], bits)
	}
	wrapper := map[string]interface{}{
		"dim": len(vec),
		"vec": data,
	}
	encoded, _ := json.Marshal(wrapper)
	return encoded
}

// DecodeVector 从字节解码为 float64 向量
func DecodeVector(data []byte) ([]float64, error) {
	var wrapper struct {
		Dim int    `json:"dim"`
		Vec []byte `json:"vec"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("decode vector: %w", err)
	}
	vec := make([]float64, wrapper.Dim)
	for i := range vec {
		bits := binary.LittleEndian.Uint64(wrapper.Vec[i*8:])
		vec[i] = math.Float64frombits(bits)
	}
	return vec, nil
}
