package vectorstore

import (
	"encoding/json"
	"sort"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type SQLiteStore struct {
	db *gorm.DB
}

func init() {
	RegisterStore("sqlite", NewSQLiteStore)
}

func NewSQLiteStore(config map[string]interface{}) (Store, error) {
	path := "data/vectors.db"
	if v, ok := config["path"].(string); ok && v != "" {
		path = v
	}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&vectorRow{})
	return &SQLiteStore{db: db}, nil
}

type vectorRow struct {
	SkillID   string `gorm:"primaryKey;type:varchar(36)"`
	SkillName string `gorm:"type:varchar(200)"`
	Vector    string `gorm:"type:text"`
	Metadata  string `gorm:"type:text"`
}

func (s *SQLiteStore) Name() string { return "sqlite" }

func (s *SQLiteStore) Index(skillID, skillName string, vector []float64) error {
	b, _ := json.Marshal(vector)
	return s.db.Save(&vectorRow{SkillID: skillID, SkillName: skillName, Vector: string(b)}).Error
}

func (s *SQLiteStore) Update(skillID, skillName string, vector []float64) error {
	var existing vectorRow
	err := s.db.First(&existing, "skill_id = ?", skillID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return s.Index(skillID, skillName, vector)
		}
		return err
	}
	b, _ := json.Marshal(vector)
	return s.db.Model(&existing).Updates(map[string]interface{}{
		"skill_name": skillName,
		"vector":     string(b),
	}).Error
}

func (s *SQLiteStore) Search(queryVector []float64, topK int, threshold float64) ([]Match, error) {
	var rows []vectorRow
	s.db.Find(&rows)
	type candidate struct {
		match Match
		score float64
	}
	var candidates []candidate
	for _, row := range rows {
		var vec []float64
		json.Unmarshal([]byte(row.Vector), &vec)
		score := CosineSimilarity(queryVector, vec)
		if score >= threshold {
			candidates = append(candidates, candidate{Match{SkillID: row.SkillID, SkillName: row.SkillName, Score: score, Vector: vec}, score})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	if topK > len(candidates) {
		topK = len(candidates)
	}
	result := make([]Match, 0, topK)
	for i := 0; i < topK; i++ {
		result = append(result, candidates[i].match)
	}
	return result, nil
}

func (s *SQLiteStore) HybridSearch(queryVector []float64, topK int, threshold float64, alpha float64) ([]Match, error) {
	var rows []vectorRow
	s.db.Find(&rows)
	type candidate struct {
		match Match
		score float64
	}
	var candidates []candidate
	for _, row := range rows {
		var vec []float64
		json.Unmarshal([]byte(row.Vector), &vec)
		semScore := CosineSimilarity(queryVector, vec)
		bm25Score := BM25Score(queryVector, vec)
		score := alpha*semScore + (1-alpha)*bm25Score
		if score >= threshold {
			candidates = append(candidates, candidate{Match{SkillID: row.SkillID, SkillName: row.SkillName, Score: score, Vector: vec}, score})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	if topK > len(candidates) {
		topK = len(candidates)
	}
	result := make([]Match, 0, topK)
	for i := 0; i < topK; i++ {
		result = append(result, candidates[i].match)
	}
	return result, nil
}

func (s *SQLiteStore) SearchWithMetadata(queryVector []float64, topK int, threshold float64, filters map[string]string) ([]Match, error) {
	var rows []vectorRow
	s.db.Find(&rows)
	type candidate struct {
		match Match
		score float64
	}
	var candidates []candidate
	for _, row := range rows {
		matched := true
		if len(filters) > 0 {
			meta := parseMetadata(row.Metadata)
			for k, v := range filters {
				if meta[k] != v {
					matched = false
					break
				}
			}
		}
		if !matched {
			continue
		}
		var vec []float64
		json.Unmarshal([]byte(row.Vector), &vec)
		score := CosineSimilarity(queryVector, vec)
		if score >= threshold {
			candidates = append(candidates, candidate{Match{SkillID: row.SkillID, SkillName: row.SkillName, Score: score, Vector: vec}, score})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	if topK > 0 && len(candidates) > topK {
		candidates = candidates[:topK]
	}
	result := make([]Match, len(candidates))
	for i, c := range candidates {
		result[i] = c.match
	}
	return result, nil
}

func (s *SQLiteStore) Remove(skillID string) error {
	return s.db.Delete(&vectorRow{}, "skill_id = ?", skillID).Error
}

func (s *SQLiteStore) Count() int {
	var count int64
	s.db.Model(&vectorRow{}).Count(&count)
	return int(count)
}

func (s *SQLiteStore) Clear() {
	s.db.Where("1 = 1").Delete(&vectorRow{})
}
