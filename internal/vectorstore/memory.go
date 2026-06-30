package vectorstore

import (
	"sort"
)

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
	metadata  map[string]string
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
		metadata:  map[string]string{},
	}
	return nil
}

func (m *MemoryStore) Update(skillID, skillName string, vector []float64) error {
	if existing, ok := m.vectors[skillID]; ok {
		m.vectors[skillID] = vectorEntry{
			skillID:   skillID,
			skillName: skillName,
			vector:    vector,
			metadata:  existing.metadata,
		}
		return nil
	}
	return m.Index(skillID, skillName, vector)
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

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if topK > 0 && len(candidates) > topK {
		candidates = candidates[:topK]
	}

	result := make([]Match, len(candidates))
	for i, c := range candidates {
		result[i] = c.match
	}

	return result, nil
}

func (m *MemoryStore) HybridSearch(queryVector []float64, topK int, threshold float64, alpha float64) ([]Match, error) {
	type candidate struct {
		match Match
		score float64
	}

	var candidates []candidate
	for _, entry := range m.vectors {
		semScore := CosineSimilarity(queryVector, entry.vector)
		bm25Score := BM25Score(queryVector, entry.vector)
		score := alpha*semScore + (1-alpha)*bm25Score
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

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if topK > 0 && len(candidates) > topK {
		candidates = candidates[:topK]
	}

	result := make([]Match, len(candidates))
	for i, c := range candidates {
		result[i] = c.match
	}

	return result, nil
}

func (m *MemoryStore) SearchWithMetadata(queryVector []float64, topK int, threshold float64, filters map[string]string) ([]Match, error) {
	if len(filters) == 0 {
		return m.Search(queryVector, topK, threshold)
	}

	type candidate struct {
		match Match
		score float64
	}

	var candidates []candidate
	for _, entry := range m.vectors {
		matched := true
		for k, v := range filters {
			if entry.metadata[k] != v {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
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

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

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
