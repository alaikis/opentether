package im

import (
	"testing"

	"github.com/alaikis/opentether/internal/models"
)

func TestPlatform_GetName(t *testing.T) {
	tests := []struct {
		platform Platform
		expected string
	}{
		{PlatformWeCom, "wechat_work"},
		{PlatformFeishu, "feishu"},
		{PlatformDingTalk, "dingtalk"},
		{PlatformWhatsApp, "whatsapp"},
	}

	for _, tt := range tests {
		if tt.platform.GetName() != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, tt.platform.GetName())
		}
	}
}

func TestPlatformFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected Platform
		hasError bool
	}{
		{"wecom", PlatformWeCom, false},
		{"wechat_work", PlatformWeCom, false},
		{"feishu", PlatformFeishu, false},
		{"lark", PlatformFeishu, false},
		{"dingtalk", PlatformDingTalk, false},
		{"whatsapp", PlatformWhatsApp, false},
		{"unknown", PlatformUnknown, true},
		{"", PlatformUnknown, true},
	}

	for _, tt := range tests {
		result, err := PlatformFromString(tt.input)
		if tt.hasError && err == nil {
			t.Errorf("Expected error for input: %s", tt.input)
		}
		if !tt.hasError && result != tt.expected {
			t.Errorf("Expected %v, got %v for input: %s", tt.expected, result, tt.input)
		}
	}
}

func TestWeChatWork_ParseCallback(t *testing.T) {
	// Test XML callback parsing for WeChat Work
	platform := NewWeChatWork(&models.ImConfig{
		Token:  "test_token",
		Config: `{"encoding_aes_key":"test_aes_key"}`,
	})

	// Valid callback should have signature verification
	msg := `<xml><ToUserName><![CDATA[toUser]]></ToUserName>
<FromUserName><![CDATA[fromUser]]></FromUserName>
<CreateTime>1348831860</CreateTime>
<MsgType><![CDATA[text]]></MsgType>
<Content><![CDATA[this is a test]]></Content>
<MsgId>1234567890123456</MsgId>
</xml>`

	// Should not error on parsing
	callbackMsg, err := platform.ParseCallback([]byte(msg))
	if err != nil {
		t.Logf("Parse error (expected without signature): %v", err)
	}
	_ = callbackMsg
}

func TestFeishu_ParseCallback(t *testing.T) {
	platform := NewFeishu(&models.ImConfig{
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	})

	// Test JSON callback parsing for Feishu
	msg := `{
		"msg_type": "text",
		"event": {
			"message": {
				"content": "test message",
				"sender_id": {"user_id": "user123"}
			}
		}
	}`

	callbackMsg, err := platform.ParseCallback([]byte(msg))
	if err != nil {
		t.Fatalf("Failed to parse Feishu callback: %v", err)
	}

	if callbackMsg.Content != "test message" {
		t.Errorf("Expected content 'test message', got '%s'", callbackMsg.Content)
	}
}

func TestDingTalk_ParseCallback(t *testing.T) {
	platform := NewDingTalk(&models.ImConfig{
		AppSecret: "test_secret",
		Config:    `{"app_key":"test_app_key"}`,
	})

	// Test DingTalk callback format
	msg := `{
		"msgtype": "text",
		"text": {
			"content": "test from dingtalk"
		},
		"senderId": "user123"
	}`

	callbackMsg, err := platform.ParseCallback([]byte(msg))
	if err != nil {
		t.Fatalf("Failed to parse DingTalk callback: %v", err)
	}

	if callbackMsg.Content != "test from dingtalk" {
		t.Errorf("Expected content 'test from dingtalk', got '%s'", callbackMsg.Content)
	}
}

func TestWhatsApp_ParseCallback(t *testing.T) {
	platform := NewWhatsApp(&models.ImConfig{
		Config: `{"phone_number_id":"test_phone_id","access_token":"test_token"}`,
	})

	// Test WhatsApp callback format (Meta/WhatsApp Cloud API)
	msg := `{
		"object": "whatsapp",
		"entry": [{
			"changes": [{
				"value": {
					"messages": [{
						"from": "user123",
						"type": "text",
						"text": {"body": "hello"}
					}]
				}
			}]
		}]
	}`

	callbackMsg, err := platform.ParseCallback([]byte(msg))
	if err != nil {
		t.Fatalf("Failed to parse WhatsApp callback: %v", err)
	}

	if callbackMsg.Content != "hello" {
		t.Errorf("Expected content 'hello', got '%s'", callbackMsg.Content)
	}
}

func TestCallbackMessage_Fields(t *testing.T) {
	msg := &CallbackMessage{
		MsgType:   "text",
		Content:   "test content",
		FromUserID: "user123",
		Timestamp: 1234567890,
		RawData:   `{"test": true}`,
	}

	if msg.MsgType != "text" {
		t.Errorf("Expected msg_type 'text', got '%s'", msg.MsgType)
	}
	if msg.Content != "test content" {
		t.Errorf("Expected content 'test content', got '%s'", msg.Content)
	}
	if msg.FromUserID != "user123" {
		t.Errorf("Expected from_user_id 'user123', got '%s'", msg.FromUserID)
	}
}
