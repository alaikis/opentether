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
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Temperature    float64         `json:"temperature,omitempty"`
	TopP           float64         `json:"top_p,omitempty"`
	Stream         bool            `json:"stream,omitempty"`
	Stop           []string        `json:"stop,omitempty"`
	Tools          []Tool          `json:"tools,omitempty"`
	ToolChoice     string          `json:"tool_choice,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ResponseFormat struct {
	Type       string      `json:"type"`
	JSONSchema interface{} `json:"json_schema,omitempty"`
}

type Message struct {
	Role         string        `json:"role"`
	Content      string        `json:"content"`
	Name         string        `json:"name,omitempty"`
	ContentParts []ContentPart `json:"content_parts,omitempty"`
}

type ContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL *struct {
		URL    string `json:"url"`
		Detail string `json:"detail,omitempty"`
	} `json:"image_url,omitempty"`
}

type Tool struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatResponse struct {
	Content      string
	Model        string
	FinishReason string
	ToolCalls    []ToolCall
	Usage        Usage
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
