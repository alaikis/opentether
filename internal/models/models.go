package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// 用户角色常量
const (
	RoleAdmin = "admin" // 管理员：完全访问
	RoleUser  = "user"  // 普通用户：基础功能
	RoleGuest = "guest" // 访客：只读
)

// 用户权限常量
const (
	PermissionRead   = "read"   // 读取
	PermissionWrite  = "write"  // 写入
	PermissionDelete = "delete" // 删除
	PermissionAdmin  = "admin"  // 管理
)

type User struct {
	ID           string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	CompanyID    string     `json:"company_id" gorm:"type:varchar(100);index"`
	GlobalUserID       string     `json:"global_user_id" gorm:"type:varchar(100);uniqueIndex;not null"`
	ExternalEmployeeID string     `json:"external_employee_id" gorm:"type:varchar(100);index"` // 公司/外部系统员工识别号
	Name               string     `json:"name" gorm:"type:varchar(255);not null"`
	Email        string     `json:"email" gorm:"type:varchar(255);uniqueIndex"`
	Department   string     `json:"department" gorm:"type:varchar(100)"`
	Position     string     `json:"position" gorm:"type:varchar(100)"`
	Role         string     `json:"role" gorm:"type:varchar(20);default:user"` // admin, user, guest
	Permissions  string     `json:"permissions" gorm:"type:varchar(500)"`      // JSON: 权限列表
	SSOID        string     `json:"sso_id" gorm:"type:varchar(100)"`
	Status       string     `json:"status" gorm:"type:varchar(20);default:active"` // active, inactive, suspended
	PasswordHash string     `json:"-" gorm:"type:varchar(255)"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedBy    string     `json:"created_by" gorm:"type:varchar(36)"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Relations
	Groups        []UserGroup    `json:"groups" gorm:"many2many:user_group_members;"`
	Conversations []Conversation `json:"conversations" gorm:"foreignKey:UserID"`
	ImBindings    []ImBinding    `json:"im_bindings" gorm:"foreignKey:UserID"`
	SkillAccess   []SkillAccess  `json:"skill_access" gorm:"foreignKey:UserID"`
}

