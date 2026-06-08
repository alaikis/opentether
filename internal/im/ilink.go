package im

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/alaikis/opentether/internal/models"
)

// iLink AI (ilinkai.weixin.qq.com) 平台适配器
// 通过微信公众号 + iLink Webhook 接入，支持多员工独立 OpenID 识别

type ILinkAI struct {
	config       *models.ImConfig
	appID        string
	appSecret    string
	token        string // iLink 配置的 Token（用于签名验证）
	encodingAESKey string
}

func NewILinkAI(config *models.ImConfig) *ILinkAI {
	return &ILinkAI{
		config:         config,
		appID:          config.AppID,
		appSecret:      config.AppSecret,
		token:          getConfigString(config, "token"),
		encodingAESKey: getConfigString(config, "encoding_aes_key"),
	}
}

func (i *ILinkAI) GetPlatform() Platform {
	return PlatformWeCom // 底层协议与微信一致，复用 PlatformWeCom
}

// ParseCallback 解析 iLink Webhook 回调消息
// iLink 支持两种模式：
// 1. 透传模式：直接将用户消息原文 POST 到 Webhook URL，附带用户 OpenID 等元信息
// 2. iLink 处理模式：iLink 先做意图识别，再回调（本例用透传模式）
func (i *ILinkAI) ParseCallback(data []byte) (*CallbackMessage, error) {
	var raw struct {
		// 基础字段
		MsgType string `json:"msg_type"` // text, image, event
		Content string `json:"content"`

		// 用户标识
		UserID   string `json:"user_id"`   // 用户 OpenID
		UserName string `json:"user_name"` // 微信昵称（可选）

		// 消息元信息
		MsgID     string `json:"msg_id"`
		Timestamp int64  `json:"timestamp"`
		Source    string `json:"source"` // "ilink"

		// 扩展字段
		Extra map[string]interface{} `json:"extra,omitempty"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("iLink 消息解析失败: %w", err)
	}

	if raw.UserID == "" {
		return nil, fmt.Errorf("iLink 回调缺少用户 OpenID")
	}

	return &CallbackMessage{
		MsgType:    raw.MsgType,
		Content:    raw.Content,
		FromUserID: raw.UserID, // OpenID → ImBinding.ImUserID
		Timestamp:  raw.Timestamp,
		RawData:    string(data),
		Extra:      raw.Extra,
		MessageID:  raw.MsgID,
	}, nil
}

// SendMessage 通过 iLink Webhook 发送回复消息
// iLink 使用 HTTP POST 回复，格式为 JSON
func (i *ILinkAI) SendMessage(toUserID, content string) error {
	// iLink 的回复通过 HTTP Response 或单独的回调 URL
	// 实际实现需要根据 iLink 配置的回复方式：
	// - 同步回复：直接返回 JSON（在 HTTP Response 中）
	// - 异步回复：POST 到 iLink 的消息推送接口

	// 这里返回格式提示，实际由 handleIMCallback 中的 response 处理
	_ = toUserID
	_ = content

	// 注：iLink 的同步回复在 HTTP Response 中完成，
	// 异步回复需要调用 iLink 的客服消息接口：
	// POST https://ilinkai.weixin.qq.com/api/message/send
	// Body: {"touser": toUserID, "msgtype": "text", "text": {"content": content}}
	return fmt.Errorf("iLink 回复需要访问令牌，请在 HTTP Response 中同步返回")
}

// VerifySignature 验证 iLink Webhook 签名
// iLink 使用与微信公众号相同的签名算法：
// 1. 将 token、timestamp、nonce 按字典序排序
// 2. SHA1 哈希
func (i *ILinkAI) VerifySignature(signature, timestamp, nonce string, data []byte) bool {
	if i.token == "" {
		return true // 未配置 token 时跳过验证
	}

	tmpArr := []string{i.token, timestamp, nonce}
	sort.Strings(tmpArr)
	tmpStr := strings.Join(tmpArr, "")

	hash := sha1.New()
	hash.Write([]byte(tmpStr))
	calculated := hex.EncodeToString(hash.Sum(nil))

	return calculated == signature
}

// FormatILinkReply 格式化 iLink 同步回复的 JSON
// iLink Webhook 期望的回复格式
func FormatILinkReply(toUserID, content string) map[string]interface{} {
	return map[string]interface{}{
		"msg_type": "text",
		"content":  content,
		"touser":   toUserID,
	}
}
