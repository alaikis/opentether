package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AgentTaskGraph struct {
	ID              string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	ParentGraphID   string    `json:"parent_graph_id" gorm:"type:varchar(36);index"`
	ConversationID  string    `json:"conversation_id" gorm:"type:varchar(36);index"`
	UserID          string    `json:"user_id" gorm:"type:varchar(36);index"`
	Status          string    `json:"status" gorm:"type:varchar(30);default:pending;index"`
	Priority        int       `json:"priority" gorm:"default:5;index"`
	MaxDuration     int       `json:"max_duration_sec" gorm:"default:3600"`
	Progress        int       `json:"progress" gorm:"default:0"`
	Goal            string    `json:"goal" gorm:"type:text"`
	PlanJSON        string    `json:"plan_json" gorm:"type:text"`
	StateSchemaJSON string    `json:"state_schema_json" gorm:"type:text"`
	Summary         string    `json:"summary" gorm:"type:text"`
	Error           string    `json:"error" gorm:"type:text"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (m *AgentTaskGraph) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

type AgentTaskNode struct {
	ID              string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	GraphID         string     `json:"graph_id" gorm:"type:varchar(36);index;not null"`
	SubgraphID      string     `json:"subgraph_id" gorm:"type:varchar(36);index"`
	ParentID        string     `json:"parent_id" gorm:"type:varchar(36);index"`
	Type            string     `json:"type" gorm:"type:varchar(50);index"`
	Name            string     `json:"name" gorm:"type:varchar(200)"`
	InputJSON       string     `json:"input_json" gorm:"type:text"`
	DependsOnJSON   string     `json:"depends_on_json" gorm:"type:text"`
	ConditionJSON   string     `json:"condition_json" gorm:"type:text"`
	Status          string     `json:"status" gorm:"type:varchar(30);default:pending;index"`
	ReviewStatus    string     `json:"review_status" gorm:"type:varchar(30);default:none"`
	ReviewComment   string     `json:"review_comment" gorm:"type:text"`
	OutputRef       string     `json:"output_ref" gorm:"type:varchar(1000)"`
	Summary         string     `json:"summary" gorm:"type:text"`
	CheckpointJSON  string     `json:"checkpoint_json" gorm:"type:text"`
	Error           string     `json:"error" gorm:"type:text"`
	RetryConfigJSON string     `json:"retry_config_json" gorm:"type:text"`
	RetryCount      int        `json:"retry_count" gorm:"default:0"`
	ExecutionGroup  string     `json:"execution_group" gorm:"type:varchar(50)"`
	MaxParallel     int        `json:"max_parallel" gorm:"default:1"`
	StartedAt       *time.Time `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (m *AgentTaskNode) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

type AgentTaskOutput struct {
	ID          string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	NodeID      string    `json:"node_id" gorm:"type:varchar(36);index;not null"`
	Type        string    `json:"type" gorm:"type:varchar(50);index"`
	ContentJSON string    `json:"content_json" gorm:"type:text"`
	StorageURL  string    `json:"storage_url" gorm:"type:varchar(1000)"`
	Summary     string    `json:"summary" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
}

func (m *AgentTaskOutput) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

type WebhookConfig struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name      string    `json:"name" gorm:"type:varchar(120);not null"`
	URL       string    `json:"url" gorm:"type:varchar(1000)"`
	Secret    string    `json:"secret" gorm:"type:varchar(500)"`
	Events    string    `json:"events" gorm:"type:text"`
	Enabled   bool      `json:"enabled" gorm:"default:true;index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (m *WebhookConfig) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

type WebhookDeliveryLog struct {
	ID          string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	ConfigID    string    `json:"config_id" gorm:"type:varchar(36);index;not null"`
	Event       string    `json:"event" gorm:"type:varchar(50);index"`
	PayloadJSON string    `json:"payload_json" gorm:"type:text"`
	Status      string    `json:"status" gorm:"type:varchar(20);index"`
	StatusCode  int       `json:"status_code"`
	Response    string    `json:"response" gorm:"type:text"`
	Error       string    `json:"error" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
}

func (m *WebhookDeliveryLog) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}
