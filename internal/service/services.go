package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/agent"
	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/database"
	"github.com/alaikis/opentether/internal/embedding"
	"github.com/alaikis/opentether/internal/im"
	"github.com/alaikis/opentether/internal/llm"
	"github.com/alaikis/opentether/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewAuthService(db *gorm.DB, cfg *config.Config) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

func (s *AuthService) Login(username, password string) (string, error) {
	if s.db == nil {
		return "", errors.New("database not initialized, please complete setup first")
	}

	var user models.User

	// Try to find by email or global_user_id
	if err := s.db.Where("email = ? OR global_user_id = ?", username, username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("invalid credentials")
		}
		return "", err
	}

	if user.Status != "active" {
		return "", errors.New("user account is not active")
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	// 设置默认角色
	if user.Role == "" {
		if user.CreatedBy == "system" {
			user.Role = models.RoleAdmin
		} else {
			user.Role = models.RoleUser
		}
	}

	// 更新最后登录时间
	s.db.Model(&user).Update("last_login_at", time.Now())

	// Generate JWT with role
	token, err := s.generateToken(user.ID, user.GlobalUserID, user.Name, user.Role)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) RefreshToken(tokenString string) (string, error) {
	if s.db == nil {
		return "", errors.New("database not initialized, please complete setup first")
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.Security.JWT.Secret), nil
	})

	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}

	claims := token.Claims.(jwt.MapClaims)
	userID := claims["user_id"].(string)

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return "", errors.New("user not found")
	}

	if user.Status != "active" {
		return "", errors.New("user account is not active")
	}

	// 设置默认角色
	if user.Role == "" {
		if user.CreatedBy == "system" {
			user.Role = models.RoleAdmin
		} else {
			user.Role = models.RoleUser
		}
	}

	return s.generateToken(user.ID, user.GlobalUserID, user.Name, user.Role)
}

func (s *AuthService) generateToken(userID, globalUserID, name, role string) (string, error) {
	expire := time.Now().Add(24 * time.Hour)

	claims := jwt.MapClaims{
		"user_id":        userID,
		"global_user_id": globalUserID,
		"name":           name,
		"role":           role,
		"exp":            expire.Unix(),
		"iat":            time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Security.JWT.Secret))
}

func (s *AuthService) ValidateToken(tokenString string) (map[string]interface{}, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.Security.JWT.Secret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return token.Claims.(jwt.MapClaims), nil
}

// UserService handles user operations
type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) List() ([]models.User, error) {
	var users []models.User
	err := s.db.Preload("Groups").Find(&users).Error
	return users, err
}

func (s *UserService) Create(input CreateUserInput) (*models.User, error) {
	// Hash password (use default password if not provided)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := models.User{
		GlobalUserID: input.GlobalUserID,
		Name:         input.Name,
		Email:        input.Email,
		Department:   input.Department,
		Position:     input.Position,
		SSOID:        input.SSOID,
		Status:       "active",
		PasswordHash: string(passwordHash),
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	// Add to groups
	if len(input.Groups) > 0 {
		var groups []models.UserGroup
		s.db.Find(&groups, input.Groups)
		s.db.Model(&user).Association("Groups").Append(groups)
	}

	return &user, nil
}

func (s *UserService) Update(id string, input UpdateUserInput) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"name":       input.Name,
		"email":      input.Email,
		"department": input.Department,
		"position":   input.Position,
		"status":     input.Status,
	}

	if err := s.db.Model(&user).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *UserService) Delete(id string) error {
	return s.db.Delete(&models.User{}, id).Error
}

func (s *UserService) BatchCreate(inputs []CreateUserInput) ([]models.User, error) {
	var users []models.User
	for _, input := range inputs {
		user, err := s.Create(input)
		if err != nil {
			return users, err
		}
		users = append(users, *user)
	}
	return users, nil
}

// UserGroupService handles user group operations
type UserGroupService struct {
	db *gorm.DB
}

func NewUserGroupService(db *gorm.DB) *UserGroupService {
	return &UserGroupService{db: db}
}

func (s *UserGroupService) List() ([]models.UserGroup, error) {
	var groups []models.UserGroup
	err := s.db.Preload("Members").Find(&groups).Error
	return groups, err
}

