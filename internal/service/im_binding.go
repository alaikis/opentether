package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================================
// IM 自助绑定（扫码流程）
// ============================================

// PlatformInfo 平台信息（返回给前端用于展示绑定方式）
type PlatformInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	Description string `json:"description"`
	BindMethod  string `json:"bind_method"` // qrcode, token, manual
	QrcodeURL   string `json:"qrcode_url,omitempty"`
	HelpText    string `json:"help_text"`
}

// GenerateBindingToken 为员工生成 IM 绑定 token
// 返回绑定信息（含 token，前端可据此生成二维码或展示验证指引）
func (s *IMService) GenerateBindingToken(userID, imConfigID string) (map[string]interface{}, error) {
	// 检查 IM 配置是否存在且启用
	var cfg models.ImConfig
	if err := s.db.Where("id = ? AND enabled = ?", imConfigID, true).First(&cfg).Error; err != nil {
		return nil, fmt.Errorf("IM 平台不存在或未启用")
	}

	// 检查是否已绑定
	var existing models.ImBinding
	err := s.db.Where("user_id = ? AND im_config_id = ?", userID, imConfigID).First(&existing).Error
	if err == nil {
		return nil, fmt.Errorf("已绑定该平台: %s", existing.ImUserName)
	}

	// 生成绑定 token（有效期 30 分钟）
	token := uuid.New().String()
	bindingToken := &models.IMBindingToken{
		UserID:     userID,
		ImConfigID: imConfigID,
		Token:      token,
		Status:     "pending",
		ExpiresAt:  time.Now().Add(30 * time.Minute),
	}

	if err := s.db.Create(bindingToken).Error; err != nil {
		return nil, fmt.Errorf("生成绑定 token 失败: %w", err)
	}

	// 构造绑定 URL（供 IM 平台回调使用）
	bindURL := fmt.Sprintf("/api/v1/external/im/confirm-bind?token=%s", token)

	result := map[string]interface{}{
		"token_id":       bindingToken.ID,
		"token":          token,
		"platform":       cfg.Platform,
		"platform_name":  cfg.Name,
		"bind_url":       bindURL,
		"expires_at":     bindingToken.ExpiresAt,
		"expires_in":     "30 minutes",
		"qrcode_content": bindURL, // 前端可用此内容生成二维码
		"help_text":      getPlatformHelpText(cfg.Platform),
	}

	return result, nil
}

// ConfirmBinding 通过 token 确认 IM 绑定（由 IM 回调或前端提交）
func (s *IMService) ConfirmBinding(token, imUserID, imUserName string) (*models.ImBinding, error) {
	// 查找有效的绑定 token
	var bindingToken models.IMBindingToken
	if err := s.db.Where("token = ? AND status = ?", token, "pending").
		Preload("ImConfig").First(&bindingToken).Error; err != nil {
		return nil, fmt.Errorf("无效的绑定 token")
	}

	// 检查过期
	if bindingToken.ExpiresAt.Before(time.Now()) {
		bindingToken.Status = "expired"
		s.db.Save(&bindingToken)
		return nil, fmt.Errorf("绑定 token 已过期，请重新发起绑定")
	}

	// 检查 Duplicate
	var existing models.ImBinding
	err := s.db.Where("im_config_id = ? AND im_user_id = ?", bindingToken.ImConfigID, imUserID).First(&existing).Error
	if err == nil {
		// 已存在绑定，更新用户映射
		if existing.UserID != bindingToken.UserID {
			existing.UserID = bindingToken.UserID
			existing.ImUserName = imUserName
			s.db.Save(&existing)
		}
		bindingToken.Status = "confirmed"
		s.db.Save(&bindingToken)
		return &existing, nil
	}

	// 创建绑定
	binding := models.ImBinding{
		ImConfigID:   bindingToken.ImConfigID,
		UserID:       bindingToken.UserID,
		ImUserID:     imUserID,
		ImUserName:   imUserName,
		BindingToken: token,
		Status:       "active",
	}

	if err := s.db.Create(&binding).Error; err != nil {
		return nil, fmt.Errorf("创建绑定失败: %w", err)
	}

	// 标记 token 已使用
	bindingToken.Status = "confirmed"
	s.db.Save(&bindingToken)

	return &binding, nil
}

// ListAvailablePlatforms 列出所有可绑定的 IM 平台
func (s *IMService) ListAvailablePlatforms() ([]PlatformInfo, error) {
	var configs []models.ImConfig
	if err := s.db.Where("enabled = ?", true).Order("created_at DESC").Find(&configs).Error; err != nil {
		return nil, err
	}

	platforms := make([]PlatformInfo, 0, len(configs))
	for _, cfg := range configs {
		platforms = append(platforms, PlatformInfo{
			ID:          cfg.ID,
			Name:        cfg.Name,
			Platform:    cfg.Platform,
			Description: getPlatformDescription(cfg.Platform),
			BindMethod:  getPlatformBindMethod(cfg.Platform),
			HelpText:    getPlatformHelpText(cfg.Platform),
		})
	}

	return platforms, nil
}

