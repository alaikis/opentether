package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SystemSetting struct {
	Key       string    `json:"key" gorm:"type:varchar(120);primaryKey"`
	Value     string    `json:"value" gorm:"type:text"`
	Sensitive bool      `json:"sensitive" gorm:"default:false"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SkillPublishRequest struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	SkillID   string    `json:"skill_id" gorm:"type:varchar(36);index;not null"`
	Status    string    `json:"status" gorm:"type:varchar(30);default:pending;index"`
	Reason    string    `json:"reason" gorm:"type:text"`
	Reviewer  string    `json:"reviewer" gorm:"type:varchar(36)"`
	Comment   string    `json:"comment" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (m *SkillPublishRequest) BeforeCreate(tx *gorm.DB) error { if m.ID == "" { m.ID = uuid.New().String() }; return nil }

type BackupRecord struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Path      string    `json:"path" gorm:"type:varchar(1000)"`
	Status    string    `json:"status" gorm:"type:varchar(30);index"`
	Error     string    `json:"error" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
}

func (m *BackupRecord) BeforeCreate(tx *gorm.DB) error { if m.ID == "" { m.ID = uuid.New().String() }; return nil }
