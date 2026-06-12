package service

import (
	"time"

	"github.com/alaikis/opentether/internal/agent"
	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/storage"
	"gorm.io/gorm"
)

type Services struct {
	Auth         *AuthService
	User         *UserService
	UserGroup    *UserGroupService
	Provider     *ProviderService
	DataSource   *DataSourceService
	Skill        *SkillService
	Task         *TaskService
	IM           *IMService
	Log          *LogService
	Agent        *AgentService
	Conversation *ConversationService
	// 新增服务
	ApiKey        *ApiKeyService            // API 密钥管理
	SkillMarkdown *SkillFromMarkdownService // MD 文件解析生成 Skill
	MCP           *MCPService               // MCP 协议集成
	PDF           *PDFService               // PDF 报表生成
	MarkdownPDF   *MarkdownPDFService       // Markdown 转 PDF
	Experience    *ExperienceService        // 经验管理
	SQLAudit      *SQLAuditService          // SQL 审计
	Soul          *SoulService              // Soul/记忆管理
	Storage       storage.Driver            // 对象存储
}

func NewServices(db *gorm.DB, cfg *config.Config, store storage.Driver) *Services {
	mcp := NewMCPService(db)
	skillSvc := NewSkillService(db, cfg.Server.Mode == "development", store)
	agentSvc := NewAgentService(db, cfg, store)
	agentSvc.SetMCPProvider(mcp)
	go mcp.StartEnabledServers()
	go skillSvc.ValidateContextMDFiles()
	go func() {
		agent.RecoverRuntimeJobs(db)
		time.Sleep(2 * time.Second)
		agentSvc.AutoRecoverRuntimeJobs(10)
	}()

	return &Services{
		Auth:         NewAuthService(db, cfg),
		User:         NewUserService(db),
		UserGroup:    NewUserGroupService(db),
		Provider:     NewProviderService(db),
		DataSource:   NewDataSourceService(db, nil), // nil - LLM client not available in service layer
		Skill:        skillSvc,
		Task:         NewTaskService(db, cfg),
		IM:           NewIMService(db),
		Log:          NewLogService(db),
		Agent:        agentSvc,
		Conversation: NewConversationService(db),
		// 新增服务初始化
		ApiKey:        NewApiKeyService(db),
		SkillMarkdown: NewSkillFromMarkdownService(db),
		MCP:           mcp,
		PDF:           NewPDFService(),
		MarkdownPDF:   NewMarkdownPDFService(),
		Experience:    NewExperienceService(db, cfg),
		SQLAudit:      NewSQLAuditService(db),
		Soul:          NewSoulService(db, store),
		Storage:       store,
	}
}

// Input types
type CreateUserInput struct {
	CompanyID          string   `json:"company_id"`
	GlobalUserID       string   `json:"global_user_id"`
	ExternalEmployeeID string   `json:"external_employee_id"`
	Name               string   `json:"name"`
	Email              string   `json:"email"`
	Department         string   `json:"department"`
	Position           string   `json:"position"`
	Role               string   `json:"role"`
	SSOID              string   `json:"sso_id"`
	Password           string   `json:"password"`
	Groups             []string `json:"groups"`
}

type UpdateUserInput struct {
	CompanyID          string `json:"company_id"`
	ExternalEmployeeID string `json:"external_employee_id"`
	Name               string `json:"name"`
	Email              string `json:"email"`
	Department         string `json:"department"`
	Position           string `json:"position"`
	Role               string `json:"role"`
	Status             string `json:"status"`
	Password           string `json:"password"`
}

type CreateUserGroupInput struct {
	CompanyID       string `json:"company_id"`
	GroupName       string `json:"group_name"`
	GroupCode       string `json:"group_code"`
	ExternalGroupID string `json:"external_group_id"`
	Description     string `json:"description"`
	DataAccessScope string `json:"data_access_scope"`
	ParentGroupID   string `json:"parent_group_id"`
}