func (s *UserGroupService) Create(input CreateUserGroupInput) (*models.UserGroup, error) {
	group := models.UserGroup{
		GroupName:       input.GroupName,
		GroupCode:       input.GroupCode,
		Description:     input.Description,
		DataAccessScope: input.DataAccessScope,
		ParentGroupID:   input.ParentGroupID,
	}

	if err := s.db.Create(&group).Error; err != nil {
		return nil, err
	}

	return &group, nil
}

func (s *UserGroupService) Update(id string, input UpdateUserGroupInput) (*models.UserGroup, error) {
	var group models.UserGroup
	if err := s.db.First(&group, id).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"group_name":        input.GroupName,
		"description":       input.Description,
		"data_access_scope": input.DataAccessScope,
	}

	if err := s.db.Model(&group).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &group, nil
}

func (s *UserGroupService) Delete(id string) error {
	return s.db.Delete(&models.UserGroup{}, id).Error
}

func (s *UserGroupService) AddMembers(groupID string, userIDs []string) error {
	var group models.UserGroup
	if err := s.db.First(&group, groupID).Error; err != nil {
		return err
	}

	var users []models.User
	s.db.Find(&users, userIDs)

	if err := s.db.Model(&group).Association("Members").Append(&users); err != nil {
		return err
	}
	return nil
}

// ProviderService handles LLM provider operations
type ProviderService struct {
	db *gorm.DB
}

func NewProviderService(db *gorm.DB) *ProviderService {
	return &ProviderService{db: db}
}

func (s *ProviderService) List() ([]models.Provider, error) {
	var providers []models.Provider
	err := s.db.Where("enabled = ?", true).Find(&providers).Error
	return providers, err
}

func (s *ProviderService) Create(input CreateProviderInput) (*models.Provider, error) {
	provider := models.Provider{
		ProviderType: input.ProviderType,
		ProviderName: input.ProviderName,
		APIBase:      input.APIBase,
		APIKey:       input.APIKey,
		Model:        input.Model,
		Enabled:      input.Enabled,
		Priority:     input.Priority,
		Config:       input.Config,
	}

	if err := s.db.Create(&provider).Error; err != nil {
		return nil, err
	}

	return &provider, nil
}

func (s *ProviderService) Update(id string, input UpdateProviderInput) (*models.Provider, error) {
	var provider models.Provider
	if err := s.db.First(&provider, id).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"provider_name": input.ProviderName,
		"api_base":      input.APIBase,
		"api_key":       input.APIKey,
		"model":         input.Model,
		"enabled":       input.Enabled,
		"priority":      input.Priority,
		"config":        input.Config,
	}

	if err := s.db.Model(&provider).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &provider, nil
}

func (s *ProviderService) Delete(id string) error {
	return s.db.Delete(&models.Provider{}, id).Error
}

func (s *ProviderService) Test(id string) (map[string]interface{}, error) {
	var provider models.Provider
	if err := s.db.First(&provider, id).Error; err != nil {
		return nil, err
	}

	// TODO: Actually test the provider connection
	return map[string]interface{}{
		"success": true,
		"message": "Provider connection test not implemented",
	}, nil
}

// Placeholder implementations for other services
type DataSourceService struct {
	db        *gorm.DB
	llmClient llm.Client
}

func NewDataSourceService(db *gorm.DB, llmClient llm.Client) *DataSourceService {
	return &DataSourceService{db: db, llmClient: llmClient}
}

func (s *DataSourceService) List() ([]models.DataSource, error) {
	var ds []models.DataSource
	err := s.db.Where("enabled = ?", true).Find(&ds).Error
	return ds, err
}

func (s *DataSourceService) GetByID(id string) (*models.DataSource, error) {
	var ds models.DataSource
	if err := s.db.First(&ds, id).Error; err != nil {
		return nil, err
	}
	return &ds, nil
}

func (s *DataSourceService) Create(input CreateDataSourceInput) (*models.DataSource, error) {
	ds := models.DataSource{
		Name:       input.Name,
		SourceType: input.SourceType,
		Host:       input.Host,
		Port:       input.Port,
		User:       input.User,
		Password:   input.Password,
		Database:   input.Database,
		Connection: input.Connection,
		Enabled:    input.Enabled,
	}
	err := s.db.Create(&ds).Error
	return &ds, err
}

