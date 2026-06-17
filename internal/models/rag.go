package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RAGDocument struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Title     string    `json:"title" gorm:"type:varchar(500)"`
	Source    string    `json:"source" gorm:"type:varchar(500);index"`
	Content   string    `json:"content" gorm:"type:longtext"`
	Chunks    int       `json:"chunks" gorm:"default:1"`
	Embedding string    `json:"embedding" gorm:"type:longtext"`
	Tags      string    `json:"tags" gorm:"type:varchar(1000)"`
	Enabled   bool      `json:"enabled" gorm:"default:true;index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (m *RAGDocument) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

type RAGChunk struct {
	ID         string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	DocumentID string    `json:"document_id" gorm:"type:varchar(36);index;not null"`
	Content    string    `json:"content" gorm:"type:text"`
	ChunkIndex int       `json:"chunk_index"`
	Embedding  string    `json:"embedding" gorm:"type:longtext"`
	Metadata   string    `json:"metadata" gorm:"type:varchar(1000)"`
	CreatedAt  time.Time `json:"created_at"`
}

func (m *RAGChunk) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}
