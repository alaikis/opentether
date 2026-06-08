package im

import (
	"encoding/json"
	"fmt"

	"github.com/alaikis/opentether/internal/models"
)

// Feishu implements PlatformHandler for Feishu (Lark)
type Feishu struct {
	config   *models.ImConfig
	appID    string
	appSecret string
}

// NewFeishu creates a new Feishu handler
func NewFeishu(config *models.ImConfig) *Feishu {
	return &Feishu{
		config:    config,
		appID:     config.AppID,
		appSecret: config.AppSecret,
	}
}

func (f *Feishu) GetPlatform() Platform {
	return PlatformFeishu
}

// ParseCallback parses Feishu JSON callback
func (f *Feishu) ParseCallback(data []byte) (*CallbackMessage, error) {
	var callback struct {
		MsgType string `json:"msg_type"`
		Event   struct {
			Message struct {
				Content   string `json:"content"`
				SenderID  struct {
					UserID string `json:"user_id"`
					OpenID string `json:"open_id"`
				} `json:"sender_id"`
			} `json:"message"`
		} `json:"event"`
		Type   string `json:"type"`
		Header struct {
			EventType string `json:"event_type"`
		} `json:"header"`
	}

	if err := json.Unmarshal(data, &callback); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	msg := &CallbackMessage{
		MsgType:   callback.MsgType,
		Content:   callback.Event.Message.Content,
		FromUserID: callback.Event.Message.SenderID.UserID,
		RawData:   string(data),
		EventType: callback.Header.EventType,
	}

	// Try to get sender from alternative field
	if msg.FromUserID == "" {
		msg.FromUserID = callback.Event.Message.SenderID.OpenID
	}

	return msg, nil
}

// SendMessage sends a text message via Feishu
func (f *Feishu) SendMessage(toUserID, content string) error {
	// In a real implementation, this would call Feishu API
	// POST https://open.feishu.cn/open-apim/message/v3/send/
	// Requires access token and proper message format
	return fmt.Errorf("Feishu send message not implemented - requires access token")
}

// VerifySignature verifies the Feishu callback signature
func (f *Feishu) VerifySignature(signature string, timestamp string, nonce string, data []byte) bool {
	if f.appSecret == "" {
		return false
	}

	// Feishu uses: signature = HMAC-SHA256(timestamp + nonce + body, app_secret)
	// Then base64 encoded
	// This is a simplified version
	combined := timestamp + nonce + string(data)
	_ = combined
	_ = signature

	// Real implementation would calculate and compare signature
	return false
}

// GetAccessToken gets Feishu access token
func (f *Feishu) GetAccessToken() (string, error) {
	// In a real implementation, would call:
	// POST https://open.feishu.cn/open-apim/authen/v1/tenant_access_token/internal
	// with {"app_id": "...", "app_secret": "..."}
	return "", fmt.Errorf("not implemented")
}