func (s *DataSourceService) Update(id string, input UpdateDataSourceInput) (*models.DataSource, error) {
	var ds models.DataSource
	if err := s.db.First(&ds, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{
		"name":       input.Name,
		"host":       input.Host,
		"port":       input.Port,
		"user":       input.User,
		"password":   input.Password,
		"database":   input.Database,
		"connection": input.Connection,
		"enabled":    input.Enabled,
	}
	err := s.db.Model(&ds).Updates(updates).Error
	return &ds, err
}

func (s *DataSourceService) Delete(id string) error {
	return s.db.Delete(&models.DataSource{}, id).Error
}

// Test 测试数据库连接
func (s *DataSourceService) Test(id string) (map[string]interface{}, error) {
	// 获取数据源
	ds, err := s.GetByID(id)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": "数据源不存在",
			"error":   "not_found",
		}, nil
	}

	// 构建数据库配置
	cfg := database.ExternalDBConfig{
		Host:     ds.Host,
		Port:     ds.Port,
		User:     ds.User,
		Password: ds.Password,
		Database: ds.Database,
		Type:     ds.SourceType,
	}

	// 测试连接
	return database.TestConnection(cfg)
}

// Analyze 分析数据库 Schema 并使用 AI 识别表关系
func (s *DataSourceService) Analyze(id string) (map[string]interface{}, error) {
	// 获取数据源
	ds, err := s.GetByID(id)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": "数据源不存在",
			"error":   "not_found",
		}, nil
	}

	// 构建数据库配置
	cfg := database.ExternalDBConfig{
		Host:     ds.Host,
		Port:     ds.Port,
		User:     ds.User,
		Password: ds.Password,
		Database: ds.Database,
		Type:     ds.SourceType,
	}

	// 1. 获取表结构
	tables, err := database.GetSchema(cfg)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("获取表结构失败: %v", err),
			"error":   "schema_failed",
		}, nil
	}

	// 2. 获取已有的外键关系
	existingRelations, _ := database.GetTableRelations(cfg)

	// 3. 生成 Schema 描述
	schemaText := database.GenerateSchemaJSON(tables)

	// 4. 如果有 LLM，使用 AI 分析表关系
	aiRelations := existingRelations
	if s.llmClient != nil {
		aiRelations, err = s.analyzeRelationsWithLLM(schemaText, tables)
		if err != nil {
			// 如果 AI 分析失败，使用数据库的外键关系
			aiRelations = existingRelations
		}
	}

	// 5. 构建返回结果
	result := map[string]interface{}{
		"success":            true,
		"message":            "分析完成",
		"tables":             tables,
		"relations":          aiRelations,
		"existing_relations": existingRelations,
	}

	// 6. 保存 Schema 信息到数据库
	schemaJSON, _ := json.Marshal(tables)
	relationsJSON, _ := json.Marshal(aiRelations)

	updates := map[string]interface{}{
		"schema_info":     string(schemaJSON),
		"table_relations": string(relationsJSON),
	}
	s.db.Model(ds).Updates(updates)

	return result, nil
}

