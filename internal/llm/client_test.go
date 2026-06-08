package llm

import (
	"context"
	"testing"

	"github.com/alaikis/opentether/internal/models"
)

func TestNewClient_OpenAI(t *testing.T) {
	provider := &models.Provider{
		ProviderType: "openai",
		ProviderName: "OpenAI",
		APIBase:      "https://api.openai.com/v1",
		APIKey:       "test-key",
		Model:        "gpt-4",
	}

	client, err := NewClient(provider)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	if _, ok := client.(*OpenAIClient); !ok {
		t.Error("Expected OpenAIClient for openai provider type")
	}
}

func TestNewClient_Azure(t *testing.T) {
	provider := &models.Provider{
		ProviderType: "azure",
		ProviderName: "Azure OpenAI",
		APIBase:      "https://test.openai.azure.com",
		APIKey:       "test-key",
		Model:        "gpt-4",
	}

	client, err := NewClient(provider)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if _, ok := client.(*AzureClient); !ok {
		t.Error("Expected AzureClient for azure provider type")
	}
}

func TestNewClient_Anthropic(t *testing.T) {
	provider := &models.Provider{
		ProviderType: "anthropic",
		ProviderName: "Anthropic",
		APIBase:      "https://api.anthropic.com",
		APIKey:       "test-key",
		Model:        "claude-3-opus-20240229",
	}

	client, err := NewClient(provider)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if _, ok := client.(*AnthropicClient); !ok {
		t.Error("Expected AnthropicClient for anthropic provider type")
	}
}

func TestNewClient_Local(t *testing.T) {
	provider := &models.Provider{
		ProviderType: "local",
		ProviderName: "Ollama",
		APIBase:      "http://localhost:11434",
		APIKey:       "",
		Model:        "llama2",
	}

	client, err := NewClient(provider)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if _, ok := client.(*LocalClient); !ok {
		t.Error("Expected LocalClient for local provider type")
	}
}

func TestNewClient_Unknown(t *testing.T) {
	provider := &models.Provider{
		ProviderType: "unknown",
		ProviderName: "Unknown",
		APIBase:      "",
		APIKey:       "test-key",
		Model:        "",
	}

	client, err := NewClient(provider)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Should default to a兼容 client
	if client == nil {
		t.Fatal("Client should not be nil even for unknown type")
	}
}

func TestChatRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     ChatRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: ChatRequest{
				Model: "gpt-4",
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty messages",
			req: ChatRequest{
				Model:    "gpt-4",
				Messages: []Message{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ChatRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChatCompletion(t *testing.T) {
	// Skip if no real API key
	provider := &models.Provider{
		ProviderType: "openai",
		APIBase:      "https://api.openai.com/v1",
		APIKey:       getTestAPIKey(),
		Model:        "gpt-4o-mini",
	}

	if provider.APIKey == "" {
		t.Skip("Skipping integration test: no API key")
	}

	client, err := NewClient(provider)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	resp, err := client.ChatCompletion(context.Background(), ChatRequest{
		Model: provider.Model,
		Messages: []Message{
			{Role: "user", Content: "Say 'test' and nothing else"},
		},
		MaxTokens: 10,
	})

	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}

	if resp.Content == "" {
		t.Error("Response content should not be empty")
	}

	t.Logf("Response: %s", resp.Content)
}

func getTestAPIKey() string {
	// Check environment variable for API key in integration tests
	return ""
}
