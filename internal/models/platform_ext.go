package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AccessPolicy struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name      string    `json:"name" gorm:"type:varchar(120);not null"`
	Scope     string    `json:"scope" gorm:"type:varchar(50);index"`
	Resource  string    `json:"resource" gorm:"type:varchar(255);index"`
	Effect    string    `json:"effect" gorm:"type:varchar(20);default:allow"`
	RulesJSON string    `json:"rules_json" gorm:"type:text"`
	Enabled   bool      `json:"enabled" gorm:"default:true;index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (m *AccessPolicy) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

type PrecomputeJob struct {
	ID          string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name        string     `json:"name" gorm:"type:varchar(120);not null"`
	TargetTable string     `json:"target_table" gorm:"type:varchar(120);index"`
	SQL         string     `json:"sql" gorm:"type:text;not null"`
	Schedule    string     `json:"schedule" gorm:"type:varchar(80)"`
	Status      string     `json:"status" gorm:"type:varchar(30);default:pending;index"`
	LastRunAt   *time.Time `json:"last_run_at"`
	LastError   string     `json:"last_error" gorm:"type:text"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (m *PrecomputeJob) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}