type UpdateUserGroupInput struct {
	CompanyID       string `json:"company_id"`
	ExternalGroupID string `json:"external_group_id"`
	GroupName       string `json:"group_name"`
	Description     string `json:"description"`
	DataAccessScope string `json:"data_access_scope"`
}

type CreateProviderInput struct {
	ProviderType string `json:"provider_type"`
	ProviderName string `json:"provider_name"`
	APIBase      string `json:"api_base"`
	APIKey       string `json:"api_key"`
	Model        string `json:"model"`
	Enabled      bool   `json:"enabled"`
	Priority     int    `json:"priority"`
	Config       string `json:"config"`
}

type UpdateProviderInput struct {
	ProviderName string `json:"provider_name"`
	APIBase      string `json:"api_base"`
	APIKey       string `json:"api_key"`
	Model        string `json:"model"`
	Enabled      bool   `json:"enabled"`
	Priority     int    `json:"priority"`
	Config       string `json:"config"`
}

type CreateDataSourceInput struct {
	Name       string `json:"name"`
	SourceType string `json:"source_type"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	Password   string `json:"password"`
	Database   string `json:"database"`
	Connection string `json:"connection"`
	Enabled    bool   `json:"enabled"`
}

type UpdateDataSourceInput struct {
	Name           string `json:"name"`
	SourceType     string `json:"source_type"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	User           string `json:"user"`
	Password       string `json:"password"`
	Database       string `json:"database"`
	Connection     string `json:"connection"`
	Enabled        bool   `json:"enabled"`
	TableRelations string `json:"table_relations"`
	SchemaInfo     string `json:"schema_info"`
}

type CreateSkillInput struct {
	Name            string   `json:"name"`
	SkillType       string   `json:"skill_type"`
	Description     string   `json:"description"`
	Keywords        []string `json:"keywords"`
	Category        string   `json:"category"`
	Enabled         bool     `json:"enabled"`
	Config          string   `json:"config"`
	PromptTemplate  string   `json:"prompt_template"`
	AllowedGroups   string   `json:"allowed_groups"`
	DataScope       string   `json:"data_scope"`
	RequireApproval bool     `json:"require_approval"`
}

type UpdateSkillInput struct {
	Name            string   `json:"name"`
	SkillType       string   `json:"skill_type"`
	Description     string   `json:"description"`
	Keywords        []string `json:"keywords"`
	Category        string   `json:"category"`
	Enabled         bool     `json:"enabled"`
	Config          string   `json:"config"`
	PromptTemplate  string   `json:"prompt_template"`
	AllowedGroups   string   `json:"allowed_groups"`
	DataScope       string   `json:"data_scope"`
	RequireApproval bool     `json:"require_approval"`
}

type CreateTaskInput struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	CronExpression string `json:"cron_expression"`
	ExecutorType   string `json:"executor_type"`
	ScriptContent  string `json:"script_content"`
	ScriptPath     string `json:"script_path"`
	Parameters     string `json:"parameters"`
	Enabled        bool   `json:"enabled"`
}

type UpdateTaskInput struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	CronExpression string `json:"cron_expression"`
	ExecutorType   string `json:"executor_type"`
	ScriptContent  string `json:"script_content"`
	ScriptPath     string `json:"script_path"`
	Parameters     string `json:"parameters"`
	Enabled        bool   `json:"enabled"`
}

type CreateIMConfigInput struct {
	Platform    string `json:"platform"`
	Name        string `json:"name"`
	AppID       string `json:"app_id"`
	AppSecret   string `json:"app_secret"`
	Token       string `json:"token"`
	WebhookURL  string `json:"webhook_url"`
	CallbackURL string `json:"callback_url"`
	Enabled     bool   `json:"enabled"`
	Config      string `json:"config"`
}

type UpdateIMConfigInput struct {
	Name        string `json:"name"`
	AppID       string `json:"app_id"`
	AppSecret   string `json:"app_secret"`
	Token       string `json:"token"`
	WebhookURL  string `json:"webhook_url"`
	CallbackURL string `json:"callback_url"`
	Enabled     bool   `json:"enabled"`
	Config      string `json:"config"`
}
