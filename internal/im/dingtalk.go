package im

import (
	"encoding/json"
	"fmt"

	"github.com/alaikis/opentether/internal/models"
)

// DingTalk implements PlatformHandler for DingTalk
type DingTalk struct {
	config    *models.ImConfig
	appKey    string
	appSecret string
}

// NewDingTalk creates a new DingTalk handler
func NewDingTalk(config *models.ImConfig) *DingTalk {
	return &DingTalk{
		config:    config,
		appKey:    getConfigString(config, "app_key"),
		appSecret: config.AppSecret,
	}
}

func (d *DingTalk) GetPlatform() Platform {
	return PlatformDingTalk
}

// ParseCallback parses DingTalk JSON callback
func (d *DingTalk) ParseCallback(data []byte) (*CallbackMessage, error) {
	var callback struct {
		MsgType  string `json:"msgtype"`
		Text     struct {
			Content string `json:"content"`
		} `json:"text"`
		Event     string `json:"event"`
		EventType string `json:"event_type"`
		SenderID  string `json:"senderId"`
		RobotCode string `json:"robotCode"`
		SessionId string `json:"sessionId"`
	}

	if err := json.Unmarshal(data, &callback); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &CallbackMessage{
		MsgType:    callback.MsgType,
		Content:    callback.Text.Content,
		FromUserID: callback.SenderID,
		RawData:    string(data),
		EventType:  callback.EventType,
	}, nil
}

// SendMessage sends a text message via DingTalk
func (d *DingTalk) SendMessage(toUserID, content string) error {
	// In a real implementation, this would call DingTalk API
	// POST https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend
	return fmt.Errorf("DingTalk send message not implemented - requires access token")
}

// VerifySignature verifies the DingTalk callback signature
func (d *DingTalk) VerifySignature(signature string, timestamp string, nonce string, data []byte) bool {
	if d.appSecret == "" {
		return false
	}

	// DingTalk uses: signature = HMAC-SHA256(timestamp + "\n" + secret, secret)
	_ = signature
	_ = timestamp
	_ = nonce
	_ = data

	return false
}
