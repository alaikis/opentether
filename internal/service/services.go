package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/agent"
	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/database"
	"github.com/alaikis/opentether/internal/embedding"
	"github.com/alaikis/opentether/internal/im"
	"github.com/alaikis/opentether/internal/llm"
	"github.com/alaikis/opentether/internal/models"
	"github.com/alaikis/opentether/internal/storage"
	"github.com/alaikis/opentether/internal/templating"
	"github.com/alaikis/opentether/internal/text2sql"
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

func (s *AuthService) Login(username, password string) (string, *models.User, error) {
	if s.db == nil {
		return "", nil, errors.New("database not initialized, please complete setup first")
	}

	var user models.User

	// Try to find by email or global_user_id
	if err := s.db.Where("email = ? OR global_user_id = ?", username, username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("invalid credentials")
		}
		return "", nil, err
	}

	if user.Status != "active" {
		return "", nil, errors.New("user account is not active")
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	// 设置默认角色
	if user.Role == "" {
		if user.CreatedBy == "system" || user.CreatedBy == "" {
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
		return "", nil, err
	}

	return token, &user, nil
}

func (s *AuthService) RefreshToken(tokenString string) (string, *models.User, error) {
	if s.db == nil {
		return "", nil, errors.New("database not initialized, please complete setup first")
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.Security.JWT.Secret), nil
	})

	if err != nil || !token.Valid {
		return "", nil, errors.New("invalid token")
	}

	claims := token.Claims.(jwt.MapClaims)
	userID := claims["user_id"].(string)

	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return "", nil, errors.New("user not found")
	}

	if user.Status != "active" {
		return "", nil, errors.New("user account is not active")
	}

	// 设置默认角色
	if user.Role == "" {
		if user.CreatedBy == "system" || user.CreatedBy == "" {
			user.Role = models.RoleAdmin
		} else {
			user.Role = models.RoleUser
		}
	}

	newToken, err := s.generateToken(user.ID, user.GlobalUserID, user.Name, user.Role)
	return newToken, &user, err
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
	err := s.db.Preload("Groups").Order("created_at DESC").Find(&users).Error
	return users, err
}

