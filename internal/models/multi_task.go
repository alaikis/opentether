package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MultiTaskPlanModel struct {
	ID         string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID     string    `json:"user_id" gorm:"type:varchar(36);index"`
	Original   string    `json:"original" gorm:"type:text"`
	SubTasks   string    `json:"sub_tasks" gorm:"type:text"`
	TotalSteps int       `json:"total_steps"`
	IsTree     bool      `json:"is_tree"`
	Status     string    `json:"status" gorm:"default:pending"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (m *MultiTaskPlanModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

type MultiTaskExecutionModel struct {
	ID         string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	PlanID     string    `json:"plan_id" gorm:"type:varchar(36);index"`
	UserID     string    `json:"user_id" gorm:"type:varchar(36);index"`
	Summary    string    `json:"summary" gorm:"type:text"`
	Data       string    `json:"data" gorm:"type:text"`
	Status     string    `json:"status" gorm:"default:running"`
	CreatedAt  time.Time `json:"created_at"`
	FinishedAt time.Time `json:"finished_at"`
}

func (m *MultiTaskExecutionModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}