// IsAdmin 检查是否为管理员
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// HasPermission 检查是否有指定权限
func (u *User) HasPermission(perm string) bool {
	// 管理员拥有所有权限
	if u.IsAdmin() {
		return true
	}

	// 检查用户权限列表（JSON 数组格式）
	if u.Permissions != "" {
		var perms []string
		if err := json.Unmarshal([]byte(u.Permissions), &perms); err == nil {
			for _, p := range perms {
				if p == perm || p == PermissionAdmin {
					return true
				}
			}
		}
	}

	// 检查用户组权限
	for _, group := range u.Groups {
		if group.GroupCode == "admin" || group.GroupCode == "Administrators" {
			return true
		}
	}

	return false
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

type UserGroup struct {
	ID              string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	CompanyID       string    `json:"company_id" gorm:"type:varchar(100);index"`
	GroupName       string    `json:"group_name" gorm:"type:varchar(100);not null"`
	GroupCode       string    `json:"group_code" gorm:"type:varchar(50);uniqueIndex;not null"`
	ExternalGroupID string    `json:"external_group_id" gorm:"type:varchar(100);index"` // 公司/外部系统用户组识别号
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
	Host           string    `json:"host" gorm:"type:varchar(100)"`       // 数据库主机
	Port           int       `json:"port"`                                // 端口
	User           string    `json:"user" gorm:"type:varchar(100)"`       // 用户名
	Password       string    `json:"password" gorm:"type:varchar(500)"`   // 密码（加密存储）
	Database       string    `json:"database" gorm:"type:varchar(100)"`   // 数据库名
	Connection     string    `json:"connection" gorm:"type:text"`         // 连接字符串（兼容旧版本）
	SchemaInfo     string    `json:"schema_info" gorm:"type:text"`        // JSON schema - 表结构信息
	TableRelations string    `json:"table_relations" gorm:"type:text"`    // JSON - 表关系信息
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

// SkillRuntimeMemory stores validated runtime learnings for a Skill, especially Text2SQL table/field/relation/query patterns.
type RouteExample struct {
	ID         string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Text       string    `json:"text" gorm:"type:text"`
	Route      string    `json:"route" gorm:"type:varchar(50);index"` // fast_local, fast_chat, fast_text2sql, agent_loop
	Intent     string    `json:"intent" gorm:"type:varchar(100);index"`
	Source     string    `json:"source" gorm:"type:varchar(50);index"`                // builtin, runtime, admin, rejected
	Status     string    `json:"status" gorm:"type:varchar(20);index;default:active"` // active, pending, rejected
	Confidence float64   `json:"confidence" gorm:"default:0.8"`
	UseCount   int       `json:"use_count" gorm:"default:1"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (r *RouteExample) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	if r.Source == "" {
		r.Source = "runtime"
	}
	if r.Status == "" {
		r.Status = "pending"
	}
	if r.Confidence == 0 {
		r.Confidence = 0.7
	}
	if r.UseCount == 0 {
		r.UseCount = 1
	}
	return nil
}

type SkillRuntimeMemory struct {
	ID           string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	SkillID      string    `json:"skill_id" gorm:"type:varchar(36);index"`
	DataSourceID string    `json:"data_source_id" gorm:"type:varchar(36);index"`
	Type         string    `json:"type" gorm:"type:varchar(50);index"` // table_usage, table_relation, metric_rule, sql_pattern
	Key          string    `json:"key" gorm:"type:varchar(255);index"`
	Content      string    `json:"content" gorm:"type:text"`
	Confidence   float64   `json:"confidence" gorm:"default:0.5"`
	UseCount     int       `json:"use_count" gorm:"default:1"`
	Source       string    `json:"source" gorm:"type:varchar(50);default:runtime"` // runtime, confirmed, admin
	LastUsedAt   time.Time `json:"last_used_at" gorm:"index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (s *SkillInvocation) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

func (m *SkillRuntimeMemory) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	if m.Confidence == 0 {
		m.Confidence = 0.5
	}
	if m.UseCount == 0 {
		m.UseCount = 1
	}
	if m.Source == "" {
		m.Source = "runtime"
	}
	if m.LastUsedAt.IsZero() {
		m.LastUsedAt = time.Now()
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

type SQLAudit struct {
	ID           string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID       string     `json:"user_id" gorm:"type:varchar(36);index"`
	SkillID      string     `json:"skill_id" gorm:"type:varchar(36)"`
	Question     string     `json:"question" gorm:"type:text"`
	GeneratedSQL string     `json:"generated_sql" gorm:"type:text"`
	DataSourceID string     `json:"data_source_id" gorm:"type:varchar(36)"`
	Status       string     `json:"status" gorm:"type:varchar(20);default:pending"` // pending / approved / rejected / executed
	ApprovedBy   string     `json:"approved_by" gorm:"type:varchar(36)"`
	ApprovedAt   *time.Time `json:"approved_at"`
	RejectReason string     `json:"reject_reason" gorm:"type:text"`
	RowCount     int        `json:"row_count"`
	ExecTime     string     `json:"exec_time"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (s *SQLAudit) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

type MCPConfig struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name      string    `json:"name" gorm:"type:varchar(100)"`
	Command   string    `json:"command" gorm:"type:varchar(500)"`
	Args      string    `json:"args" gorm:"type:text"` // JSON array
	Env       string    `json:"env" gorm:"type:text"`  // JSON object
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	Status    string    `json:"status" gorm:"type:varchar(20)"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (m *MCPConfig) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
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

// ConversationState stores compressed short-term memory and task working memory for one conversation.
type ConversationState struct {
	ID             string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	ConversationID string     `json:"conversation_id" gorm:"type:varchar(36);uniqueIndex;not null"`
	UserID         string     `json:"user_id" gorm:"type:varchar(36);index"`
	Status         string     `json:"status" gorm:"type:varchar(20);index;default:active"` // active, warm_idle, archived
	Summary        string     `json:"summary" gorm:"type:text"`
	ActiveTaskID   string     `json:"active_task_id" gorm:"type:varchar(100);index"`
	EntitiesJSON   string     `json:"entities_json" gorm:"type:text"`        // active entity slots, e.g. employee/product/customer
	TasksJSON      string     `json:"tasks_json" gorm:"type:text"`           // bounded task working memory
	ArchivePath    string     `json:"archive_path" gorm:"type:varchar(500)"` // optional markdown archive path
	Version        int        `json:"version" gorm:"default:1"`
	LastActiveAt   time.Time  `json:"last_active_at" gorm:"index"`
	ArchivedAt     *time.Time `json:"archived_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CreatedAt      time.Time  `json:"created_at"`

	Conversation *Conversation `json:"conversation,omitempty" gorm:"foreignKey:ConversationID"`
	User         *User         `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

func (s *ConversationState) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	if s.Version == 0 {
		s.Version = 1
	}
	if s.Status == "" {
		s.Status = "active"
	}
	if s.LastActiveAt.IsZero() {
		s.LastActiveAt = time.Now()
	}
	return nil
}

type RuntimeJob struct {
	ID             string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID         string     `json:"user_id" gorm:"type:varchar(36);index"`
	ConversationID string     `json:"conversation_id" gorm:"type:varchar(36);index"`
	SkillID        string     `json:"skill_id" gorm:"type:varchar(36);index"`
	JobType        string     `json:"job_type" gorm:"type:varchar(50);index"`
	Status         string     `json:"status" gorm:"type:varchar(20);index"` // pending,running,paused,recovering,succeeded,failed,cancelled
	Input          string     `json:"input" gorm:"type:text"`
	Output         string     `json:"output" gorm:"type:text"`
	Error          string     `json:"error" gorm:"type:text"`
	CurrentStep    int        `json:"current_step"`
	MaxSteps       int        `json:"max_steps"`
	Recoverable    bool       `json:"recoverable" gorm:"default:true"`
	LeaseOwner     string     `json:"lease_owner" gorm:"type:varchar(100);index"`
	LeaseExpiresAt time.Time  `json:"lease_expires_at" gorm:"index"`
	StartedAt      time.Time  `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type RuntimeCheckpoint struct {
	ID             string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	JobID          string    `json:"job_id" gorm:"type:varchar(36);index"`
	Step           int       `json:"step" gorm:"index"`
	Type           string    `json:"type" gorm:"type:varchar(50);index"`
	State          string    `json:"state" gorm:"type:text"`
	IdempotencyKey string    `json:"idempotency_key" gorm:"type:varchar(255);index"`
	CreatedAt      time.Time `json:"created_at"`
}

func (j *RuntimeJob) BeforeCreate(tx *gorm.DB) error {
	if j.ID == "" {
		j.ID = uuid.New().String()
	}
	if j.Status == "" {
		j.Status = "pending"
	}
	if j.StartedAt.IsZero() {
		j.StartedAt = time.Now()
	}
	return nil
}

func (c *RuntimeCheckpoint) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
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

// ApiKey API 密钥 - 用于多系统集成，代用户发起请求
type ApiKey struct {
	ID         string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID     string     `json:"user_id" gorm:"type:varchar(36);index;not null"`
	Name       string     `json:"name" gorm:"type:varchar(100);not null"`
	Key        string     `json:"-" gorm:"type:varchar(64);uniqueIndex;not null"` // 完整 key（hash 存储）
	KeyPrefix  string     `json:"key_prefix" gorm:"type:varchar(16)"`             // 前缀显示: ot_sk_xxxx
	Scopes     string     `json:"scopes" gorm:"type:varchar(500)"`                // JSON: 权限范围
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`

	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (k *ApiKey) BeforeCreate(tx *gorm.DB) error {
	if k.ID == "" {
		k.ID = uuid.New().String()
	}
	return nil
}

// IMBindingToken 临时 IM 绑定 token（用于扫码绑定验证）
type IMBindingToken struct {
	ID         string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID     string    `json:"user_id" gorm:"type:varchar(36);index;not null"`
	ImConfigID string    `json:"im_config_id" gorm:"type:varchar(36);index;not null"`
	Token      string    `json:"token" gorm:"type:varchar(64);uniqueIndex;not null"`
	Status     string    `json:"status" gorm:"type:varchar(20);default:pending"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`

	User     *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	ImConfig *ImConfig `json:"im_config,omitempty" gorm:"foreignKey:ImConfigID"`
}

func (t *IMBindingToken) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

// AgentTask 下发到独立 Agent 的任务
type AgentTask struct {
	ID          string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	AgentID     string     `json:"agent_id" gorm:"type:varchar(36);index"`
	TaskType    string     `json:"task_type"`
	Payload     string     `json:"payload" gorm:"type:text"`
	Status      string     `json:"status" gorm:"type:varchar(20);default:pending"`
	Result      string     `json:"result" gorm:"type:text"`
	Error       string     `json:"error" gorm:"type:text"`
	AssignedAt  *time.Time `json:"assigned_at"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (t *AgentTask) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = GenerateID()
	}
	return nil
}

// AgentPairing 配对码记录
type AgentPairing struct {
	ID        string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	Code      string     `json:"code" gorm:"type:varchar(16);uniqueIndex;not null"`
	AgentID   string     `json:"agent_id" gorm:"type:varchar(36);index"`
	Status    string     `json:"status" gorm:"type:varchar(20);default:pending"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
	PairedAt  *time.Time `json:"paired_at"`
}

func (p *AgentPairing) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = GenerateID()
	}
	return nil
}

// GenerateID 生成通用唯一 ID（短格式）
func GenerateID() string {
	return uuid.New().String()
}

// GeneratePairingCode 生成配对码（格式: ot_p_xxxx-xxxx）
func GeneratePairingCode() string {
	id := uuid.New().String()
	return "ot_p_" + id[:8]
}

// AgentExperience 智能体经验记录
type AgentExperience struct {
	ID             string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name           string     `json:"name" gorm:"type:varchar(200)"`
	Description    string     `json:"description" gorm:"type:text"`
	TriggerPattern string     `json:"trigger_pattern" gorm:"type:text"`
	TriggerVector  string     `json:"trigger_vector" gorm:"type:text"`
	Steps          string     `json:"steps" gorm:"type:text"`
	Scope          string     `json:"scope" gorm:"type:varchar(50)"`
	Status         string     `json:"status" gorm:"type:varchar(20);default:pending_review"`
	UsageCount     int        `json:"usage_count" gorm:"default:0"`
	SuccessCount   int        `json:"success_count" gorm:"default:0"`
	AvgTokensSaved int        `json:"avg_tokens_saved" gorm:"default:0"`
	CreatedBy      string     `json:"created_by" gorm:"type:varchar(36);index"`
	ReviewedBy     string     `json:"reviewed_by" gorm:"type:varchar(36)"`
	ReviewNote     string     `json:"review_note" gorm:"type:text"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ReviewedAt     *time.Time `json:"reviewed_at"`

	CreatedByUser *User `json:"created_by_user,omitempty" gorm:"foreignKey:CreatedBy"`
}

func (e *AgentExperience) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = GenerateID()
	}
	return nil
}

// UserMemory 用户长期记忆
type UserMemory struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID    string    `json:"user_id" gorm:"type:varchar(36);index;not null"`
	Type      string    `json:"type" gorm:"type:varchar(50)"`
	Key       string    `json:"key" gorm:"type:varchar(200)"`
	Content   string    `json:"content" gorm:"type:text"`
	Priority  int       `json:"priority" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (m *UserMemory) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = GenerateID()
	}
	return nil
}

// GroupMemory 用户组共享记忆
type GroupMemory struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	GroupID   string    `json:"group_id" gorm:"type:varchar(36);index;not null"`
	Type      string    `json:"type" gorm:"type:varchar(50)"`
	Key       string    `json:"key" gorm:"type:varchar(200)"`
	Content   string    `json:"content" gorm:"type:text"`
	Priority  int       `json:"priority" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Group *UserGroup `json:"group,omitempty" gorm:"foreignKey:GroupID"`
}

func (m *GroupMemory) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = GenerateID()
	}
	return nil
}

// UserProfile 用户 Soul/Profile（Letta-inspired Core Memory 人类块）
type UserProfile struct {
	ID                 string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID             string    `json:"user_id" gorm:"type:varchar(36);uniqueIndex"`
	Persona            string    `json:"persona" gorm:"type:text"`          // AI 助手的人格描述
	Human              string    `json:"human" gorm:"type:text"`            // 用户的人类描述
	Preferences        string    `json:"preferences" gorm:"type:text"`      // 用户偏好 JSON
	PreferredSkills    string    `json:"preferred_skills" gorm:"type:text"` // 常用 Skill IDs JSON
	LanguagePreference string    `json:"language_preference" gorm:"type:varchar(20);default:zh-CN"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (p *UserProfile) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = GenerateID()
	}
	if p.LanguagePreference == "" {
		p.LanguagePreference = "zh-CN"
	}
	return nil
}

// CompanyProfile 公司级 Soul（Letta-inspired 企业级 Core Memory）
type CompanyProfile struct {
	ID            string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name          string    `json:"name" gorm:"type:varchar(200)"`
	Persona       string    `json:"persona" gorm:"type:text"`    // 企业的 AI 人格
	BrandTone     string    `json:"brand_tone" gorm:"type:text"` // 企业语调、合规规则
	Industry      string    `json:"industry" gorm:"type:varchar(100)"`
	DefaultConfig string    `json:"default_config" gorm:"type:text"` // JSON
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AgentScript 智能体脚本
type AgentScript struct {
	ID           string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	SkillID      string     `json:"skill_id" gorm:"type:varchar(36);index"`
	ExperienceID string     `json:"experience_id" gorm:"type:varchar(36);index"`
	Name         string     `json:"name" gorm:"type:varchar(200)"`
	Language     string     `json:"language" gorm:"type:varchar(20)"` // bash, python
	Content      string     `json:"content" gorm:"type:text"`
	FilePath     string     `json:"file_path" gorm:"type:varchar(500)"` // 文件路径
	FileHash     string     `json:"file_hash" gorm:"type:varchar(64)"`  // SHA-256 文件哈希
	HashVerified bool       `json:"hash_verified" gorm:"default:true"`  // 哈希是否通过
	Description  string     `json:"description" gorm:"type:text"`
	IsPermanent  bool       `json:"is_permanent" gorm:"default:false"`
	ExpiresAt    *time.Time `json:"expires_at"`
	CreatedBy    string     `json:"created_by" gorm:"type:varchar(36)"`
	ExecCount    int        `json:"exec_count" gorm:"default:0"`
	LastExecAt   *time.Time `json:"last_exec_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (s *AgentScript) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = GenerateID()
	}
	if !s.IsPermanent && s.ExpiresAt == nil {
		t := time.Now().Add(30 * 24 * time.Hour)
		s.ExpiresAt = &t
	}
	return nil
}
