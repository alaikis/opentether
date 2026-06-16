package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EvalCase struct {
	ID                  string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name                string    `json:"name" gorm:"type:varchar(120);not null"`
	SkillID             string    `json:"skill_id" gorm:"type:varchar(36);index"`
	Question            string    `json:"question" gorm:"type:text;not null"`
	ExpectedContains    string    `json:"expected_contains" gorm:"type:text"`
	ExpectedSQLContains string    `json:"expected_sql_contains" gorm:"type:text"`
	Enabled             bool      `json:"enabled" gorm:"default:true;index"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (m *EvalCase) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

type EvalRun struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	CaseID    string    `json:"case_id" gorm:"type:varchar(36);index;not null"`
	SkillID   string    `json:"skill_id" gorm:"type:varchar(36);index"`
	Status    string    `json:"status" gorm:"type:varchar(20);index"`
	Passed    bool      `json:"passed" gorm:"index"`
	Output    string    `json:"output" gorm:"type:text"`
	Error     string    `json:"error" gorm:"type:text"`
	Duration  int64     `json:"duration_ms"`
	CreatedAt time.Time `json:"created_at"`
}

func (m *EvalRun) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	if m.Status == "" {
		m.Status = "pending"
	}
	return nil
}
