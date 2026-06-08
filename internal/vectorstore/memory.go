package vectorstore

func init() {
	RegisterStore("memory", NewMemoryStore)
}

// MemoryStore 默认内存向量存储——暴力搜索，零依赖
type MemoryStore struct {
	vectors map[string]vectorEntry
}

type vectorEntry struct {
	skillID   string
	skillName string
	vector    []float64
}

func NewMemoryStore(config map[string]interface{}) (Store, error) {
	return &MemoryStore{
		vectors: map[string]vectorEntry{},
	}, nil
}

func (m *MemoryStore) Name() string { return "memory" }

func (m *MemoryStore) Index(skillID, skillName string, vector []float64) error {
	m.vectors[skillID] = vectorEntry{
		skillID:   skillID,
		skillName: skillName,
		vector:    vector,
	}
	return nil
}

func (m *MemoryStore) Search(queryVector []float64, topK int, threshold float64) ([]Match, error) {
	type candidate struct {
		match Match
		score float64
	}

	var candidates []candidate
	for _, entry := range m.vectors {
		score := CosineSimilarity(queryVector, entry.vector)
		if score >= threshold {
			candidates = append(candidates, candidate{
				match: Match{
					SkillID:   entry.skillID,
					SkillName: entry.skillName,
					Score:     score,
					Vector:    entry.vector,
				},
				score: score,
			})
		}
	}

	// 简单降序排序 TopK
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].score > candidates[i].score {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	if topK > 0 && len(candidates) > topK {
		candidates = candidates[:topK]
	}

	result := make([]Match, len(candidates))
	for i, c := range candidates {
		result[i] = c.match
	}

	return result, nil
}

func (m *MemoryStore) Remove(skillID string) error {
	delete(m.vectors, skillID)
	return nil
}

func (m *MemoryStore) Count() int {
	return len(m.vectors)
}

func (m *MemoryStore) Clear() {
	m.vectors = map[string]vectorEntry{}
}
