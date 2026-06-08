package im

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alaikis/opentether/internal/models"
)

// Platform represents an IM platform type
type Platform int

const (
	PlatformUnknown Platform = iota
	PlatformWeCom
	PlatformFeishu
	PlatformDingTalk
	PlatformWhatsApp
)

func (p Platform) String() string {
	switch p {
	case PlatformWeCom:
		return "wecom"
	case PlatformFeishu:
		return "feishu"
	case PlatformDingTalk:
		return "dingtalk"
	case PlatformWhatsApp:
		return "whatsapp"
	default:
		return "unknown"
	}
}

func (p Platform) GetName() string {
	switch p {
	case PlatformWeCom:
		return "wechat_work"
	case PlatformFeishu:
		return "feishu"
	case PlatformDingTalk:
		return "dingtalk"
	case PlatformWhatsApp:
		return "whatsapp"
	default:
		return "unknown"
	}
}

// PlatformFromString converts a string to Platform
func PlatformFromString(s string) (Platform, error) {
	switch s {
	case "wecom", "wechat_work", "wechat", "企微":
		return PlatformWeCom, nil
	case "feishu", "lark", "飞书", " Lark":
		return PlatformFeishu, nil
	case "dingtalk", "ding", "钉钉":
		return PlatformDingTalk, nil
	case "whatsapp", "whats app":
		return PlatformWhatsApp, nil
	case "ilink", "ilinkai", "wechat_oa":
		return PlatformWeCom, nil // iLink 复用微信公众号协议
	default:
		return PlatformUnknown, errors.New("unknown platform: " + s)
	}
}

// PlatformHandler is the interface for IM platform handlers
type PlatformHandler interface {
	// GetPlatform returns the platform type
	GetPlatform() Platform

	// ParseCallback parses the incoming callback data
	ParseCallback(data []byte) (*CallbackMessage, error)

	// SendMessage sends a message to a user
	SendMessage(toUserID, content string) error

	// VerifySignature verifies the request signature
	VerifySignature(signature string, timestamp string, nonce string, data []byte) bool
}

// CallbackMessage represents a parsed callback message
type CallbackMessage struct {
	MsgType     string                 // Message type: text, image, event, etc.
	Content     string                 // Text content
	FromUserID  string                 // Sender user ID
	ToUserID    string                 // Recipient user ID
	Timestamp   int64                  // Message timestamp
	EventType   string                 // Event type for event messages
	RawData     string                 // Raw JSON/XML data
	Extra       map[string]interface{} // Extra fields
	MessageID   string                 // Unique message ID
	ReplyToken  string                 // Reply token for async responses
}

// NewPlatformHandler creates a platform handler based on config
func NewPlatformHandler(config *models.ImConfig) (PlatformHandler, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	platform, err := PlatformFromString(config.Platform)
	if err != nil {
		return nil, fmt.Errorf("invalid platform: %w", err)
	}

	switch platform {
	case PlatformWeCom:
		// iLink 和企微共用微信协议，根据 Config 中的 platform 字段区分
		if config.Platform == "ilink" || config.Platform == "ilinkai" || config.Platform == "wechat_oa" {
			return NewILinkAI(config), nil
		}
		return NewWeChatWork(config), nil
	case PlatformFeishu:
		return NewFeishu(config), nil
	case PlatformDingTalk:
		return NewDingTalk(config), nil
	case PlatformWhatsApp:
		return NewWhatsApp(config), nil
	default:
		return nil, errors.New("unsupported platform")
	}
}

// PlatformNames returns a list of supported platform names
func PlatformNames() []string {
	return []string{
		"wecom",
		"feishu",
		"dingtalk",
		"whatsapp",
		"ilink",
	}
}

// MarshalJSON implements custom JSON marshaling for Platform
func (p Platform) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

// UnmarshalJSON implements custom JSON unmarshaling for Platform
func (p *Platform) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	platform, err := PlatformFromString(s)
	if err != nil {
		return err
	}
	*p = platform
	return nil
}
