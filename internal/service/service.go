package service

import (
	"github.com/alaikis/opentether/internal/config"
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
}

func NewServices(db *gorm.DB, cfg *config.Config) *Services {
	return &Services{
		Auth:         NewAuthService(db, cfg),
		User:         NewUserService(db),
		UserGroup:    NewUserGroupService(db),
		Provider:     NewProviderService(db),
		DataSource:   NewDataSourceService(db, nil), // nil - LLM client not available in service layer
		Skill:        NewSkillService(db),
		Task:         NewTaskService(db, cfg),
		IM:           NewIMService(db),
		Log:          NewLogService(db),
		Agent:        NewAgentService(db, cfg),
		Conversation: NewConversationService(db),
	}
}

// Input types
type CreateUserInput struct {
	GlobalUserID string   `json:"global_user_id"`
	Name         string   `json:"name"`
	Email        string   `json:"email"`
	Department   string   `json:"department"`
	Position     string   `json:"position"`
	SSOID        string   `json:"sso_id"`
	Groups       []string `json:"groups"`
}

type UpdateUserInput struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Department string `json:"department"`
	Position   string `json:"position"`
	Status     string `json:"status"`
}

type CreateUserGroupInput struct {
	GroupName       string `json:"group_name"`
	GroupCode       string `json:"group_code"`
	Description     string `json:"description"`
	DataAccessScope string `json:"data_access_scope"`
	ParentGroupID   string `json:"parent_group_id"`
}

type UpdateUserGroupInput struct {
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

type CreateSkillInput struct {
	Name            string `json:"name"`
	SkillType       string `json:"skill_type"`
	Description     string `json:"description"`
	Keywords        string `json:"keywords"`
	Category        string `json:"category"`
	Enabled         bool   `json:"enabled"`
	Config          string `json:"config"`
	PromptTemplate  string `json:"prompt_template"`
	AllowedGroups   string `json:"allowed_groups"`
	DataScope       string `json:"data_scope"`
	RequireApproval bool   `json:"require_approval"`
}

type UpdateSkillInput struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	Keywords        string `json:"keywords"`
	Category        string `json:"category"`
	Enabled         bool   `json:"enabled"`
	Config          string `json:"config"`
	PromptTemplate  string `json:"prompt_template"`
	AllowedGroups   string `json:"allowed_groups"`
	DataScope       string `json:"data_scope"`
	RequireApproval bool   `json:"require_approval"`
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
