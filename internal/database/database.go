package database

import (
	"fmt"
	"log"

	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/models"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Initialize(cfg config.DatabaseConfig) (*gorm.DB, error) {
	// 如果数据库类型为 "none"，表示未配置，跳过初始化
	if cfg.Type == "none" || cfg.Type == "" {
		return nil, nil
	}

	var dialector gorm.Dialector

	switch cfg.Type {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
		dialector = mysql.Open(dsn)
	case "postgres":
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
			cfg.Host, cfg.User, cfg.Password, cfg.Name, cfg.Port, cfg.SSLMode)
		dialector = postgres.Open(dsn)
	case "sqlite":
		dialector = sqlite.Open(cfg.Name)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Printf("Connected to database: %s", cfg.Type)
	return db, nil
}

func Migrate(db *gorm.DB) error {
	log.Println("Running database migrations...")

	err := db.AutoMigrate(
		&models.User{},
		&models.UserGroup{},
		&models.Role{},
		&models.Permission{},
		&models.Provider{},
		&models.DataSource{},
		&models.Skill{},
		&models.RouteExample{},
		&models.SkillRuntimeMemory{},
		&models.MCPConfig{},
		&models.ImConfig{},
		&models.ImBinding{},
		&models.Conversation{},
		&models.Message{},
		&models.ConversationState{},
		&models.RuntimeJob{},
		&models.RuntimeCheckpoint{},
		&models.AuditLog{},
		&models.ScheduledTask{},
		&models.TaskExecution{},
		&models.ApiKey{},
		&models.IMBindingToken{},
		&models.AgentTask{},
		&models.AgentPairing{},
		&models.AgentExperience{},
		&models.UserMemory{},
		&models.GroupMemory{},
		&models.AgentScript{},
		&models.SQLAudit{},
		&models.UserProfile{},
		&models.CompanyProfile{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// 注册内置系统 Skills
	seedSystemSkills(db)

	// 确保至少有一个管理员用户
	seedAdminUser(db)

	log.Println("Database migrations completed successfully")
	return nil
}

// seedSystemSkills 注册内置系统 Skills（首次启动时自动创建）
func seedSystemSkills(db *gorm.DB) {
	systemSkills := []struct {
		Name        string
		SkillType   string
		Description string
		Keywords    string
		Config      string
	}{
		{
			Name:        "通用对话",
			SkillType:   "chat",
			Description: "通用对话能力，回答各类问题、提供建议、解释概念",
			Keywords:    "对话,聊天,问答,帮助,解释,建议,分析",
			Config:      `{"builtin":true,"tool":"chat"}`,
		},
		{
			Name:        "数据查询",
			SkillType:   "text2sql",
			Description: "将自然语言转为 SQL 查询数据库。支持多步查询、数据分析、报表生成",
			Keywords:    "查询,SQL,数据,统计,订单,报表,分析,排名,趋势,销售额,业绩,库存,销量",
			Config:      `{"builtin":true,"tool":"text2sql"}`,
		},

		{
			Name:        "环境管理",
			SkillType:   "env_setup",
			Description: "安装 uv 包管理器并设置 Python 虚拟环境，自动检测脚本依赖并安装所需包",
			Keywords:    "环境,安装,依赖,pip,uv,python,包管理,虚拟环境",
			Config:      `{"builtin":true,"tool":"setup_env"}`,
		},
		{
			Name:        "脚本执行",
			SkillType:   "script_exec",
			Description: "执行 bash 或 Python 脚本。bash 优先，Python 在 uv 环境中运行。支持数据查询、文件处理、报表生成等任务",
			Keywords:    "脚本,执行,运行,bash,shell,python,自动化,任务",
			Config:      `{"builtin":true,"tool":"execute_script"}`,
		},
		{
			Name:        "PDF 报表",
			SkillType:   "pdf",
			Description: "生成图文并茂的 PDF 报表。支持表格数据、分析文字、图表、封面页脚",
			Keywords:    "PDF,报表,导出,下载,打印,报告,图表,图文",
			Config:      `{"builtin":true,"tool":"generate_pdf"}`,
		},
		{
			Name:        "Excel 处理",
			SkillType:   "excel",
			Description: "读取 Excel 文件数据，或生成 Excel 报表。支持多 Sheet、公式、图表、数据透视",
			Keywords:    "Excel,xlsx,表格,电子表格,导出Excel,读取Excel,数据透视",
			Config:      `{"builtin":true,"tool":"generate_report"}`,
		},
		{
			Name:        "Word 文档",
			SkillType:   "word",
			Description: "生成 Word 文档 (docx)。支持标题、段落、表格、图片、目录、页眉页脚",
			Keywords:    "Word,docx,文档,报告,合同,标书,公文,排版",
			Config:      `{"builtin":true,"tool":"generate_report"}`,
		},
		{
			Name:        "PPT 设计",
			SkillType:   "ppt",
			Description: "设计演示文稿 (PPT/PPTX)。支持主题、布局、图表、动画、演讲备注",
			Keywords:    "PPT,演示,幻灯片,汇报,演讲,展示,ppt,pptx",
			Config:      `{"builtin":true,"tool":"generate_report"}`,
		},
	}

	for _, sk := range systemSkills {
		var count int64
		db.Model(&models.Skill{}).Where("skill_type = ?", sk.SkillType).Count(&count)
		if count > 0 {
			continue // 已存在，跳过
		}

		skill := &models.Skill{
			Name:        sk.Name,
			SkillType:   sk.SkillType,
			Description: sk.Description,
			Keywords:    sk.Keywords,
			Category:    "系统内置",
			Enabled:     true,
			Config:      sk.Config,
		}
		if err := db.Create(skill).Error; err != nil {
			log.Printf("[Seed] 注册系统 Skill 失败 %s: %v", sk.Name, err)
		} else {
			log.Printf("[Seed] 注册系统 Skill: %s", sk.Name)
		}
	}
}

// seedAdminUser 确保至少有一个管理员用户（用户表为空时创建默认 admin）
func seedAdminUser(db *gorm.DB) {
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count > 0 {
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[Seed] 密码加密失败: %v", err)
		return
	}

	adminUser := &models.User{
		GlobalUserID: "admin",
		Name:         "管理员",
		Email:        "admin@example.com",
		Role:         models.RoleAdmin,
		Status:       "active",
		PasswordHash: string(hashedPassword),
		CreatedBy:    "system",
	}
	if err := db.Create(adminUser).Error; err != nil {
		log.Printf("[Seed] 创建默认管理员失败: %v", err)
	} else {
		log.Printf("[Seed] ========================================")
		log.Printf("[Seed]   默认管理员已创建")
		log.Printf("[Seed]   用户名: admin")
		log.Printf("[Seed]   密码:   admin123")
		log.Printf("[Seed]   ⚠️  请登录后立即修改密码！")
		log.Printf("[Seed] ========================================")
	}
}
