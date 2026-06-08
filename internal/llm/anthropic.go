package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/alaikis/opentether/internal/models"
)

// AnthropicClient implements Client for Anthropic API
type AnthropicClient struct {
	provider   *models.Provider
	httpClient *http.Client
}

// NewAnthropicClient creates a new Anthropic client
func NewAnthropicClient(provider *models.Provider) *AnthropicClient {
	return &AnthropicClient{
		provider:   provider,
		httpClient: &http.Client{},
	}
}

func (c *AnthropicClient) ChatCompletion(ctx interface{}, req ChatRequest) (*ChatResponse, error) {
	httpCtx := context.Background()
	if ctx != nil {
		if c, ok := ctx.(context.Context); ok {
			httpCtx = c
		}
	}

	// Convert OpenAI-style messages to Anthropic format
	anthropicReq := c.convertRequest(req)
	jsonData, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.provider.APIBase + "/v1/messages"
	if !strings.HasSuffix(c.provider.APIBase, "/") {
		url = c.provider.APIBase + "/v1/messages"
	}

	httpReq, err := http.NewRequestWithContext(httpCtx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.provider.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result AnthropicResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return c.convertResponse(result), nil
}

func (c *AnthropicClient) ChatCompletionStream(ctx interface{}, req ChatRequest) (*StreamReader, error) {
	// Anthropic doesn't support true streaming, so we call non-streaming
	// and return the result as a single chunk
	resp, err := c.ChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	stream := &StreamReader{
		Chunks: make(chan string, 1),
		Err:    make(chan error, 1),
		Done:   make(chan struct{}),
	}

	// Send the response as a single chunk
	stream.Chunks <- resp.Content
	close(stream.Chunks)
	close(stream.Done)

	return stream, nil
}

func (c *AnthropicClient) GetModel() string {
	return c.provider.Model
}

func (c *AnthropicClient) GetProviderType() string {
	return c.provider.ProviderType
}

func (c *AnthropicClient) convertRequest(req ChatRequest) AnthropicRequest {
	// Anthropic uses system, user, assistant roles
	messages := make([]AnthropicMessage, len(req.Messages))
	for i, msg := range req.Messages {
		role := msg.Role
		// Convert "system" role to user for Anthropic if it's the first message
		// Anthropic handles system separately
		if role == "system" && i == 0 {
			continue // Skip system, we'll handle it separately
		}
		messages[i] = AnthropicMessage{
			Role:    role,
			Content: msg.Content,
		}
	}

	// Extract system message if present
	var system string
	if len(req.Messages) > 0 && req.Messages[0].Role == "system" {
		system = req.Messages[0].Content
		messages = messages[1:]
	}

	// Map model names
	model := req.Model
	modelMap := map[string]string{
		"claude-3-opus-20240229":     "claude-3-opus-20240229",
		"claude-3-sonnet-20240229":   "claude-3-sonnet-20240229",
		"claude-3-haiku-20240307":    "claude-3-haiku-20240307",
		"claude-3-5-sonnet-20241022": "claude-3-5-sonnet-20241022",
	}
	if m, ok := modelMap[model]; ok {
		model = m
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	return AnthropicRequest{
		Model:         model,
		Messages:      messages,
		MaxTokens:     maxTokens,
		System:        system,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		StopSequences: req.Stop,
	}
}

func (c *AnthropicClient) convertResponse(resp AnthropicResponse) *ChatResponse {
	if len(resp.Content) == 0 {
		return &ChatResponse{
			Model: resp.Model,
		}
	}

	content := ""
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &ChatResponse{
		Content:      content,
		Model:        resp.Model,
		FinishReason: string(resp.StopReason),
		Usage: Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}

// Anthropic API types

type AnthropicRequest struct {
	Model         string             `json:"model"`
	Messages      []AnthropicMessage `json:"messages"`
	MaxTokens     int                `json:"max_tokens"`
	System        string             `json:"system,omitempty"`
	Temperature   float64            `json:"temperature,omitempty"`
	TopP          float64            `json:"top_p,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	Content    []AnthropicContentBlock `json:"content"`
	Model      string                  `json:"model"`
	StopReason AnthropicStopReason     `json:"stop_reason"`
	Usage      AnthropicUsage          `json:"usage"`
}

type AnthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type AnthropicStopReason string

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
