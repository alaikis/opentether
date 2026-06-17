package service

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

type RAGService struct {
	db *gorm.DB
}

func NewRAGService(db *gorm.DB) *RAGService { return &RAGService{db: db} }

func (s *RAGService) Ingest(title, source, content string, tags []string, chunkSize int) (*models.RAGDocument, error) {
	if chunkSize <= 0 {
		chunkSize = 800
	}
	chunks := splitContent(content, chunkSize)
	doc := &models.RAGDocument{Title: title, Source: source, Content: content, Chunks: len(chunks), Tags: strings.Join(tags, ","), Enabled: true}
	if err := s.db.Create(doc).Error; err != nil {
		return nil, err
	}
	for i, chunk := range chunks {
		c := &models.RAGChunk{DocumentID: doc.ID, Content: chunk, ChunkIndex: i}
		s.db.Create(c)
	}
	return doc, nil
}

func (s *RAGService) Retrieve(query string, limit int) ([]models.RAGChunk, error) {
	if limit <= 0 {
		limit = 5
	}
	terms := strings.Fields(query)
	var chunks []models.RAGChunk
	s.db.Where("enabled = ?", true).Joins("JOIN rag_documents ON rag_chunks.document_id = rag_documents.id").Where("rag_documents.enabled = ?", true).Find(&chunks)
	scored := make([]struct {
		chunk models.RAGChunk
		score int
	}, 0, len(chunks))
	for _, c := range chunks {
		score := 0
		lower := strings.ToLower(c.Content)
		for _, term := range terms {
			if len(term) >= 2 && strings.Contains(lower, strings.ToLower(term)) {
				score += 3
			}
		}
		if score > 0 {
			scored = append(scored, struct {
				chunk models.RAGChunk
				score int
			}{chunk: c, score: score})
		}
	}
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
	result := make([]models.RAGChunk, 0, limit)
	for i := 0; i < limit && i < len(scored); i++ {
		result = append(result, scored[i].chunk)
	}
	return result, nil
}

func (s *RAGService) BuildContext(query string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 3000
	}
	chunks, err := s.Retrieve(query, 8)
	if err != nil || len(chunks) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, chunk := range chunks {
		text := strings.TrimSpace(chunk.Content)
		if utf8.RuneCountInString(text) > 600 {
			text = string([]rune(text)[:600]) + "..."
		}
		sb.WriteString(text)
		sb.WriteString("\n\n")
	}
	ctx := sb.String()
	if len(ctx) > maxChars {
		ctx = ctx[:maxChars]
	}
	return stripEnvironmentDetails(ctx)
}

func (s *RAGService) ListDocuments() ([]models.RAGDocument, error) {
	var docs []models.RAGDocument
	return docs, s.db.Order("created_at DESC").Find(&docs).Error
}

func (s *RAGService) DeleteDocument(id string) error {
	s.db.Where("document_id = ?", id).Delete(&models.RAGChunk{})
	return s.db.Delete(&models.RAGDocument{}, "id = ?", id).Error
}

func splitContent(content string, size int) []string {
	if size <= 0 {
		size = 800
	}
	runes := []rune(content)
	var chunks []string
	for i := 0; i < len(runes); i += size {
		end := i + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}

func (s *RAGService) SaveDocument(doc *models.RAGDocument) error {
	return s.db.Save(doc).Error
}

func (s *RAGService) IngestJSON(title, source, content string, tags []string) (*models.RAGDocument, error) {
	b, _ := json.Marshal(map[string]string{"title": title, "source": source, "content": content, "tags": strings.Join(tags, ",")})
	return s.Ingest(title, source, string(b), tags, 800)
}
