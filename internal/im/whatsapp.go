package im

import (
	"encoding/json"
	"fmt"

	"github.com/alaikis/opentether/internal/models"
)

// WhatsApp implements PlatformHandler for WhatsApp (Meta)
type WhatsApp struct {
	config        *models.ImConfig
	phoneNumberID string
	accessToken   string
	appSecret     string
}

// NewWhatsApp creates a new WhatsApp handler
func NewWhatsApp(config *models.ImConfig) *WhatsApp {
	return &WhatsApp{
		config:        config,
		phoneNumberID: getConfigString(config, "phone_number_id"),
		accessToken:   getConfigString(config, "access_token"),
		appSecret:     config.AppSecret,
	}
}

func (w *WhatsApp) GetPlatform() Platform {
	return PlatformWhatsApp
}

// ParseCallback parses WhatsApp Cloud API JSON callback
func (w *WhatsApp) ParseCallback(data []byte) (*CallbackMessage, error) {
	var callback struct {
		Object string `json:"object"`
		Entry  []struct {
			Changes []struct {
				Value struct {
					Messages []struct {
						From   string `json:"from"`
						Type   string `json:"type"`
						Text   struct {
							Body string `json:"body"`
						} `json:"text"`
						Image struct {
							Caption string `json:"caption"`
							ID      string `json:"id"`
						} `json:"image"`
						Document struct {
							Caption string `json:"caption"`
							ID      string `json:"id"`
							Filename string `json:"filename"`
						} `json:"document"`
						Audio struct {
							ID string `json:"id"`
						} `json:"audio"`
					} `json:"messages"`
					MessagingProduct string `json:"messaging_product"`
				} `json:"value"`
			} `json:"changes"`
		} `json:"entry"`
	}

	if err := json.Unmarshal(data, &callback); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Extract the first message
	if len(callback.Entry) == 0 || len(callback.Entry[0].Changes) == 0 {
		return nil, fmt.Errorf("no messages in callback")
	}

	changes := callback.Entry[0].Changes[0]
	value := changes.Value

	if len(value.Messages) == 0 {
		return nil, fmt.Errorf("no messages in value")
	}

	msg := value.Messages[0]

	result := &CallbackMessage{
		MsgType:    msg.Type,
		FromUserID: msg.From,
		RawData:    string(data),
	}

	switch msg.Type {
	case "text":
		result.Content = msg.Text.Body
	case "image":
		result.Content = fmt.Sprintf("[Image: %s]", msg.Image.Caption)
		result.Extra = map[string]interface{}{"image_id": msg.Image.ID}
	case "document":
		result.Content = fmt.Sprintf("[Document: %s]", msg.Document.Filename)
		result.Extra = map[string]interface{}{"document_id": msg.Document.ID}
	case "audio":
		result.Content = "[Audio message]"
		result.Extra = map[string]interface{}{"audio_id": msg.Audio.ID}
	default:
		result.Content = fmt.Sprintf("[Unsupported message type: %s]", msg.Type)
	}

	return result, nil
}

// SendMessage sends a text message via WhatsApp
func (w *WhatsApp) SendMessage(toUserID, content string) error {
	// In a real implementation, this would call Meta WhatsApp API
	// POST https://graph.facebook.com/v18.0/{phone-number-id}/messages
	// Headers: Authorization: Bearer {access_token}
	// Body: {"messaging_product": "whatsapp", "to": "user_id", "type": "text", "text": {"body": "content"}}
	return fmt.Errorf("WhatsApp send message not implemented - requires access token")
}

// VerifySignature verifies the WhatsApp Cloud API callback signature
func (w *WhatsApp) VerifySignature(signature string, timestamp string, nonce string, data []byte) bool {
	if w.appSecret == "" {
		return false
	}

	// WhatsApp uses: X-Hub-Signature-256 header with SHA256 of body
	// signature = sha256(body, app_secret)
	_ = signature
	_ = timestamp
	_ = nonce
	_ = data

	return false
}

// VerifyWebhook Verify webhook for WhatsApp (handles verification challenge)
func (w *WhatsApp) VerifyWebhook(mode, token, challenge string) (string, error) {
	if token != w.accessToken {
		return "", fmt.Errorf("invalid webhook token")
	}
	return challenge, nil
}