// ExternalBindUser 外部系统通过 API Key 代员工绑定 IM
	// companyID: 公司 ID（外部系统传入，如 OA/ERP 中的公司标识）
	// globalUserID: 外部系统（OA/ERP）中的用户唯一标识
	// userName: 员工姓名
	func (s *IMService) ExternalBindUser(companyID, globalUserID, userName, imConfigID, imUserID, imUserName string) (map[string]interface{}, error) {
		// 查找或创建用户
		var user models.User
		err := s.db.Where("global_user_id = ?", globalUserID).First(&user).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// 自动创建用户
				user = models.User{
					CompanyID:    companyID,
					GlobalUserID: globalUserID,
					Name:         userName,
					Role:         models.RoleUser,
					Status:       "active",
				}
			if createErr := s.db.Create(&user).Error; createErr != nil {
				return nil, fmt.Errorf("创建用户失败: %w", createErr)
			}
		} else {
			return nil, fmt.Errorf("查询用户失败: %w", err)
		}
	}

	// 检查 IM 配置
	var cfg models.ImConfig
	if err := s.db.Where("id = ? AND enabled = ?", imConfigID, true).First(&cfg).Error; err != nil {
		return nil, fmt.Errorf("IM 平台不存在或未启用")
	}

	// 检查是否已有绑定
	var existing models.ImBinding
	err = s.db.Where("im_config_id = ? AND im_user_id = ?", imConfigID, imUserID).First(&existing).Error
	if err == nil {
		// 更新已有绑定
		existing.UserID = user.ID
		existing.ImUserName = imUserName
		existing.Status = "active"
		s.db.Save(&existing)
		return map[string]interface{}{
			"binding_id": existing.ID,
			"user_id":    user.ID,
			"platform":   cfg.Platform,
			"action":     "updated",
		}, nil
	}

	// 创建新绑定
	binding := models.ImBinding{
		ImConfigID:   imConfigID,
		UserID:       user.ID,
		ImUserID:     imUserID,
		ImUserName:   imUserName,
		BindingToken: uuid.New().String(),
		Status:       "active",
	}

	if err := s.db.Create(&binding).Error; err != nil {
		return nil, fmt.Errorf("创建绑定失败: %w", err)
	}

	// 尝试从配置中提取 MCP 等自动化配置
	if cfg.Config != "" {
		var extraConfig map[string]interface{}
		if json.Unmarshal([]byte(cfg.Config), &extraConfig) == nil {
			// 配置中包含 MCP 自动化等，由调度器处理
		}
	}

	return map[string]interface{}{
		"binding_id": binding.ID,
		"user_id":    user.ID,
		"platform":   cfg.Platform,
		"action":     "created",
	}, nil
}

// getPlatformDescription 获取平台描述
func getPlatformDescription(platform string) string {
	descriptions := map[string]string{
		"wecom":             "企业微信 - 支持扫码一键绑定",
		"personal-wechat":   "个人微信 - 通过微信扫码绑定",
		"feishu":            "飞书 - 支持扫码绑定",
		"dingtalk":          "钉钉 - 支持扫码绑定",
		"whatsapp":          "WhatsApp - 通过验证码绑定",
		"whatsapp-business": "WhatsApp Business - 通过验证码绑定",
		"whatsapp-personal": "WhatsApp Personal - 通过验证码绑定",
		"ilink":             "iLink AI - 微信公众号扫码绑定",
	}
	if desc, ok := descriptions[platform]; ok {
		return desc
	}
	return "即时通讯平台绑定"
}

// getPlatformBindMethod 获取平台绑定方式
func getPlatformBindMethod(platform string) string {
	switch platform {
	case "wecom", "feishu", "dingtalk", "ilink":
		return "qrcode"
	case "whatsapp", "whatsapp-business", "whatsapp-personal":
		return "token"
	default:
		return "manual"
	}
}

// getPlatformHelpText 获取平台绑定帮助文本
func getPlatformHelpText(platform string) string {
	helpTexts := map[string]string{
		"wecom":             "请使用企业微信扫描二维码完成绑定。绑定后可直接在企业微信中与 AI 助手对话。",
		"personal-wechat":   "请使用微信扫描二维码完成绑定。",
		"feishu":            "请使用飞书扫描二维码完成绑定。",
		"dingtalk":          "请使用钉钉扫描二维码完成绑定。",
		"whatsapp":          "请向 WhatsApp 机器人发送验证码完成绑定。",
		"whatsapp-business": "请向 WhatsApp Business 机器人发送验证码完成绑定。",
		"whatsapp-personal": "请向 WhatsApp 机器人发送验证码完成绑定。",
		"ilink":             "请使用微信扫码关注公众号完成绑定，关注后即可在微信中与 AI 助手对话。",
	}
	if text, ok := helpTexts[platform]; ok {
		return text
	}
	return "请按照平台指引完成绑定。"
}
