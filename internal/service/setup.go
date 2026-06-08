package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/database"
	"github.com/alaikis/opentether/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SetupService 系统初始化服务
type SetupService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewSetupService(db *gorm.DB, cfg *config.Config) *SetupService {
	return &SetupService{db: db, cfg: cfg}
}

// IsInitialized 检查系统是否已初始化
func (s *SetupService) IsInitialized() (bool, error) {
	var count int64
	err := s.db.Model(&models.User{}).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SetupRequest 初始化请求
type SetupRequest struct {
	// 数据库配置
	DBType     string `json:"db_type"`     // mysql, postgres, sqlite
	DBHost     string `json:"db_host"`     // 主机
	DBPort     int    `json:"db_port"`     // 端口
	DBUser     string `json:"db_user"`     // 用户名
	DBPassword string `json:"db_password"` // 密码
	DBName     string `json:"db_name"`     // 数据库名

	// 管理员账号
	AdminUsername string `json:"admin_username"`
	AdminPassword string `json:"admin_password"`
	AdminName     string `json:"admin_name"`
	AdminEmail    string `json:"admin_email"`

	// 品牌配置
	AdminTitle string `json:"admin_title"`
	Theme      string `json:"theme"` // light, dark
}

// Setup 执行系统初始化
func (s *SetupService) Setup(req *SetupRequest) (map[string]interface{}, error) {
	// 1. 规范化 SQLite 数据库路径
	dbName := req.DBName
	if req.DBType == "sqlite" {
		// 如果 db_name 不包含路径分隔符，说明只是文件名，需要添加到 data/ 目录
		if !strings.Contains(dbName, "/") && !strings.Contains(dbName, "\\") {
			dbName = "data/" + dbName
		}
		// 如果没有 .db 后缀，添加它
		if !strings.HasSuffix(dbName, ".db") {
			dbName = dbName + ".db"
		}
		// 确保 data 目录存在
		dir := filepath.Dir(dbName)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建数据目录失败: %v", err)
		}
	}

	// 2. 根据请求参数创建数据库配置
	dbCfg := config.DatabaseConfig{
		Type:        req.DBType,
		Host:        req.DBHost,
		Port:        req.DBPort,
		Name:        dbName,
		User:        req.DBUser,
		Password:    req.DBPassword,
		AutoMigrate: true,
	}

	// 2. 测试数据库连接
	if req.DBType != "sqlite" {
		cfg := database.ExternalDBConfig{
			Host:     req.DBHost,
			Port:     req.DBPort,
			User:     req.DBUser,
			Password: req.DBPassword,
			Database: req.DBName,
			Type:     req.DBType,
		}

		result, err := database.TestConnection(cfg)
		if err != nil {
			return nil, fmt.Errorf("数据库连接测试失败: %v", err)
		}

		if !result["success"].(bool) {
			return result, nil
		}
	}

	// 3. 创建新的数据库连接
	db, err := database.Initialize(dbCfg)
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %v", err)
	}
	if db == nil {
		return nil, fmt.Errorf("无法创建数据库连接")
	}

	// 4. 运行迁移
	if err := database.Migrate(db); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %v", err)
	}

	// 5. 创建管理员账号
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %v", err)
	}

	// 生成 JWT secret
	jwtSecret := generateRandomKey(32)

	// 创建管理员用户
	user := models.User{
		ID:           generateUUID(),
		GlobalUserID: req.AdminUsername,
		Name:         req.AdminName,
		Email:        req.AdminEmail,
		PasswordHash: string(hashedPassword),
		Status:       "active",
		CreatedBy:    "system",
	}

	err = db.Create(&user).Error
	if err != nil {
		return nil, fmt.Errorf("创建管理员失败: %v", err)
	}

	// 保存配置到 config（这里简化处理）
	// 实际应该保存到数据库或配置文件
	_ = req.AdminTitle
	_ = req.Theme

	return map[string]interface{}{
		"success": true,
		"message": "系统初始化完成",
		"data": map[string]interface{}{
			"jwt_secret": jwtSecret,
			"admin_id":   user.ID,
		},
	}, nil
}

// generateRandomKey 生成随机密钥
func generateRandomKey(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateUUID 生成简单 UUID
func generateUUID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
