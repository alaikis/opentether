package llm

import (
	"errors"
	"fmt"

	"github.com/alaikis/opentether/internal/models"
)

// Client interface for LLM providers
type Client interface {
	// ChatCompletion sends a chat completion request and returns the response
	ChatCompletion(ctx interface{}, req ChatRequest) (*ChatResponse, error)

	// ChatCompletionStream sends a chat completion request with streaming
	ChatCompletionStream(ctx interface{}, req ChatRequest) (*StreamReader, error)

	// GetModel Returns the model name
	GetModel() string

	// GetProviderType returns the provider type
	GetProviderType() string
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
}

// Validate validates the chat request
func (r *ChatRequest) Validate() error {
	if len(r.Messages) == 0 {
		return errors.New("messages cannot be empty")
	}
	if r.Model == "" {
		return errors.New("model cannot be empty")
	}
	return nil
}

// Message represents a single message in the conversation
type Message struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	Content      string
	Model        string
	FinishReason string
	Usage        Usage
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamReader is returned when streaming is enabled
type StreamReader struct {
	// Channel that yields response chunks
	Chunks chan string
	// Error channel
	Err chan error
	// Done channel, closed when stream is complete
	Done chan struct{}
}

// NewClient creates a new LLM client based on the provider type
func NewClient(provider *models.Provider) (Client, error) {
	if provider == nil {
		return nil, errors.New("provider cannot be nil")
	}

	switch provider.ProviderType {
	case "openai":
		return NewOpenAIClient(provider), nil
	case "azure":
		return NewAzureClient(provider), nil
	case "anthropic":
		return NewAnthropicClient(provider), nil
	case "local":
		return NewLocalClient(provider), nil
	default:
		// Default to OpenAI-compatible client
		return NewOpenAIClient(provider), nil
	}
}

// ChatWithProvider sends a chat completion request using the specified provider
func ChatWithProvider(provider *models.Provider, req ChatRequest) (*ChatResponse, error) {
	client, err := NewClient(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	return client.ChatCompletion(nil, req)
}
