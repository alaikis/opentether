package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	GlobalUserID string    `json:"global_user_id" gorm:"type:varchar(100);uniqueIndex;not null"`
	Name         string    `json:"name" gorm:"type:varchar(255);not null"`
	Email        string    `json:"email" gorm:"type:varchar(255);uniqueIndex"`
	Department   string    `json:"department" gorm:"type:varchar(100)"`
	Position     string    `json:"position" gorm:"type:varchar(100)"`
	SSOID        string    `json:"sso_id" gorm:"type:varchar(100)"`
	Status       string    `json:"status" gorm:"type:varchar(20);default:active"` // active, inactive, suspended
	PasswordHash string    `json:"-" gorm:"type:varchar(255)"`
	CreatedBy    string    `json:"created_by" gorm:"type:varchar(36)"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relations
	Groups        []UserGroup    `json:"groups" gorm:"many2many:user_group_members;"`
	Conversations []Conversation `json:"conversations" gorm:"foreignKey:UserID"`
	ImBindings    []ImBinding    `json:"im_bindings" gorm:"foreignKey:UserID"`
	SkillAccess   []SkillAccess  `json:"skill_access" gorm:"foreignKey:UserID"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

type UserGroup struct {
	ID              string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	GroupName       string    `json:"group_name" gorm:"type:varchar(100);not null"`
	GroupCode       string    `json:"group_code" gorm:"type:varchar(50);uniqueIndex;not null"`
	Description     string    `json:"description" gorm:"type:text"`
	DataAccessScope string    `json:"data_access_scope" gorm:"type:varchar(20);default:self"` // all, self, department, custom
	DataAccessConds string    `json:"data_access_conds" gorm:"type:text"`                     // JSON conditions
	ParentGroupID   string    `json:"parent_group_id" gorm:"type:varchar(36)"`
	CreatedBy       string    `json:"created_by" gorm:"type:varchar(36)"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relations
	Members     []User        `json:"members" gorm:"many2many:user_group_members;"`
	Parent      *UserGroup    `json:"parent,omitempty" gorm:"foreignKey:ParentGroupID"`
	Children    []UserGroup   `json:"children,omitempty" gorm:"foreignKey:ParentGroupID"`
	SkillAccess []SkillAccess `json:"skill_access" gorm:"foreignKey:GroupID"`
}

func (g *UserGroup) BeforeCreate(tx *gorm.DB) error {
	if g.ID == "" {
		g.ID = uuid.New().String()
	}
	return nil
}

type Role struct {
	ID          string       `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name        string       `json:"name" gorm:"type:varchar(50);uniqueIndex;not null"`
	Description string       `json:"description" gorm:"type:text"`
	Permissions []Permission `json:"permissions" gorm:"many2many:role_permissions;"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

type Permission struct {
	ID           string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name         string     `json:"name" gorm:"type:varchar(100);uniqueIndex;not null"`
	ResourceType string     `json:"resource_type" gorm:"type:varchar(50)"` // user, provider, datasource, skill, etc.
	ResourceID   string     `json:"resource_id" gorm:"type:varchar(36)"`
	Action       string     `json:"action" gorm:"type:varchar(50)"` // read, write, execute, admin
	Conditions   string     `json:"conditions" gorm:"type:text"`    // JSON conditions
	ExpiresAt    *time.Time `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

type Provider struct {
	ID           string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	ProviderType string    `json:"provider_type" gorm:"type:varchar(50)"` // openai, azure, anthropic, local, etc.
	ProviderName string    `json:"provider_name" gorm:"type:varchar(100);not null"`
	APIBase      string    `json:"api_base" gorm:"type:varchar(500)"`
	APIKey       string    `json:"api_key" gorm:"type:varchar(500)"` // encrypted
	Model        string    `json:"model" gorm:"type:varchar(100)"`
	Enabled      bool      `json:"enabled" gorm:"default:true"`
	Priority     int       `json:"priority" gorm:"default:0"`
	Config       string    `json:"config" gorm:"type:text"` // JSON config
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (p *Provider) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

type DataSource struct {
	ID             string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name           string    `json:"name" gorm:"type:varchar(100);not null"`
	SourceType     string    `json:"source_type" gorm:"type:varchar(50)"` // mysql, postgres, mongodb, excel, csv, api
	Host           string    `json:"host" gorm:"type:varchar(100)"`        // 数据库主机
	Port           int       `json:"port"`                                  // 端口
	User           string    `json:"user" gorm:"type:varchar(100)"`        // 用户名
	Password       string    `json:"password" gorm:"type:varchar(500)"`    // 密码（加密存储）
	Database       string    `json:"database" gorm:"type:varchar(100)"`    // 数据库名
	Connection     string    `json:"connection" gorm:"type:text"`          // 连接字符串（兼容旧版本）
	SchemaInfo     string    `json:"schema_info" gorm:"type:text"`         // JSON schema - 表结构信息
	TableRelations string    `json:"table_relations" gorm:"type:text"`     // JSON - 表关系信息
	Enabled        bool      `json:"enabled" gorm:"default:true"`
	CreatedBy      string    `json:"created_by" gorm:"type:varchar(36)"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (d *DataSource) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

type Skill struct {
	ID              string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name            string    `json:"name" gorm:"type:varchar(100);not null"`
	SkillType       string    `json:"skill_type" gorm:"type:varchar(50)"` // chat, text2sql, file_process, api_caller, etc.
	Description     string    `json:"description" gorm:"type:text"`
	Keywords        string    `json:"keywords" gorm:"type:text"` // JSON array
	Category        string    `json:"category" gorm:"type:varchar(50)"`
	Enabled         bool      `json:"enabled" gorm:"default:true"`
	Config          string    `json:"config" gorm:"type:text"` // JSON config
	PromptTemplate  string    `json:"prompt_template" gorm:"type:text"`
	VectorIndex     []byte    `json:"vector_index" gorm:"type:blob"`
	VectorEnabled   bool      `json:"vector_enabled" gorm:"default:false"`
	AllowedGroups   string    `json:"allowed_groups" gorm:"type:text"`                // JSON array
	DataScope       string    `json:"data_scope" gorm:"type:varchar(20);default:all"` // all, self, department
	RequireApproval bool      `json:"require_approval" gorm:"default:false"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Invocations []SkillInvocation `json:"invocations" gorm:"foreignKey:SkillID"`
}

func (s *Skill) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

type SkillInvocation struct {
	ID         string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID     string    `json:"user_id" gorm:"type:varchar(36);index"`
	SkillID    string    `json:"skill_id" gorm:"type:varchar(36);index"`
	Input      string    `json:"input" gorm:"type:text"`         // JSON input
	Output     string    `json:"output" gorm:"type:text"`        // JSON output
	Status     string    `json:"status" gorm:"type:varchar(20)"` // success, failed, pending
	DurationMs int64     `json:"duration_ms"`
	ErrorMsg   string    `json:"error_msg" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *SkillInvocation) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

type SkillAccess struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID    string    `json:"user_id" gorm:"type:varchar(36);index"`
	GroupID   string    `json:"group_id" gorm:"type:varchar(36);index"`
	SkillID   string    `json:"skill_id" gorm:"type:varchar(36);index"`
	DataScope string    `json:"data_scope" gorm:"type:varchar(20)"` // all, self, department
	CreatedAt time.Time `json:"created_at"`
}

func (s *SkillAccess) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

type ImConfig struct {
	ID          string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Platform    string    `json:"platform" gorm:"type:varchar(50)"` // wecom, feishu, dingtalk, whatsapp
	Name        string    `json:"name" gorm:"type:varchar(100)"`
	AppID       string    `json:"app_id" gorm:"type:varchar(100)"`
	AppSecret   string    `json:"app_secret" gorm:"type:varchar(500)"` // encrypted
	Token       string    `json:"token" gorm:"type:varchar(500)"`      // encrypted
	WebhookURL  string    `json:"webhook_url" gorm:"type:varchar(500)"`
	CallbackURL string    `json:"callback_url" gorm:"type:varchar(500)"`
	Enabled     bool      `json:"enabled" gorm:"default:true"`
	Config      string    `json:"config" gorm:"type:text"` // JSON config
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Bindings []ImBinding `json:"bindings" gorm:"foreignKey:ImConfigID"`
}

func (i *ImConfig) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.New().String()
	}
	return nil
}

type ImBinding struct {
	ID           string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	ImConfigID   string     `json:"im_config_id" gorm:"type:varchar(36);index"`
	UserID       string     `json:"user_id" gorm:"type:varchar(36);index"`
	ImUserID     string     `json:"im_user_id" gorm:"type:varchar(100);index"`
	ImUserName   string     `json:"im_user_name" gorm:"type:varchar(100)"`
	BindingToken string     `json:"binding_token" gorm:"type:varchar(100);uniqueIndex"`
	Status       string     `json:"status" gorm:"type:varchar(20);default:active"` // active, inactive
	ExpiresAt    *time.Time `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	ImConfig *ImConfig `json:"im_config,omitempty" gorm:"foreignKey:ImConfigID"`
	User     *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (i *ImBinding) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.New().String()
	}
	return nil
}

type Conversation struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID    string    `json:"user_id" gorm:"type:varchar(36);index"`
	GroupID   string    `json:"group_id" gorm:"type:varchar(36);index"`
	Source    string    `json:"source" gorm:"type:varchar(20)"` // web, im, api
	Title     string    `json:"title" gorm:"type:varchar(255)"`
	Messages  []Message `json:"messages" gorm:"foreignKey:ConversationID"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	User  *User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Group *UserGroup `json:"group,omitempty" gorm:"foreignKey:GroupID"`
}

func (c *Conversation) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

type Message struct {
	ID             string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	ConversationID string    `json:"conversation_id" gorm:"type:varchar(36);index"`
	Role           string    `json:"role" gorm:"type:varchar(20)"` // user, assistant, system
	Content        string    `json:"content" gorm:"type:text"`
	Metadata       string    `json:"metadata" gorm:"type:text"` // JSON metadata
	SkillUsed      string    `json:"skill_used" gorm:"type:varchar(100)"`
	TokenCount     int       `json:"token_count"`
	CreatedAt      time.Time `json:"created_at"`

	Conversation *Conversation `json:"conversation,omitempty" gorm:"foreignKey:ConversationID"`
}

func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

type AuditLog struct {
	ID           string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Timestamp    time.Time `json:"timestamp" gorm:"index"`
	UserID       string    `json:"user_id" gorm:"type:varchar(36);index"`
	UserName     string    `json:"user_name" gorm:"type:varchar(100)"`
	Action       string    `json:"action" gorm:"type:varchar(100);index"`
	ResourceType string    `json:"resource_type" gorm:"type:varchar(50);index"`
	ResourceID   string    `json:"resource_id" gorm:"type:varchar(36)"`
	Details      string    `json:"details" gorm:"type:text"` // JSON details
	IPAddress    string    `json:"ip_address" gorm:"type:varchar(45)"`
	UserAgent    string    `json:"user_agent" gorm:"type:varchar(500)"`
	Status       string    `json:"status" gorm:"type:varchar(20)"` // success, fail
}

func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	if a.Timestamp.IsZero() {
		a.Timestamp = time.Now()
	}
	return nil
}

type ScheduledTask struct {
	ID             string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name           string     `json:"name" gorm:"type:varchar(100);not null"`
	Description    string     `json:"description" gorm:"type:text"`
	CronExpression string     `json:"cron_expression" gorm:"type:varchar(100)"`
	ExecutorType   string     `json:"executor_type" gorm:"type:varchar(50)"` // script, python, api
	ScriptContent  string     `json:"script_content" gorm:"type:text"`
	ScriptPath     string     `json:"script_path" gorm:"type:varchar(500)"`
	Parameters     string     `json:"parameters" gorm:"type:text"` // JSON parameters
	Enabled        bool       `json:"enabled" gorm:"default:true"`
	Status         string     `json:"status" gorm:"type:varchar(20);default:idle"` // idle, running, paused
	LastRunAt      *time.Time `json:"last_run_at"`
	NextRunAt      *time.Time `json:"next_run_at"`
	CreatedBy      string     `json:"created_by" gorm:"type:varchar(36)"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	Executions []TaskExecution `json:"executions" gorm:"foreignKey:TaskID"`
}

func (t *ScheduledTask) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

type TaskExecution struct {
	ID          string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	TaskID      string     `json:"task_id" gorm:"type:varchar(36);index"`
	Status      string     `json:"status" gorm:"type:varchar(20)"` // pending, running, success, failed
	Output      string     `json:"output" gorm:"type:text"`
	ErrorMsg    string     `json:"error_msg" gorm:"type:text"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	DurationMs  int64      `json:"duration_ms"`
	CreatedAt   time.Time  `json:"created_at"`

	Task *ScheduledTask `json:"task,omitempty" gorm:"foreignKey:TaskID"`
}

func (e *TaskExecution) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	return nil
}