// analyzeRelationsWithLLM 使用 LLM 分析表关系
func (s *DataSourceService) analyzeRelationsWithLLM(schemaText string, tables []database.TableInfo) ([]map[string]string, error) {
	// 构建 Prompt
	prompt := fmt.Sprintf(`我有一个数据库，包含以下表结构：

%s

请分析这些表之间的逻辑关系（不仅包括外键，还包括业务上的关联关系）。

请以 JSON 数组格式返回表关系，每个关系包含以下字段：
- from_table: 来源表
- from_column: 来源字段
- to_table: 目标表
- to_column: 目标字段
- relationship_type: 关系类型 (one_to_one, one_to_many, many_to_many)
- description: 关系描述

只返回 JSON 数组，不要其他内容。`, schemaText)

	// 调用 LLM
	resp, err := s.llmClient.ChatCompletion(context.Background(), llm.ChatRequest{
		Model: s.llmClient.GetModel(),
		Messages: []llm.Message{
			{Role: "system", Content: "你是一个数据库专家，擅长分析表结构和建议表关系。请以 JSON 格式返回分析结果。"},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   2000,
		Temperature: 0.2,
	})

	if err != nil {
		return nil, fmt.Errorf("LLM 调用失败: %v", err)
	}

	// 解析 JSON 响应
	var relations []map[string]string
	content := resp.Content
	// 清理可能的 markdown 代码块
	content = cleanJSONResponse(content)

	if err := json.Unmarshal([]byte(content), &relations); err != nil {
		return nil, fmt.Errorf("解析 LLM 响应失败: %v", err)
	}

	return relations, nil
}

// cleanJSONResponse 清理 JSON 响应，去除可能的 markdown 代码块
func cleanJSONResponse(content string) string {
	// 去除 ```json 和 ``` 标记
	content = removePrefix(content, "```json")
	content = removePrefix(content, "```")
	content = removePrefix(content, "`")
	content = removeSuffix(content, "```")
	content = removeSuffix(content, "`")
	return content
}

func removePrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func removeSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

// UpdateTableRelations 手动更新表关系
func (s *DataSourceService) UpdateTableRelations(id string, relations string) error {
	var ds models.DataSource
	if err := s.db.First(&ds, id).Error; err != nil {
		return err
	}
	return s.db.Model(&ds).Update("table_relations", relations).Error
}

// UpdateSchemaInfo 手动更新 Schema 信息
func (s *DataSourceService) UpdateSchemaInfo(id string, schemaInfo string) error {
	var ds models.DataSource
	if err := s.db.First(&ds, id).Error; err != nil {
		return err
	}
	return s.db.Model(&ds).Update("schema_info", schemaInfo).Error
}

type SkillService struct {
	db *gorm.DB
}

func NewSkillService(db *gorm.DB) *SkillService {
	return &SkillService{db: db}
}

func (s *SkillService) List() ([]models.Skill, error) {
	var skills []models.Skill
	err := s.db.Where("enabled = ?", true).Find(&skills).Error
	return skills, err
}

func (s *SkillService) Create(input CreateSkillInput) (*models.Skill, error) {
	skill := models.Skill{
		Name:            input.Name,
		SkillType:       input.SkillType,
		Description:     input.Description,
		Keywords:        input.Keywords,
		Category:        input.Category,
		Enabled:         input.Enabled,
		Config:          input.Config,
		PromptTemplate:  input.PromptTemplate,
		AllowedGroups:   input.AllowedGroups,
		DataScope:       input.DataScope,
		RequireApproval: input.RequireApproval,
	}
	err := s.db.Create(&skill).Error
	return &skill, err
}

func (s *SkillService) Update(id string, input UpdateSkillInput) (*models.Skill, error) {
	var skill models.Skill
	if err := s.db.First(&skill, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{
		"name": input.Name, "description": input.Description, "enabled": input.Enabled,
		"config": input.Config, "prompt_template": input.PromptTemplate,
		"allowed_groups": input.AllowedGroups, "data_scope": input.DataScope,
	}
	err := s.db.Model(&skill).Updates(updates).Error
	return &skill, err
}

func (s *SkillService) Delete(id string) error {
	return s.db.Delete(&models.Skill{}, id).Error
}

func (s *SkillService) Test(id, input string) (map[string]interface{}, error) {
	return map[string]interface{}{"success": true, "output": "Not implemented"}, nil
}

func (s *SkillService) SyncVector(id string) error {
	// 加载所有启用的 Skill
	var skills []models.Skill
	if err := s.db.Where("enabled = ?", true).Find(&skills).Error; err != nil {
		return fmt.Errorf("获取 Skill 列表失败: %w", err)
	}

	if len(skills) == 0 {
		return nil
	}

	// 构建语料库并创建 Embedder
	docs := make([]string, len(skills))
	for i, sk := range skills {
		docs[i] = sk.Name + " " + sk.Description + " " + sk.Keywords
	}

	embCfg := map[string]interface{}{"corpus": docs}
	emb, err := embedding.Create("tfidf", embCfg)
	if err != nil {
		return fmt.Errorf("创建 Embedder 失败: %w", err)
	}

	// 为每个 Skill 计算并存储向量
	for i, skill := range skills {
		v, err := emb.Embed(docs[i])
		if err != nil {
			return fmt.Errorf("向量化失败 [%s]: %w", skill.Name, err)
		}

		encoded := agent.EncodeVector(v)
		if err := s.db.Model(&skill).Updates(map[string]interface{}{
			"vector_index":   encoded,
			"vector_enabled": true,
		}).Error; err != nil {
			return fmt.Errorf("存储向量失败 [%s]: %w", skill.Name, err)
		}
	}

	return nil
}

type TaskService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewTaskService(db *gorm.DB, cfg *config.Config) *TaskService {
	return &TaskService{db: db, cfg: cfg}
}

func (s *TaskService) List() ([]models.ScheduledTask, error) {
	var tasks []models.ScheduledTask
	err := s.db.Find(&tasks).Error
	return tasks, err
}

func (s *TaskService) Create(input CreateTaskInput) (*models.ScheduledTask, error) {
	task := models.ScheduledTask{
		Name:           input.Name,
		Description:    input.Description,
		CronExpression: input.CronExpression,
		ExecutorType:   input.ExecutorType,
		ScriptContent:  input.ScriptContent,
		ScriptPath:     input.ScriptPath,
		Parameters:     input.Parameters,
		Enabled:        input.Enabled,
		Status:         "idle",
	}
	err := s.db.Create(&task).Error
	return &task, err
}

func (s *TaskService) Update(id string, input UpdateTaskInput) (*models.ScheduledTask, error) {
	var task models.ScheduledTask
	if err := s.db.First(&task, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{
		"name": input.Name, "description": input.Description, "cron_expression": input.CronExpression,
		"executor_type": input.ExecutorType, "script_content": input.ScriptContent,
		"parameters": input.Parameters, "enabled": input.Enabled,
	}
	err := s.db.Model(&task).Updates(updates).Error
	return &task, err
}

func (s *TaskService) Delete(id string) error {
	return s.db.Delete(&models.ScheduledTask{}, id).Error
}

func (s *TaskService) Run(id string) (map[string]interface{}, error) {
	return map[string]interface{}{"success": true, "message": "Task started"}, nil
}

func (s *TaskService) GetLogs(id string) ([]models.TaskExecution, error) {
	var logs []models.TaskExecution
	err := s.db.Where("task_id = ?", id).Order("created_at DESC").Find(&logs).Error
	return logs, err
}

type IMService struct {
	db *gorm.DB
}

// BindIMInput 用户绑定 IM 输入
// (CreateIMConfigInput, UpdateIMConfigInput 已在 service.go 定义)
type BindIMInput struct {
	ImConfigID string `json:"im_config_id"`
	ImUserID   string `json:"im_user_id"`
	ImUserName string `json:"im_user_name"`
}

// IMContext contains the parsed IM callback context
type IMContext struct {
	Platform string
	UserID   string
	UserName string
	Message  string
	ReplyTo  string // IM user ID to reply to
}

func NewIMService(db *gorm.DB) *IMService {
	return &IMService{db: db}
}

func (s *IMService) ListConfigs() ([]models.ImConfig, error) {
	var configs []models.ImConfig
	err := s.db.Where("enabled = ?", true).Find(&configs).Error
	return configs, err
}

func (s *IMService) CreateConfig(input CreateIMConfigInput) (*models.ImConfig, error) {
	cfg := models.ImConfig{
		Platform:    input.Platform,
		Name:        input.Name,
		AppID:       input.AppID,
		AppSecret:   input.AppSecret,
		Token:       input.Token,
		WebhookURL:  input.WebhookURL,
		CallbackURL: input.CallbackURL,
		Enabled:     input.Enabled,
		Config:      input.Config,
	}
	err := s.db.Create(&cfg).Error
	return &cfg, err
}

func (s *IMService) UpdateConfig(id string, input UpdateIMConfigInput) (*models.ImConfig, error) {
	var cfg models.ImConfig
	if err := s.db.First(&cfg, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{
		"name": input.Name, "app_id": input.AppID, "app_secret": input.AppSecret,
		"token": input.Token, "webhook_url": input.WebhookURL,
		"callback_url": input.CallbackURL, "enabled": input.Enabled, "config": input.Config,
	}
	err := s.db.Model(&cfg).Updates(updates).Error
	return &cfg, err
}

func (s *IMService) DeleteConfig(id string) error {
	return s.db.Delete(&models.ImConfig{}, id).Error
}

func (s *IMService) TestConfig(id string) (map[string]interface{}, error) {
	var cfg models.ImConfig
	if err := s.db.First(&cfg, id).Error; err != nil {
		return nil, err
	}
	// 尝试创建 platform handler 来验证配置
	_, err := im.NewPlatformHandler(&cfg)
	if err != nil {
		return map[string]interface{}{"success": false, "message": err.Error()}, nil
	}
	return map[string]interface{}{"success": true, "message": "配置有效"}, nil
}

// BindUser 用户绑定 IM 账号（员工个人操作）
func (s *IMService) BindUser(userID string, input BindIMInput) (*models.ImBinding, error) {
	// 检查是否已绑定同一平台
	var existing models.ImBinding
	err := s.db.Where("user_id = ? AND im_config_id = ?", userID, input.ImConfigID).First(&existing).Error
	if err == nil {
		return nil, fmt.Errorf("已绑定该平台账号: %s", existing.ImUserName)
	}

	binding := models.ImBinding{
		ImConfigID:   input.ImConfigID,
		UserID:       userID,
		ImUserID:     input.ImUserID,
		ImUserName:   input.ImUserName,
		BindingToken: uuid.New().String(),
		Status:       "active",
	}
	if err := s.db.Create(&binding).Error; err != nil {
		return nil, err
	}
	return &binding, nil
}

// ListUserBindings 列出当前用户的 IM 绑定（员工查看自己的绑定）
func (s *IMService) ListUserBindings(userID string) ([]models.ImBinding, error) {
	var bindings []models.ImBinding
	err := s.db.Preload("ImConfig").Where("user_id = ?", userID).Find(&bindings).Error
	return bindings, err
}

// ListPairings 管理员查看所有绑定
func (s *IMService) ListPairings() ([]models.ImBinding, error) {
	var bindings []models.ImBinding
	err := s.db.Preload("User").Preload("ImConfig").Find(&bindings).Error
	return bindings, err
}

// Unbind 解绑 IM 账号
func (s *IMService) Unbind(id string) error {
	return s.db.Delete(&models.ImBinding{}, id).Error
}

// HandleCallback 处理 IM 回调：解析消息 → 识别用户 → 返回上下文
func (s *IMService) HandleCallback(platform string, body []byte) (*IMContext, error) {
	// 查找对应平台的 IM 配置
	var cfg models.ImConfig
	if err := s.db.Where("platform = ? AND enabled = ?", platform, true).First(&cfg).Error; err != nil {
		return nil, fmt.Errorf("未找到启用的 IM 配置: %s", platform)
	}

	// 创建平台 handler 解析消息
	handler, err := im.NewPlatformHandler(&cfg)
	if err != nil {
		return nil, fmt.Errorf("创建处理器失败: %w", err)
	}

	msg, err := handler.ParseCallback(body)
	if err != nil {
		return nil, fmt.Errorf("解析消息失败: %w", err)
	}

	// 根据 IM 用户 ID 查找绑定的系统用户
	var binding models.ImBinding
	if err := s.db.Where("im_config_id = ? AND im_user_id = ? AND status = ?",
		cfg.ID, msg.FromUserID, "active").Preload("User").First(&binding).Error; err != nil {
		return nil, fmt.Errorf("未找到 IM 用户绑定: %s", msg.FromUserID)
	}

	// 检查用户是否激活
	if binding.User != nil && binding.User.Status != "active" {
		return nil, fmt.Errorf("用户账户已被禁用")
	}

	return &IMContext{
		Platform: platform,
		UserID:   binding.UserID,
		UserName: binding.User.Name,
		Message:  msg.Content,
		ReplyTo:  msg.FromUserID,
	}, nil
}

// SendIMReply 向 IM 用户发送回复
func (s *IMService) SendIMReply(platform string, imUserID string, content string) error {
	var cfg models.ImConfig
	if err := s.db.Where("platform = ? AND enabled = ?", platform, true).First(&cfg).Error; err != nil {
		return fmt.Errorf("未找到 IM 配置: %s", platform)
	}

	handler, err := im.NewPlatformHandler(&cfg)
	if err != nil {
		return err
	}

	return handler.SendMessage(imUserID, content)
}

type LogService struct {
	db *gorm.DB
}

func NewLogService(db *gorm.DB) *LogService {
	return &LogService{db: db}
}

func (s *LogService) ListAudit(page, limit string) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	err := s.db.Order("timestamp DESC").Limit(100).Find(&logs).Error
	return logs, err
}

func (s *LogService) ListRequest(page, limit string) ([]models.AuditLog, error) {
	return s.ListAudit(page, limit)
}

func (s *LogService) Export(logType, format string) ([]byte, error) {
	return []byte("[]"), nil
}

// Audit creates an audit log entry for an operation
func (s *LogService) Audit(userID, userName, action, resourceType, resourceID, details, ipAddress, userAgent string) error {
	logEntry := models.AuditLog{
		UserID:       userID,
		UserName:     userName,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       "success",
	}
	return s.db.Create(&logEntry).Error
}

// AuditWithStatus creates an audit log entry with status (success/failure)
func (s *LogService) AuditWithStatus(userID, userName, action, resourceType, resourceID, details, ipAddress, userAgent, status, errorMsg string) error {
	detailsJSON := details
	if errorMsg != "" {
		detailsJSON = details + " | Error: " + errorMsg
	}
	logEntry := models.AuditLog{
		UserID:       userID,
		UserName:     userName,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      detailsJSON,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       status,
	}
	return s.db.Create(&logEntry).Error
}

type AgentService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewAgentService(db *gorm.DB, cfg *config.Config) *AgentService {
	return &AgentService{db: db, cfg: cfg}
}

func (s *AgentService) Chat(userID, message, conversationID string) (map[string]interface{}, error) {
	// 创建或获取会话
	conv, err := s.getOrCreateConversation(userID, conversationID)
	if err != nil {
		return nil, fmt.Errorf("会话管理失败: %w", err)
	}

	// 存储用户消息
	userMsg := models.Message{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        message,
		Metadata:       "{}",
	}
	if createErr := s.db.Create(&userMsg).Error; createErr != nil {
		return nil, fmt.Errorf("保存消息失败: %w", createErr)
	}

	// 初始化 AgentEngine 并执行 Agentic Loop
	engine := agent.NewAgentEngine(s.db, s.cfg)

	ctx, cancel := context.WithTimeout(context.Background(), agent.LoopTimeout)
	defer cancel()

	// 获取用户上下文
	var user models.User
	if err := s.db.Preload("Groups").First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	agentUser := &agent.UserContext{
		UserID:       user.ID,
		GlobalUserID: user.GlobalUserID,
		Name:         user.Name,
		Department:   user.Department,
		Status:       user.Status,
	}

	for _, g := range user.Groups {
		agentUser.Groups = append(agentUser.Groups, agent.GroupContext{
			ID:              g.ID,
			Name:            g.GroupName,
			Code:            g.GroupCode,
			DataAccessScope: g.DataAccessScope,
		})
	}

	// 获取可用 Skills
	var skills []models.Skill
	s.db.Where("enabled = ?", true).Find(&skills)
	for _, sk := range skills {
		agentUser.AvailableSkills = append(agentUser.AvailableSkills, "skill_"+sk.SkillType)
	}

	// 执行 Agentic Loop
	resp, loopErr := engine.ExecuteLoop(ctx, agentUser, message, conv.ID)

	// 存储 AI 回复
	aiMsg := models.Message{
		ConversationID: conv.ID,
		Role:           "assistant",
		Content:        resp.Message,
		TokenCount:     resp.TokensUsed,
		Metadata:       fmt.Sprintf(`{"skill_used":"%s","iterations":%v}`, resp.SkillUsed, resp.Data["iterations"]),
	}
	s.db.Create(&aiMsg)

	// 更新会话标题
	if conv.Title == "" || conv.Title == "新对话" {
		title := message
		if len([]rune(title)) > 30 {
			title = string([]rune(title)[:30]) + "..."
		}
		s.db.Model(&conv).Update("title", title)
	}

	if loopErr != nil {
		return map[string]interface{}{
			"message":         resp.Message,
			"conversation_id": conv.ID,
			"skill_used":      resp.SkillUsed,
		}, loopErr
	}

	return map[string]interface{}{
		"message":         resp.Message,
		"conversation_id": conv.ID,
		"skill_used":      resp.SkillUsed,
		"tokens":          resp.TokensUsed,
		"data":            resp.Data,
	}, nil
}

// ChatStream sends a streaming chat response
// Returns a channel that yields response chunks
func (s *AgentService) ChatStream(userID, message, conversationID string) (<-chan string, error) {
	// 创建或获取会话
	conv, err := s.getOrCreateConversation(userID, conversationID)
	if err != nil {
		return nil, fmt.Errorf("会话管理失败: %w", err)
	}

	// 存储用户消息
	userMsg := models.Message{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        message,
		Metadata:       "{}",
	}
	s.db.Create(&userMsg)

	// 获取活跃的 LLM 提供商
	var provider models.Provider
	if err := s.db.Where("enabled = ?", true).Order("priority ASC").First(&provider).Error; err != nil {
		return nil, fmt.Errorf("no active provider: %w", err)
	}

	// Get the llm package
	llmClient, err := llm.NewClient(&provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// 构建消息历史
	history, _ := s.getConversationHistory(conv.ID, 20)
	llmMessages := s.buildLLMMessages(history, message)

	// Create a channel for streaming responses
	ch := make(chan string, 20)

	// Collect full response for saving
	var fullResponse strings.Builder

	// Start streaming in a goroutine
	go func() {
		defer close(ch)

		resp, err := llmClient.ChatCompletionStream(nil, llm.ChatRequest{
			Model:       provider.Model,
			Messages:    llmMessages,
			MaxTokens:   4096,
			Temperature: 0.7,
		})

		if err != nil {
			ch <- fmt.Sprintf("error: %v", err)
			return
		}

		// Stream chunks from the response
		for chunk := range resp.Chunks {
			fullResponse.WriteString(chunk)
			ch <- chunk
		}

		if err := <-resp.Err; err != nil {
			ch <- fmt.Sprintf("error: %v", err)
			return
		}

		// 存储 AI 回复
		aiMsg := models.Message{
			ConversationID: conv.ID,
			Role:           "assistant",
			Content:        fullResponse.String(),
			Metadata:       fmt.Sprintf(`{"model":"%s"}`, provider.Model),
		}
		s.db.Create(&aiMsg)

		// 更新会话标题
		if conv.Title == "" || conv.Title == "新对话" {
			title := message
			if len([]rune(title)) > 30 {
				title = string([]rune(title)[:30]) + "..."
			}
			s.db.Model(&conv).Update("title", title)
		}
	}()

	return ch, nil
}

// getOrCreateConversation 获取或创建会话
func (s *AgentService) getOrCreateConversation(userID, conversationID string) (*models.Conversation, error) {
	if conversationID != "" {
		var conv models.Conversation
		if err := s.db.Where("id = ? AND user_id = ?", conversationID, userID).First(&conv).Error; err == nil {
			return &conv, nil
		}
	}

	// 创建新会话
	conv := models.Conversation{
		UserID: userID,
		Title:  "新对话",
		Source: "web",
	}
	if err := s.db.Create(&conv).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

// getConversationHistory 获取最近 N 条消息历史
func (s *AgentService) getConversationHistory(conversationID string, limit int) ([]models.Message, error) {
	var messages []models.Message
	err := s.db.Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Limit(limit).
		Find(&messages).Error
	return messages, err
}

// buildLLMMessages 构建 LLM 消息列表（含历史 + 当前消息）
func (s *AgentService) buildLLMMessages(history []models.Message, currentMessage string) []llm.Message {
	// 系统提示词
	messages := []llm.Message{
		{
			Role:    "system",
			Content: "你是 OpenTether AI 助手，一个企业级 AI Agent。请用中文回答用户问题，保持专业、准确、简洁。你可以帮助用户查询数据、生成报表、执行任务。",
		},
	}

	// 注入历史消息（最多 20 条）
	startIdx := 0
	if len(history) > 20 {
		startIdx = len(history) - 20
	}
	for _, msg := range history[startIdx:] {
		role := msg.Role
		if role != "user" && role != "assistant" && role != "system" {
			role = "user"
		}
		messages = append(messages, llm.Message{
			Role:    role,
			Content: msg.Content,
		})
	}

	return messages
}

type ConversationService struct {
	db *gorm.DB
}

func NewConversationService(db *gorm.DB) *ConversationService {
	return &ConversationService{db: db}
}

func (s *ConversationService) List(userID string) ([]models.Conversation, error) {
	var convs []models.Conversation
	err := s.db.Where("user_id = ?", userID).Preload("Messages").Find(&convs).Error
	return convs, err
}

func (s *ConversationService) Get(id string) (*models.Conversation, error) {
	var conv models.Conversation
	err := s.db.Preload("Messages").First(&conv, id).Error
	return &conv, err
}
