package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ApiKeyService API 密钥管理服务
type ApiKeyService struct {
	db *gorm.DB
}

func NewApiKeyService(db *gorm.DB) *ApiKeyService {
	return &ApiKeyService{db: db}
}

const (
	apiKeyPrefix = "ot_sk_"
	apiKeyLength = 48 // 前缀 6 + 随机 42 字符
)

// List 列出用户的所有 API 密钥（不返回完整密钥）
func (s *ApiKeyService) List(userID string) ([]models.ApiKey, error) {
	var keys []models.ApiKey
	query := s.db.Model(&models.ApiKey{})
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	err := query.Order("created_at desc").Find(&keys).Error
	return keys, err
}

// Create 创建新的 API 密钥
func (s *ApiKeyService) Create(userID string, name string, scopes []string, expiresInDays int) (*models.ApiKey, string, error) {
	if userID == "" {
		return nil, "", fmt.Errorf("user_id is required")
	}
	if name == "" {
		return nil, "", fmt.Errorf("name is required")
	}

	// 生成新密钥
	rawKey, err := generateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// 存储 key 的 SHA-256 哈希
	keyHash := hashAPIKey(rawKey)
	keyPrefix := rawKey[:len(apiKeyPrefix)+8] // ot_sk_ + 8 字符

	// 序列化 scopes
	scopesStr := "*"
	if len(scopes) > 0 {
		scopesStr = strings.Join(scopes, ",")
	}

	apiKey := models.ApiKey{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      name,
		Key:       keyHash,
		KeyPrefix: keyPrefix,
		Scopes:    scopesStr,
	}

	if expiresInDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, expiresInDays)
		apiKey.ExpiresAt = &expiresAt
	}

	if err := s.db.Create(&apiKey).Error; err != nil {
		return nil, "", fmt.Errorf("failed to create API key: %w", err)
	}

	// 返回原始密钥（仅此一次可见）
	return &apiKey, rawKey, nil
}

// Get 获取单个 API 密钥
func (s *ApiKeyService) Get(id string) (*models.ApiKey, error) {
	var key models.ApiKey
	if err := s.db.Preload("User").First(&key, id).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

// Delete 删除 API 密钥
func (s *ApiKeyService) Delete(id string) error {
	return s.db.Delete(&models.ApiKey{}, id).Error
}

// Validate 验证 API 密钥并返回对应的 ApiKey 记录
func (s *ApiKeyService) Validate(rawKey string) (*models.ApiKey, error) {
	if rawKey == "" {
		return nil, fmt.Errorf("api key is empty")
	}

	keyHash := hashAPIKey(rawKey)

	var apiKey models.ApiKey
	if err := s.db.Preload("User").Where("key = ?", keyHash).First(&apiKey).Error; err != nil {
		return nil, fmt.Errorf("invalid api key")
	}

	// 检查是否过期
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("api key expired")
	}

	// 更新最后使用时间
	now := time.Now()
	s.db.Model(&apiKey).Update("last_used_at", now)

	return &apiKey, nil
}

// Regenerate 重新生成 API 密钥（保留原始配置，生成新密钥）
func (s *ApiKeyService) Regenerate(id string) (*models.ApiKey, string, error) {
	var existing models.ApiKey
	if err := s.db.First(&existing, id).Error; err != nil {
		return nil, "", err
	}

	// 生成新密钥
	rawKey, err := generateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key: %w", err)
	}

	updates := map[string]interface{}{
		"key":          hashAPIKey(rawKey),
		"key_prefix":   rawKey[:len(apiKeyPrefix)+8],
		"last_used_at": nil,
	}

	if err := s.db.Model(&existing).Updates(updates).Error; err != nil {
		return nil, "", err
	}

	// 重新加载
	if err := s.db.First(&existing, id).Error; err != nil {
		return nil, "", err
	}

	return &existing, rawKey, nil
}

// generateAPIKey 生成安全的 API 密钥
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32) // 32 字节 = 64 十六进制字符
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return apiKeyPrefix + hex.EncodeToString(bytes), nil
}

// hashAPIKey 对原始 API 密钥进行 SHA-256 哈希
func hashAPIKey(rawKey string) string {
	h := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(h[:])
}
