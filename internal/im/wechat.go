package im

import (
	"crypto/sha1"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"

	"github.com/alaikis/opentether/internal/models"
)

// WeChatWork implements PlatformHandler for WeChat Work (Enterprise WeChat)
type WeChatWork struct {
	config         *models.ImConfig
	corpID         string
	secret         string
	token          string
	encodingAESKey string
}

// NewWeChatWork creates a new WeChat Work handler
func NewWeChatWork(config *models.ImConfig) *WeChatWork {
	return &WeChatWork{
		config:         config,
		corpID:         config.AppID,
		secret:         config.AppSecret,
		token:          config.Token,
		encodingAESKey: getConfigString(config, "encoding_aes_key"),
	}
}

func (w *WeChatWork) GetPlatform() Platform {
	return PlatformWeCom
}

// ParseCallback parses WeChat Work XML callback
func (w *WeChatWork) ParseCallback(data []byte) (*CallbackMessage, error) {
	// Parse XML format
	var xmlMsg struct {
		XMLName      xml.Name `xml:"xml"`
		ToUserName   string   `xml:"ToUserName"`
		FromUserName string   `xml:"FromUserName"`
		CreateTime   int64    `xml:"CreateTime"`
		MsgType      string   `xml:"MsgType"`
		Content      string   `xml:"Content"`
		MsgID        int64    `xml:"MsgId"`
		Event        string   `xml:"Event"`
		EventKey     string   `xml:"EventKey"`
		AgentID      string   `xml:"AgentID"`
	}

	if err := xml.Unmarshal(data, &xmlMsg); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	return &CallbackMessage{
		MsgType:    xmlMsg.MsgType,
		Content:    xmlMsg.Content,
		FromUserID: xmlMsg.FromUserName,
		ToUserID:   xmlMsg.ToUserName,
		Timestamp:  xmlMsg.CreateTime,
		EventType:  xmlMsg.Event,
		RawData:    string(data),
		MessageID:  fmt.Sprintf("%d", xmlMsg.MsgID),
	}, nil
}

// SendMessage sends a text message via WeChat Work
func (w *WeChatWork) SendMessage(toUserID, content string) error {
	// In a real implementation, this would call the WeChat Work API
	// POST https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=ACCESS_TOKEN
	// {"touser":"userid","msgtype":"text","agentid":1,"text":{"content":"content"}}
	return fmt.Errorf("WeChat Work send message not implemented - requires access token")
}

// VerifySignature verifies the WeChat Work callback signature
func (w *WeChatWork) VerifySignature(signature, timestamp, nonce string, data []byte) bool {
	if w.token == "" {
		return false
	}

	// Sort token, timestamp, nonce
	strs := []string{w.token, timestamp, nonce}
	sort.Strings(strs)

	// Concatenate
	combined := strings.Join(strs, "")

	// Calculate SHA1
	h := sha1.Sum([]byte(combined))
	calcSig := fmt.Sprintf("%x", h)

	return calcSig == signature
}

// getConfigString extracts a string from config JSON
func getConfigString(config *models.ImConfig, key string) string {
	if config.Config == "" {
		return ""
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(config.Config), &m); err != nil {
		return ""
	}
	return m[key]
}