func (s *UserService) Create(input CreateUserInput) (*models.User, error) {
	// Hash password (use input password or default)
	pwd := input.Password
	if pwd == "" {
		pwd = "123456"
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := models.User{
		CompanyID:          input.CompanyID,
		GlobalUserID:       input.GlobalUserID,
		ExternalEmployeeID: input.ExternalEmployeeID,
		Name:               input.Name,
		Email:              input.Email,
		Department:         input.Department,
		Position:           input.Position,
		Role:               input.Role,
		SSOID:              input.SSOID,
		Status:             "active",
		PasswordHash:       string(passwordHash),
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
	if err := s.db.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"company_id":           input.CompanyID,
		"external_employee_id": input.ExternalEmployeeID,
		"name":                 input.Name,
		"email":                input.Email,
		"department":           input.Department,
		"position":             input.Position,
		"role":                 input.Role,
		"status":               input.Status,
	}

	if input.Password != "" {
		pwHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		updates["password_hash"] = string(pwHash)
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
	err := s.db.Preload("Members").Order("created_at DESC").Find(&groups).Error
	return groups, err
}

func (s *UserGroupService) Create(input CreateUserGroupInput) (*models.UserGroup, error) {
	group := models.UserGroup{
		CompanyID:       input.CompanyID,
		ExternalGroupID: input.ExternalGroupID,
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
	if err := s.db.Where("id = ?", id).First(&group).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"company_id":        input.CompanyID,
		"external_group_id": input.ExternalGroupID,
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
	if err := s.db.Where("id = ?", groupID).First(&group).Error; err != nil {
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
	err := s.db.Where("enabled = ?", true).Order("created_at DESC").Find(&providers).Error
	return providers, err
}

func (s *ProviderService) GetFirstEnabled() (*models.Provider, error) {
	var p models.Provider
	err := s.db.Where("enabled = ?", true).Order("priority ASC").First(&p).Error
	return &p, err
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
	if err := s.db.Where("id = ?", id).First(&provider).Error; err != nil {
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
	if err := s.db.Where("id = ?", id).First(&provider).Error; err != nil {
		return nil, err
	}

	// 发送一个简单的 ping 请求测试 provider 连通性
	req := llm.ChatRequest{
		Model:       provider.Model,
		Messages:    []llm.Message{{Role: "user", Content: "ping"}},
		MaxTokens:   5,
		Temperature: 0,
	}

	resp, err := llm.ChatWithProvider(&provider, req)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Provider 连接测试失败: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Provider %s (%s) 连接测试成功", provider.ProviderName, provider.Model),
		"model":   resp.Model,
		"usage":   resp.Usage,
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
	err := s.db.Where("enabled = ?", true).Order("created_at DESC").Find(&ds).Error
	return ds, err
}

func (s *DataSourceService) GetByID(id string) (*models.DataSource, error) {
	var ds models.DataSource
	if err := s.db.Where("id = ?", id).First(&ds).Error; err != nil {
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
	if err := s.db.Where("id = ?", id).First(&ds).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{
		"name":       input.Name,
		"host":       input.Host,
		"port":       input.Port,
		"user":       input.User,
		"database":   input.Database,
		"connection": input.Connection,
		"enabled":    input.Enabled,
	}
	// 密码不为空时才更新密码
	if input.Password != "" {
		updates["password"] = input.Password
	}
	if input.TableRelations != "" {
		updates["table_relations"] = input.TableRelations
	}
	if input.SchemaInfo != "" {
		updates["schema_info"] = input.SchemaInfo
	}
	err := s.db.Model(&ds).Updates(updates).Error
	return &ds, err
}

func (s *DataSourceService) Delete(id string) error {
	return s.db.Delete(&models.DataSource{}, id).Error
}

// TestConnection 直接测试数据库连接（使用原始凭据，不需要已有数据源）
func (s *DataSourceService) TestConnection(sourceType, host string, port int, user, password, dbName string) (map[string]interface{}, error) {
	cfg := database.ExternalDBConfig{
		Type:     sourceType,
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: dbName,
	}
	return database.TestConnection(cfg)
}

// Test 测试数据库连接（通过已存数据源 ID）
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
	if err := s.db.Where("id = ?", id).First(&ds).Error; err != nil {
		return err
	}
	return s.db.Model(&ds).Update("table_relations", relations).Error
}

// UpdateSchemaInfo 手动更新 Schema 信息
func (s *DataSourceService) UpdateSchemaInfo(id string, schemaInfo string) error {
	var ds models.DataSource
	if err := s.db.Where("id = ?", id).First(&ds).Error; err != nil {
		return err
	}
	return s.db.Model(&ds).Update("schema_info", schemaInfo).Error
}

type SkillService struct {
	db      *gorm.DB
	store   storage.Driver
	devMode bool
}

var ErrBuiltinSkillProtected = errors.New("system built-in skills cannot be modified")

func NewSkillService(db *gorm.DB, devMode bool, stores ...storage.Driver) *SkillService {
	var store storage.Driver
	if len(stores) > 0 {
		store = stores[0]
	}
	return &SkillService{db: db, store: store, devMode: devMode}
}

func isBuiltinSkill(skill models.Skill) bool {
	if skill.Category == "系统内置" {
		return true
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(skill.Config), &config); err != nil {
		return false
	}
	builtin, ok := config["builtin"].(bool)
	return ok && builtin
}

func (s *SkillService) List() ([]models.Skill, error) {
	var skills []models.Skill
	err := s.db.Where("enabled = ?", true).Order("created_at DESC").Find(&skills).Error
	return skills, err
}

func (s *SkillService) ValidateContextMDFiles() {
	if s == nil || s.db == nil || s.store == nil {
		return
	}
	var skills []models.Skill
	if err := s.db.Where("config LIKE ?", "%context_md_path%").Find(&skills).Error; err != nil {
		return
	}
	for _, skill := range skills {
		var cfg map[string]interface{}
		if json.Unmarshal([]byte(skill.Config), &cfg) != nil {
			continue
		}
		path, _ := cfg["context_md_path"].(string)
		if path == "" || s.store.Exists(context.Background(), path) {
			continue
		}
		if restored := s.restoreContextMDFromVersions(cfg); restored {
			b, _ := json.Marshal(cfg)
			_ = s.db.Model(&skill).Update("config", string(b)).Error
			continue
		}
		cfg["context_md_missing"] = true
		cfg["context_md_missing_at"] = time.Now()
		b, _ := json.Marshal(cfg)
		_ = s.db.Model(&skill).Update("config", string(b)).Error
	}
}

func (s *SkillService) restoreContextMDFromVersions(cfg map[string]interface{}) bool {
	versions, _ := cfg["context_md_versions"].([]interface{})
	for i := len(versions) - 1; i >= 0; i-- {
		entry, ok := versions[i].(map[string]interface{})
		if !ok {
			continue
		}
		path, _ := entry["path"].(string)
		url, _ := entry["url"].(string)
		if path != "" && s.store.Exists(context.Background(), path) {
			cfg["context_md_path"] = path
			cfg["context_md_url"] = url
			cfg["context_md_version"] = entry["version"]
			cfg["context_md_size"] = entry["size"]
			delete(cfg, "context_md_missing")
			delete(cfg, "context_md_missing_at")
			return true
		}
	}
	return false
}

func (s *SkillService) GetContextMD(id string) (map[string]interface{}, error) {
	var skill models.Skill
	if err := s.db.Where("id = ?", id).First(&skill).Error; err != nil {
		return nil, err
	}
	var cfg map[string]interface{}
	_ = json.Unmarshal([]byte(skill.Config), &cfg)
	content := ""
	if inline, ok := cfg["context_md"].(string); ok {
		content = inline
	} else if url, ok := cfg["context_md_url"].(string); ok && url != "" {
		content = fetchTextFromURL(url, 1024*1024)
	}
	return map[string]interface{}{
		"skill":              skill,
		"content":            content,
		"context_md_url":     cfg["context_md_url"],
		"context_md_path":    cfg["context_md_path"],
		"context_md_size":    cfg["context_md_size"],
		"context_md_version": cfg["context_md_version"],
		"versions":           cfg["context_md_versions"],
	}, nil
}

func (s *SkillService) UpdateContextMD(id, content string, publish bool) (map[string]interface{}, error) {
	var skill models.Skill
	if err := s.db.Where("id = ?", id).First(&skill).Error; err != nil {
		return nil, err
	}
	if isBuiltinSkill(skill) && !s.devMode {
		return nil, ErrBuiltinSkillProtected
	}
	if s.store == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	var cfg map[string]interface{}
	_ = json.Unmarshal([]byte(skill.Config), &cfg)
	if cfg == nil {
		cfg = map[string]interface{}{}
	}
	version := time.Now().UTC().Format("20060102150405")
	objectPath := filepath.ToSlash(filepath.Join("skills", "context", id, "v"+version+".md"))
	url, err := s.store.Save(context.Background(), objectPath, []byte(content), "text/markdown; charset=utf-8")
	if err != nil {
		return nil, err
	}
	entry := map[string]interface{}{"version": version, "path": objectPath, "url": url, "size": len([]byte(content)), "created_at": time.Now()}
	versions, _ := cfg["context_md_versions"].([]interface{})
	versions = append(versions, entry)
	if len(versions) > 20 {
		versions = versions[len(versions)-20:]
	}
	cfg["context_md_versions"] = versions
	cfg["draft_context_md_url"] = url
	cfg["draft_context_md_path"] = objectPath
	if publish {
		cfg["context_md_url"] = url
		cfg["context_md_path"] = objectPath
		cfg["context_md_size"] = len([]byte(content))
		cfg["context_md_version"] = version
	}
	delete(cfg, "context_md")
	b, _ := json.Marshal(cfg)
	if err := s.db.Model(&skill).Update("config", string(b)).Error; err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "url": url, "path": objectPath, "version": version, "published": publish}, nil
}

func fetchTextFromURL(url string, max int64) string {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, max))
	if err != nil {
		return ""
	}
	return string(b)
}

const defaultText2SQLContextJinja = `# {{ config.data_source_name|default:"Text2SQL" }} 数据上下文

## 可用表边界
{% for table in config.selected_tables %}
### {{ table.name }}
{{ table.description }}

字段：
{% for col in table.columns %}
- {{ col.name }}{% if col.type %} ({{ col.type }}){% endif %}{% if col.description %}：{{ col.description }}{% endif %}
{% endfor %}
{% endfor %}

## 表关系
{% for rel in config.table_relations %}
- {{ rel.from_table }}.{{ rel.from_column }} -> {{ rel.to_table }}.{{ rel.to_column }}{% if rel.description %}：{{ rel.description }}{% endif %}
{% endfor %}

## 业务口径
{{ config.business_rules }}
`

func (s *SkillService) persistSkillContextMD(skillID, config string) (string, bool) {
	if s == nil || s.store == nil || config == "" || skillID == "" {
		return config, false
	}
	var cfg map[string]interface{}
	if json.Unmarshal([]byte(config), &cfg) != nil {
		return config, false
	}
	md, _ := cfg["context_md"].(string) // backward compatibility for older clients
	tpl, _ := cfg["context_template"].(string)
	if strings.TrimSpace(tpl) == "" && hasText2SQLContextConfig(cfg) {
		tpl = defaultText2SQLContextJinja
		cfg["context_template"] = tpl
		cfg["template_engine"] = "jinja2"
	}
	if strings.TrimSpace(tpl) != "" {
		md = templating.SafeRender(tpl, map[string]interface{}{"skill_id": skillID, "config": cfg}, md)
	}
	if strings.TrimSpace(md) == "" {
		return config, false
	}
	version := time.Now().UTC().Format("20060102150405")
	objectPath := filepath.ToSlash(filepath.Join("skills", "context", skillID, "v"+version+".md"))
	url, err := s.store.Save(context.Background(), objectPath, []byte(md), "text/markdown; charset=utf-8")
	if err != nil {
		return config, false
	}
	cfg["context_md_path"] = objectPath
	cfg["context_md_url"] = url
	cfg["context_md_size"] = len([]byte(md))
	cfg["context_md_version"] = version
	versions, _ := cfg["context_md_versions"].([]interface{})
	versions = append(versions, map[string]interface{}{"version": version, "path": objectPath, "url": url, "size": len([]byte(md)), "created_at": time.Now()})
	if len(versions) > 20 {
		versions = versions[len(versions)-20:]
	}
	cfg["context_md_versions"] = versions
	delete(cfg, "context_md")
	b, err := json.Marshal(cfg)
	if err != nil {
		return config, false
	}
	return string(b), true
}

func hasText2SQLContextConfig(cfg map[string]interface{}) bool {
	if cfg == nil {
		return false
	}
	if tables, ok := cfg["selected_tables"].([]interface{}); ok && len(tables) > 0 {
		return true
	}
	if rules, ok := cfg["business_rules"].(string); ok && strings.TrimSpace(rules) != "" {
		return true
	}
	if rels, ok := cfg["table_relations"].([]interface{}); ok && len(rels) > 0 {
		return true
	}
	return false
}

func (s *SkillService) Create(input CreateSkillInput) (*models.Skill, error) {
	keywordsJSON, _ := json.Marshal(input.Keywords)
	skill := models.Skill{
		Name:            input.Name,
		SkillType:       input.SkillType,
		Description:     input.Description,
		Keywords:        string(keywordsJSON),
		Category:        input.Category,
		Enabled:         input.Enabled,
		Config:          input.Config,
		PromptTemplate:  input.PromptTemplate,
		AllowedGroups:   input.AllowedGroups,
		DataScope:       input.DataScope,
		RequireApproval: input.RequireApproval,
	}
	if err := s.db.Create(&skill).Error; err != nil {
		return &skill, err
	}
	if config, changed := s.persistSkillContextMD(skill.ID, skill.Config); changed {
		skill.Config = config
		_ = s.db.Model(&skill).Update("config", config).Error
	}
	return &skill, nil
}

func (s *SkillService) Update(id string, input UpdateSkillInput) (*models.Skill, error) {
	var skill models.Skill
	if err := s.db.Where("id = ?", id).First(&skill).Error; err != nil {
		return nil, err
	}
	if isBuiltinSkill(skill) && !s.devMode {
		return nil, ErrBuiltinSkillProtected
	}
	keywordsJSON, _ := json.Marshal(input.Keywords)
	updates := map[string]interface{}{
		"name": input.Name, "skill_type": input.SkillType, "description": input.Description,
		"keywords": string(keywordsJSON), "category": input.Category, "enabled": input.Enabled,
		"config": input.Config, "prompt_template": input.PromptTemplate,
		"allowed_groups": input.AllowedGroups, "data_scope": input.DataScope,
	}
	if config, changed := s.persistSkillContextMD(id, input.Config); changed {
		updates["config"] = config
	}
	err := s.db.Model(&skill).Updates(updates).Error
	return &skill, err
}

func (s *SkillService) ListRouteExamples(status string, limit int) ([]models.RouteExample, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := s.db.Model(&models.RouteExample{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var examples []models.RouteExample
	err := query.Order("updated_at DESC").Limit(limit).Find(&examples).Error
	return examples, err
}

func (s *SkillService) CreateRouteExample(text, route, intent string, confidence float64) (*models.RouteExample, error) {
	if confidence <= 0 {
		confidence = 0.9
	}
	ex := models.RouteExample{Text: text, Route: route, Intent: intent, Source: "admin", Status: "active", Confidence: confidence}
	if err := s.db.Create(&ex).Error; err != nil {
		return nil, err
	}
	return &ex, nil
}

func (s *SkillService) ReviewRouteExample(id, action string) (*models.RouteExample, error) {
	var ex models.RouteExample
	if err := s.db.Where("id = ?", id).First(&ex).Error; err != nil {
		return nil, err
	}
	switch action {
	case "approve":
		ex.Status = "active"
		ex.Source = "admin"
		ex.Confidence = 1.0
	case "reject":
		ex.Status = "rejected"
		ex.Source = "rejected"
		ex.Confidence = 0
	default:
		return nil, fmt.Errorf("unsupported route example action: %s", action)
	}
	if err := s.db.Save(&ex).Error; err != nil {
		return nil, err
	}
	return &ex, nil
}

func (s *SkillService) ListRuntimeMemories(status string, limit int) ([]models.SkillRuntimeMemory, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := s.db.Model(&models.SkillRuntimeMemory{})
	if status == "pending" {
		query = query.Where("source = ? AND confidence < ?", "runtime", 0.75)
	} else if status == "approved" {
		query = query.Where("source IN ? OR confidence >= ?", []string{"admin", "confirmed"}, 0.75)
	} else if status == "rejected" {
		query = query.Where("source = ?", "rejected")
	}
	var memories []models.SkillRuntimeMemory
	err := query.Order("updated_at DESC").Limit(limit).Find(&memories).Error
	return memories, err
}

func (s *SkillService) ReviewRuntimeMemory(id, action, content string) (*models.SkillRuntimeMemory, error) {
	var mem models.SkillRuntimeMemory
	if err := s.db.Where("id = ?", id).First(&mem).Error; err != nil {
		return nil, err
	}
	switch action {
	case "approve":
		mem.Source = "admin"
		mem.Confidence = 1.0
		if strings.TrimSpace(content) != "" {
			mem.Content = content
		}
	case "reject":
		mem.Source = "rejected"
		mem.Confidence = 0
	case "edit":
		if strings.TrimSpace(content) != "" {
			mem.Content = content
		}
		mem.Source = "admin"
		mem.Confidence = 1.0
	default:
		return nil, fmt.Errorf("unsupported review action: %s", action)
	}
	mem.LastUsedAt = time.Now()
	if err := s.db.Save(&mem).Error; err != nil {
		return nil, err
	}
	return &mem, nil
}

func (s *SkillService) DeleteRuntimeMemory(id string) error {
	return s.db.Delete(&models.SkillRuntimeMemory{}, "id = ?", id).Error
}

func (s *SkillService) Delete(id string) error {
	var skill models.Skill
	if err := s.db.Where("id = ?", id).First(&skill).Error; err != nil {
		return err
	}
	// 开发模式下允许删除内置 skills（重启后自动重新 seed）
	if isBuiltinSkill(skill) && !s.devMode {
		return ErrBuiltinSkillProtected
	}
	return s.db.Delete(&skill).Error
}

func (s *SkillService) Test(id, input string) (map[string]interface{}, error) {
	var skill models.Skill
	if err := s.db.Where("id = ?", id).First(&skill).Error; err != nil {
		return nil, err
	}
	result := map[string]interface{}{
		"skill_id":   skill.ID,
		"skill_name": skill.Name,
		"skill_type": skill.SkillType,
	}
	if skill.SkillType != "text2sql" {
		result["success"] = true
		result["score"] = 100
		result["output"] = "Skill 基础配置可用。当前仅对 text2sql 类型提供深度诊断。"
		return result, nil
	}
	var cfg map[string]interface{}
	_ = json.Unmarshal([]byte(skill.Config), &cfg)
	issues := []string{}
	score := 100
	if strings.TrimSpace(fmt.Sprint(cfg["data_source_id"])) == "" {
		issues = append(issues, "缺少 data_source_id：未绑定数据源")
		score -= 20
	}
	if strings.TrimSpace(fmt.Sprint(cfg["entry_table"])) == "" {
		issues = append(issues, "缺少 entry_table：建议设置入口表/主事实表")
		score -= 15
	}
	if arr, ok := cfg["selected_tables"].([]interface{}); !ok || len(arr) == 0 {
		issues = append(issues, "缺少 selected_tables：未选择可用表")
		score -= 15
	}
	if arr, ok := cfg["table_relations"].([]interface{}); !ok || len(arr) == 0 {
		issues = append(issues, "缺少 table_relations：建议配置表关系")
		score -= 15
	}
	if arr, ok := cfg["metric_rules"].([]interface{}); !ok || len(arr) == 0 {
		issues = append(issues, "缺少 metric_rules：建议配置订单数/销售额等指标规则")
		score -= 15
	}
	if arr, ok := cfg["entity_rules"].([]interface{}); !ok || len(arr) == 0 {
		issues = append(issues, "缺少 entity_rules：建议配置员工/客户等实体规则")
		score -= 10
	}
	if _, ok := cfg["field_aliases"].(map[string]interface{}); !ok {
		issues = append(issues, "缺少 field_aliases：建议配置中文业务词到字段的别名映射")
		score -= 5
	}
	if score < 0 {
		score = 0
	}
	output := "Text2SQL Skill 诊断完成"
	if len(issues) > 0 {
		output += "，发现问题：\n- " + strings.Join(issues, "\n- ")
	} else {
		output += "，配置完整。"
	}
	result["success"] = score >= 70
	result["score"] = score
	result["issues"] = issues
	result["output"] = output
	return result, nil
}

func (s *SkillService) GenerateSkillWithAI(description, sourceType, tableNames string) (map[string]interface{}, error) {
	provider, err := NewProviderService(s.db).GetFirstEnabled()
	if err != nil || provider == nil {
		return nil, fmt.Errorf("未配置可用的 LLM Provider")
	}
	prompt := fmt.Sprintf("你是企业级 Skill 配置专家。根据描述生成 Text2SQL Skill 配置 JSON。\n\n需求: %s\n数据源类型: %s\n相关表: %s\n\n返回JSON格式（只返回JSON）", description, sourceType, tableNames)
	client, err := llm.NewClient(provider)
	if err != nil {
		return nil, fmt.Errorf("创建客户端失败: %v", err)
	}
	resp, err := client.ChatCompletion(context.Background(), llm.ChatRequest{
		Model:       provider.Model,
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   1000,
		Temperature: 0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("AI 生成失败: %v", err)
	}
	return map[string]interface{}{"content": resp.Content}, nil
}

func (s *SkillService) GenerateText2SQLRelations(dataSourceID string, tables []map[string]interface{}) (map[string]interface{}, error) {
	if dataSourceID == "" || len(tables) == 0 {
		return nil, fmt.Errorf("请提供数据源和已选表")
	}
	provider, err := NewProviderService(s.db).GetFirstEnabled()
	if err != nil || provider == nil {
		return nil, fmt.Errorf("未配置可用的 LLM Provider")
	}
	tablesJSON, _ := json.Marshal(tables)
	prompt := fmt.Sprintf(`你是一个数据库专家。请分析以下数据库表结构和字段，生成：
1. 表之间的逻辑关系（from_table.from_column -> to_table.to_column）2. 每个字段的业务含义建议3. 业务口径推荐

表结构：%s

请严格以JSON返回：
{"relations":[{"from_table":"","from_column":"","to_table":"","to_column":"","description":""}],"field_semantics":[{"table":"","column":"","description":""}],"business_rules":"建议的业务口径"}`, string(tablesJSON))
	client, err := llm.NewClient(provider)
	if err != nil {
		return nil, fmt.Errorf("创建客户端失败: %v", err)
	}
	resp, err := client.ChatCompletion(context.Background(), llm.ChatRequest{
		Model:       provider.Model,
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   2000,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("AI 分析失败: %v", err)
	}
	content := cleanJSONResponse(resp.Content)
	var result map[string]interface{}
	if json.Unmarshal([]byte(content), &result) != nil {
		return map[string]interface{}{"raw": resp.Content}, nil
	}
	return result, nil
}

// GenerateText2SQLRelationsStream 流式 AI 分析表关系（模拟 SSE 逐行输出）
func (s *SkillService) GenerateText2SQLRelationsStream(dataSourceID string, tables []map[string]interface{}) (<-chan string, error) {
	if dataSourceID == "" || len(tables) == 0 {
		return nil, fmt.Errorf("请提供数据源和已选表")
	}
	provider, err := NewProviderService(s.db).GetFirstEnabled()
	if err != nil || provider == nil {
		return nil, fmt.Errorf("未配置可用的 LLM Provider")
	}
	tablesJSON, _ := json.Marshal(tables)
	prompt := fmt.Sprintf("你是一个数据库专家。请分析以下数据库表结构和字段，生成表关系和字段含义。以JSON格式返回: {\"relations\":[{\"from_table\":\"\",\"from_column\":\"\",\"to_table\":\"\",\"to_column\":\"\",\"description\":\"\"}],\"business_rules\":\"\"}\n表结构: %s", string(tablesJSON))
	client, err := llm.NewClient(provider)
	if err != nil {
		return nil, fmt.Errorf("创建客户端失败: %v", err)
	}
	ch := make(chan string, 50)
	go func() {
		defer close(ch)
		ch <- `{"type":"status","message":"AI 正在分析表关系..."}`
		resp, err := client.ChatCompletion(context.Background(), llm.ChatRequest{
			Model:       provider.Model,
			Messages:    []llm.Message{{Role: "user", Content: prompt}},
			MaxTokens:   2000,
			Temperature: 0.2,
		})
		if err != nil {
			ch <- fmt.Sprintf(`{"type":"error","message":"%s"}`, err.Error())
			return
		}
		content := cleanJSONResponse(resp.Content)
		var result map[string]interface{}
		if json.Unmarshal([]byte(content), &result) == nil {
			if relations, ok := result["relations"].([]interface{}); ok {
				for i, rel := range relations {
					relJSON, _ := json.Marshal(map[string]interface{}{"type": "relation", "index": i, "data": rel})
					ch <- string(relJSON)
				}
			}
			if rules, ok := result["business_rules"].(string); ok && rules != "" {
				ch <- fmt.Sprintf(`{"type":"business_rules","data":"%s"}`, rules)
			}
			ch <- `{"type":"done","message":"分析完成"}`
		} else {
			ch <- `{"type":"raw","data":"` + strings.ReplaceAll(content, `"`, `\"`) + `"}`
		}
	}()
	return ch, nil
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
	err := s.db.Order("created_at DESC").Find(&tasks).Error
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
	err := s.db.Where("enabled = ?", true).Order("created_at DESC").Find(&configs).Error
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
	if err := s.db.Where("id = ?", id).First(&cfg).Error; err != nil {
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
	err := s.db.Preload("ImConfig").Where("user_id = ?", userID).Order("created_at DESC").Find(&bindings).Error
	return bindings, err
}

// ListPairings 管理员查看所有绑定
func (s *IMService) ListPairings() ([]models.ImBinding, error) {
	var bindings []models.ImBinding
	err := s.db.Preload("User").Preload("ImConfig").Order("created_at DESC").Find(&bindings).Error
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
	err := s.db.Where("action != ?", "request").Order("timestamp DESC").Limit(100).Find(&logs).Error
	return logs, err
}

func (s *LogService) ListRequest(page, limit string) ([]map[string]interface{}, error) {
	var rows []map[string]interface{}
	err := s.db.Model(&models.AuditLog{}).
		Select("id, timestamp, user_name, details, ip_address").
		Where("action = ?", "request").
		Order("timestamp DESC").
		Limit(100).
		Find(&rows).Error
	return rows, err
}

// WriteRequestLog 写入请求日志到 audit_log 表（action="request"，details 存 JSON）
func (s *LogService) WriteRequestLog(method, path string, status int, latencyMs float64, userID, userName, ip string) {
	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"method":     method,
		"path":       path,
		"status":     status,
		"latency_ms": latencyMs,
	})
	entry := models.AuditLog{
		UserID:    userID,
		UserName:  userName,
		Action:    "request",
		Details:   string(detailsJSON),
		IPAddress: ip,
		Status:    "success",
	}
	s.db.Create(&entry)
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

type ExperienceService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewExperienceService(db *gorm.DB, cfg *config.Config) *ExperienceService {
	return &ExperienceService{db: db, cfg: cfg}
}

func (s *ExperienceService) List(status, scope string) ([]models.AgentExperience, error) {
	query := s.db.Model(&models.AgentExperience{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if scope != "" {
		query = query.Where("scope = ?", scope)
	}
	var experiences []models.AgentExperience
	err := query.Order("created_at DESC").Find(&experiences).Error
	return experiences, err
}

func (s *ExperienceService) GetByID(id string) (*models.AgentExperience, error) {
	var exp models.AgentExperience
	if err := s.db.Where("id = ?", id).First(&exp).Error; err != nil {
		return nil, err
	}
	return &exp, nil
}

func (s *ExperienceService) Review(id, reviewerID, status, note string) error {
	mgr := agent.NewExperienceManager(s.db)
	return mgr.ReviewExperience(id, reviewerID, status, note)
}

func (s *ExperienceService) PromoteToGlobal(id string) error {
	mgr := agent.NewExperienceManager(s.db)
	return mgr.PromoteToGlobal(id)
}

func (s *ExperienceService) Delete(id string) error {
	mgr := agent.NewExperienceManager(s.db)
	return mgr.DeleteExperience(id)
}

type AgentService struct {
	db         *gorm.DB
	cfg        *config.Config
	store      storage.Driver
	sqlAuditor text2sql.AuditRecorder
	mcp        agent.MCPToolProvider
	engine     *agent.AgentEngine
}

func NewAgentService(db *gorm.DB, cfg *config.Config, store storage.Driver) *AgentService {
	return &AgentService{db: db, cfg: cfg, store: store}
}

func (s *AgentService) SetMCPProvider(provider agent.MCPToolProvider) {
	s.mcp = provider
	if s.engine != nil {
		s.engine.SetMCPProvider(provider)
	}
}

// Close 关闭 AgentService 持有的长生命周期资源。
func (s *AgentService) Close() error {
	if s == nil || s.engine == nil {
		return nil
	}
	return s.engine.Close()
}

// getEngine 懒初始化并返回复用的 AgentEngine
func (s *AgentService) getEngine() *agent.AgentEngine {
	if s.engine == nil {
		s.engine = agent.NewAgentEngine(s.db, s.cfg, s.store)
		if s.sqlAuditor != nil {
			s.engine.SetSQLAuditor(s.sqlAuditor)
		}
		if s.mcp != nil {
			s.engine.SetMCPProvider(s.mcp)
		}
	}
	return s.engine
}

// SetSQLAuditor 设置 SQL 审计器
func (s *AgentService) SetSQLAuditor(auditor text2sql.AuditRecorder) {
	s.sqlAuditor = auditor
}

func (s *AgentService) ListRuntimeJobs(userID, status string, limit int) ([]models.RuntimeJob, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := s.db.Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var jobs []models.RuntimeJob
	err := query.Order("updated_at DESC").Limit(limit).Find(&jobs).Error
	return jobs, err
}

func (s *AgentService) GetRuntimeJob(userID, jobID string) (*models.RuntimeJob, []models.RuntimeCheckpoint, error) {
	var job models.RuntimeJob
	if err := s.db.Where("id = ? AND user_id = ?", jobID, userID).First(&job).Error; err != nil {
		return nil, nil, err
	}
	var checkpoints []models.RuntimeCheckpoint
	s.db.Where("job_id = ?", jobID).Order("step ASC, created_at ASC").Find(&checkpoints)
	return &job, checkpoints, nil
}

func (s *AgentService) CancelRuntimeJob(userID, jobID string) error {
	return s.db.Model(&models.RuntimeJob{}).
		Where("id = ? AND user_id = ? AND status IN ?", jobID, userID, []string{"pending", "running", "paused", "recovering"}).
		Updates(map[string]interface{}{"status": "cancelled", "error": "用户取消", "finished_at": time.Now()}).Error
}

func (s *AgentService) AutoRecoverRuntimeJobs(limit int) {
	if limit <= 0 {
		limit = 10
	}
	var jobs []models.RuntimeJob
	if err := s.db.Where("status = ? AND recoverable = ?", "paused", true).
		Order("updated_at ASC").
		Limit(limit).
		Find(&jobs).Error; err != nil {
		return
	}
	for _, job := range jobs {
		if !isRuntimeJobSafeToAutoRetry(job) {
			continue
		}
		go func(j models.RuntimeJob) {
			_, _ = s.RetryRuntimeJob(j.UserID, j.ID)
		}(job)
	}
}

func isRuntimeJobSafeToAutoRetry(job models.RuntimeJob) bool {
	if job.JobType != "agent_loop" || job.Input == "" {
		return false
	}
	lower := strings.ToLower(job.Input)
	for _, kw := range []string{"删除", "修改", "更新", "创建", "新增", "写入", "提交", "审批", "delete", "update", "insert", "drop", "alter", "create", "truncate"} {
		if strings.Contains(lower, kw) {
			return false
		}
	}
	return true
}

func (s *AgentService) RetryRuntimeJob(userID, jobID string) (map[string]interface{}, error) {
	job, _, err := s.GetRuntimeJob(userID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != "paused" && job.Status != "failed" && job.Status != "recovering" {
		return nil, fmt.Errorf("当前任务状态 %s 不允许重试/恢复", job.Status)
	}
	var input struct {
		Query string `json:"query"`
	}
	_ = json.Unmarshal([]byte(job.Input), &input)
	if input.Query == "" {
		return nil, fmt.Errorf("任务缺少原始输入，无法恢复")
	}

	engine := s.getEngine()
	ctx, cancel := context.WithTimeout(context.Background(), agent.LoopTimeout)
	defer cancel()

	var user models.User
	if err := s.db.Preload("Groups").Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	agentUser := buildAgentUser(s.db, user, job.SkillID)
	agentUser.Context["runtime_job_id"] = job.ID

	resp, loopErr := engine.ExecuteLoop(ctx, agentUser, input.Query, job.ConversationID)
	aiMsg := models.Message{
		ConversationID: job.ConversationID,
		Role:           "assistant",
		Content:        resp.Message,
		TokenCount:     resp.TokensUsed,
		Metadata:       fmt.Sprintf(`{"skill_used":"%s","recovered_job_id":"%s"}`, resp.SkillUsed, job.ID),
	}
	s.db.Create(&aiMsg)
	_ = engine.UpdateConversationMemory(agentUser, job.ConversationID, input.Query, resp.Message)

	result := map[string]interface{}{
		"message":         resp.Message,
		"conversation_id": job.ConversationID,
		"skill_used":      resp.SkillUsed,
		"tokens":          resp.TokensUsed,
		"data":            resp.Data,
		"runtime_job_id":  job.ID,
	}
	return result, loopErr
}

func (s *AgentService) Chat(userID, message, conversationID, skillID string) (map[string]interface{}, error) {
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
	engine := s.getEngine()

	ctx, cancel := context.WithTimeout(context.Background(), agent.LoopTimeout)
	defer cancel()

	// 获取用户上下文
	var user models.User
	if err := s.db.Preload("Groups").Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	agentUser := buildAgentUser(s.db, user, skillID)

	if fastResp, ok, fastErr := engine.TryFastPath(ctx, agentUser, message, conv.ID); fastErr == nil && ok {
		engine.LearnRouteExampleCandidate(message, fastResp.SkillUsedToRoute(), "", 0.7)
		aiMsg := models.Message{ConversationID: conv.ID, Role: "assistant", Content: fastResp.Message, TokenCount: fastResp.TokensUsed, Metadata: fmt.Sprintf(`{"skill_used":"%s","fast_path":true}`, fastResp.SkillUsed)}
		s.db.Create(&aiMsg)
		_ = engine.UpdateConversationMemory(agentUser, conv.ID, message, fastResp.Message)
		return map[string]interface{}{"message": fastResp.Message, "conversation_id": conv.ID, "skill_used": fastResp.SkillUsed, "tokens": fastResp.TokensUsed, "data": fastResp.Data, "fast_path": true}, nil
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
	if memErr := engine.UpdateConversationMemory(agentUser, conv.ID, message, resp.Message); memErr != nil {
		fmt.Printf("[Memory] 更新对话记忆失败: %v\n", memErr)
	}

	// 更新会话标题
	if conv.Title == "" || conv.Title == "新对话" {
		title := message
		if len([]rune(title)) > 30 {
			title = string([]rune(title)[:30]) + "..."
		}
		s.db.Model(&conv).Update("title", title)
	}

	if loopErr == nil {
		engine.LearnRouteExampleCandidate(message, "agent_loop", "", 0.6)
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

// ChatStream sends a streaming chat response (via Agent Engine)
// Returns a channel that yields response chunks
func (s *AgentService) ChatStream(userID, message, conversationID, skillID string) (<-chan string, error) {
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

	// 初始化 Agent Engine 并执行 Agentic Loop
	engine := s.getEngine()
	var user models.User
	s.db.Preload("Groups").Where("id = ?", userID).First(&user)

	agentUser := buildAgentUser(s.db, user, skillID)

	ch := make(chan string, 20)
	var fullResponse strings.Builder

	go func() {
		defer close(ch)

		ctx, cancel := context.WithTimeout(context.Background(), agent.LoopTimeout)
		defer cancel()

		if fastResp, ok, fastErr := engine.TryFastPath(ctx, agentUser, message, conv.ID); fastErr == nil && ok {
			engine.LearnRouteExampleCandidate(message, fastResp.SkillUsedToRoute(), "", 0.7)
			for _, r := range fastResp.Message {
				ch <- string(r)
			}
			fullResponse.WriteString(fastResp.Message)
			// 发送技能使用元信息（供前端显示为标签）
			if fastResp.SkillUsed != "" {
				metaJSON, _ := json.Marshal(map[string]interface{}{"type": "meta", "skill_used": fastResp.SkillUsed})
				ch <- string(metaJSON)
			}
			aiMsg := models.Message{ConversationID: conv.ID, Role: "assistant", Content: fastResp.Message, TokenCount: fastResp.TokensUsed, SkillUsed: fastResp.SkillUsed, Metadata: fmt.Sprintf(`{"skill_used":"%s","fast_path":true}`, fastResp.SkillUsed)}
			s.db.Create(&aiMsg)
			_ = engine.UpdateConversationMemory(agentUser, conv.ID, message, fastResp.Message)
			return
		}

		// 创建事件 channel 并在后台执行 Agentic Loop
		eventsCh := make(chan agent.LoopEvent, 50)

		type loopResult struct {
			resp *agent.ChatResponse
			err  error
		}
		resultCh := make(chan loopResult, 1)

		go func() {
			resp, err := engine.ExecuteLoopWithEvents(ctx, agentUser, message, conv.ID, eventsCh)
			resultCh <- loopResult{resp, err}
		}()

		var finalResp *agent.ChatResponse
		var finalErr error

		// 消费事件并转换为 SSE 消息
	loop:
		for {
			select {
			case evt, ok := <-eventsCh:
				if ok {
					sseMsg := formatLoopEvent(evt)
					if sseMsg != "" {
						ch <- sseMsg
					}
				} else {
					// events channel 已关闭，获取结果
					result := <-resultCh
					finalResp = result.resp
					finalErr = result.err
					break loop
				}
			case result := <-resultCh:
				finalResp = result.resp
				finalErr = result.err
				// 排空剩余事件
				for evt := range eventsCh {
					sseMsg := formatLoopEvent(evt)
					if sseMsg != "" {
						ch <- sseMsg
					}
				}
				break loop
			}
		}

		if finalErr != nil {
			ch <- fmt.Sprintf("抱歉，处理请求时出错: %v", finalErr)
			return
		}

		// 逐字符输出以模拟流式效果
		responseText := finalResp.Message
		for _, r := range responseText {
			ch <- string(r)
		}
		fullResponse.WriteString(responseText)

		// 发送技能使用元信息（供前端显示为标签）
		if finalResp.SkillUsed != "" {
			metaJSON, _ := json.Marshal(map[string]interface{}{"type": "meta", "skill_used": finalResp.SkillUsed})
			ch <- string(metaJSON)
		}

		// 存储 AI 回复
		aiMsg := models.Message{
			ConversationID: conv.ID,
			Role:           "assistant",
			Content:        fullResponse.String(),
			SkillUsed:      finalResp.SkillUsed,
			Metadata:       fmt.Sprintf(`{"skill_used":"%s","tokens":%d}`, finalResp.SkillUsed, finalResp.TokensUsed),
		}
		s.db.Create(&aiMsg)
		if finalErr == nil {
			engine.LearnRouteExampleCandidate(message, "agent_loop", "", 0.6)
		}
		if memErr := engine.UpdateConversationMemory(agentUser, conv.ID, message, fullResponse.String()); memErr != nil {
			fmt.Printf("[Memory] 更新对话记忆失败: %v\n", memErr)
		}

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

// formatLoopEvent 将 LoopEvent 转换为人类可读的 SSE 消息
func formatLoopEvent(evt agent.LoopEvent) string {
	var msg string
	switch evt.Type {
	case "thinking":
		msg = fmt.Sprintf("💭 正在思考: %s", evt.Content)
	case "tool_start":
		toolName := ""
		if evt.Data != nil {
			if tn, ok := evt.Data["tool_name"].(string); ok {
				toolName = tn
			}
		}
		msg = fmt.Sprintf("🔧 正在调用: %s", toolName)
	case "tool_result":
		content := evt.Content
		if len([]rune(content)) > 200 {
			content = string([]rune(content)[:200]) + "..."
		}
		msg = content
	case "topic_route":
		action := ""
		from := ""
		to := ""
		if evt.Data != nil {
			if v, ok := evt.Data["action"].(string); ok {
				action = v
			}
			if v, ok := evt.Data["from_task_id"].(string); ok {
				from = v
			}
			if v, ok := evt.Data["to_task_id"].(string); ok {
				to = v
			}
		}
		msg = fmt.Sprintf("🧭 话题路由: %s (%s → %s) %s", action, from, to, evt.Content)
	case "final_token":
		// final_token 不单独作为 SSE 事件输出，由 ChatResponse.Message 逐字符流式输出
		return ""
	case "error":
		msg = fmt.Sprintf("⚠️ %s", evt.Content)
	default:
		msg = evt.Content
	}

	data, _ := json.Marshal(map[string]string{"type": "event", "content": msg})
	return string(data)
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

func (s *ConversationService) List(userID, source string) ([]models.Conversation, error) {
	var convs []models.Conversation
	query := s.db.Where("user_id = ?", userID)
	if source != "" {
		query = query.Where("source = ?", source)
	}
	err := query.Order("created_at DESC").Find(&convs).Error
	return convs, err
}

func (s *ConversationService) Get(id, userID string) (*models.Conversation, error) {
	var conv models.Conversation
	err := s.db.Where("id = ? AND user_id = ?", id, userID).Preload("Messages").First(&conv).Error
	return &conv, err
}

func (s *ConversationService) Delete(id, userID string) error {
	// 删除对话下的所有消息
	s.db.Where("conversation_id = ?", id).Delete(&models.Message{})
	// 删除对话本身
	result := s.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Conversation{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// buildAgentUser 构建 Agent 用户上下文（注入选中 Skill 配置）
func buildAgentUser(db *gorm.DB, user models.User, selectedSkillID string) *agent.UserContext {
	ctx := &agent.UserContext{
		UserID:       user.ID,
		GlobalUserID: user.GlobalUserID,
		Name:         user.Name,
		Department:   user.Department,
		Role:         user.Role,
		Status:       user.Status,
		Context:      make(map[string]interface{}),
	}

	for _, g := range user.Groups {
		ctx.Groups = append(ctx.Groups, agent.GroupContext{
			ID:              g.ID,
			Name:            g.GroupName,
			Code:            g.GroupCode,
			DataAccessScope: g.DataAccessScope,
		})
	}

	// 注入选中 Skill 的数据源和策略
	if selectedSkillID != "" {
		var skill models.Skill
		if err := db.Where("id = ? AND enabled = ?", selectedSkillID, true).First(&skill).Error; err == nil {
			ctx.Context["selected_skill_id"] = skill.ID
			ctx.Context["selected_skill_name"] = skill.Name
			ctx.Context["selected_skill_type"] = skill.SkillType
			ctx.Context["selected_skill_config"] = skill.Config
			ctx.Context["selected_skill_data_scope"] = skill.DataScope
			ctx.AvailableSkills = append(ctx.AvailableSkills, skill.ID)

			// 从 config 提取 data_source_id
			if skill.Config != "" {
				var cfg map[string]interface{}
				if json.Unmarshal([]byte(skill.Config), &cfg) == nil {
					if dsID, ok := cfg["data_source_id"].(string); ok && dsID != "" {
						ctx.Context["data_source_id"] = dsID
					}
				}
			}
		}
	}

	return ctx
}
