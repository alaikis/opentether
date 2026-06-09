package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Security  SecurityConfig  `yaml:"security"`
	Update    UpdateConfig    `yaml:"update"`
	Executor  ExecutorConfig  `yaml:"executor"`
	Embedding EmbeddingConfig `yaml:"embedding"`
	SMTP      SMTPConfig      `yaml:"smtp"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"` // development, production
}

type DatabaseConfig struct {
	Type        string `yaml:"type"` // sqlite, mysql, postgres
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Name        string `yaml:"name"`
	User        string `yaml:"user"`
	Password    string `yaml:"password"`
	SSLMode     string `yaml:"sslmode"`
	AutoMigrate bool   `yaml:"auto_migrate"`
}

type SecurityConfig struct {
	JWT        JWTConfig        `yaml:"jwt"`
	Encryption EncryptionConfig `yaml:"encryption"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
	CORS       CORSConfig       `yaml:"cors"`
	HTTPS      HTTPSConfig      `yaml:"https"`
}

type JWTConfig struct {
	Secret        string `yaml:"secret"`
	Expire        string `yaml:"expire"`
	RefreshExpire string `yaml:"refresh_expire"`
}

type EncryptionConfig struct {
	Key string `yaml:"key"`
}

type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
}

type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
}

type HTTPSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type UpdateConfig struct {
	Enabled         bool   `yaml:"enabled"`
	CheckInterval   string `yaml:"check_interval"`
	GithubRepo      string `yaml:"github_repo"`
	AutoBackup      bool   `yaml:"auto_backup"`
	RequireApproval bool   `yaml:"require_approval"`
}

type ExecutorConfig struct {
	Mode              string            `yaml:"mode"` // embedded, independent
	EmbeddedConfig    EmbeddedConfig    `yaml:"embedded"`
	IndependentConfig IndependentConfig `yaml:"independent"`
}

type EmbeddedConfig struct {
	MaxConcurrent int    `yaml:"max_concurrent"`
	Timeout       string `yaml:"timeout"`
}

type IndependentConfig struct {
	Queue QueueConfig `yaml:"queue"`
}

type QueueConfig struct {
	Type    string `yaml:"type"` // redis, kafka
	Address string `yaml:"address"`
}

// EmbeddingConfig 向量嵌入配置
// 未配置时默认使用内置 TF-IDF（零依赖）
type EmbeddingConfig struct {
	Model         string `yaml:"model"`      // 模型名称, 空=默认 tfidf
	ModelPath     string `yaml:"model_path"` // 模型文件路径
	Dimension     int    `yaml:"dimension"`  // 向量维度
	Provider      string `yaml:"provider"`   // embedding 提供者: tfidf, openai, local
	StoreProvider string `yaml:"store"`      // vectorstore 提供者: memory, milvus, qdrant
}

// SMTPConfig SMTP 邮件配置
type SMTPConfig struct {
	Enabled    bool   `yaml:"enabled"`    // 是否启用 SMTP
	Host       string `yaml:"host"`       // SMTP 服务器地址
	Port       int    `yaml:"port"`       // SMTP 端口 (常用 587 for TLS, 465 for SSL)
	Username   string `yaml:"username"`   // SMTP 用户名
	Password   string `yaml:"password"`   // SMTP 密码
	Encryption string `yaml:"encryption"` // 加密方式: none, tls, ssl
	FromEmail  string `yaml:"from_email"` // 发件人邮箱
	FromName   string `yaml:"from_name"`  // 发件人名称
	ToEmail    string `yaml:"to_email"`   // 收件人邮箱 (用于测试和通知)
}

func Load() *Config {
	// Default configuration
	cfg := &Config{
		Server: ServerConfig{
			Port: 8886,
			Mode: "development",
		},
		Database: DatabaseConfig{
			Type:        "none", // "none"=未配置(使用安装向导), "sqlite"=SQLite, "mysql"=MySQL, "postgres"=PostgreSQL
			Name:        "data/wisehoof.db",
			AutoMigrate: true,
		},
		Security: SecurityConfig{
			JWT: JWTConfig{
				Expire:        "24h",
				RefreshExpire: "7d",
			},
			RateLimit: RateLimitConfig{
				Enabled:           true,
				RequestsPerMinute: 60,
			},
			CORS: CORSConfig{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
			},
		},
		Update: UpdateConfig{
			Enabled:         false,
			CheckInterval:   "24h",
			AutoBackup:      true,
			RequireApproval: true,
		},
		Executor: ExecutorConfig{
			Mode: "embedded",
			EmbeddedConfig: EmbeddedConfig{
				MaxConcurrent: 5,
				Timeout:       "1h",
			},
		},
		SMTP: SMTPConfig{
			Enabled:    false,
			Host:       "smtp.example.com",
			Port:       587,
			Encryption: "tls",
			FromName:   "Wisehoof System",
		},
	}

	// Load from config.yaml if exists
	if _, err := os.Stat("config.yaml"); err == nil {
		data, err := os.ReadFile("config.yaml")
		if err != nil {
			return cfg
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return cfg
		}
	}

	// Override with environment variables
	if port := os.Getenv("SERVER_PORT"); port != "" {
		cfg.Server.Port = 8080
		fmt.Sscanf(port, "%d", &cfg.Server.Port)
	}
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		cfg.Security.JWT.Secret = jwtSecret
	}
	if dbPassword := os.Getenv("DB_PASSWORD"); dbPassword != "" {
		cfg.Database.Password = dbPassword
	}

	return cfg
}

// Load loads configuration from file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// SaveToFile saves configuration to file
func SaveToFile(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
