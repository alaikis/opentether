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

// OpenAIClient implements Client for OpenAI API
type OpenAIClient struct {
	provider   *models.Provider
	httpClient *http.Client
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(provider *models.Provider) *OpenAIClient {
	return &OpenAIClient{
		provider:   provider,
		httpClient: &http.Client{},
	}
}

func (c *OpenAIClient) ChatCompletion(ctx interface{}, req ChatRequest) (*ChatResponse, error) {
	// Use context if it's a proper context.Context
	var httpCtx context.Context
	if ctx != nil {
		if c, ok := ctx.(context.Context); ok {
			httpCtx = c
		} else {
			httpCtx = context.Background()
		}
	} else {
		httpCtx = context.Background()
	}

	openAIReq := c.convertRequest(req)
	jsonData, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.provider.APIBase
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += "chat/completions"

	httpReq, err := http.NewRequestWithContext(httpCtx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.provider.APIKey)
	httpReq.Header.Set("OpenAI-Organization", "") // Optional

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
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result OpenAIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return c.convertResponse(result), nil
}

func (c *OpenAIClient) ChatCompletionStream(ctx interface{}, req ChatRequest) (*StreamReader, error) {
	req.Stream = true

	openAIReq := c.convertRequest(req)
	jsonData, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.provider.APIBase
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += "chat/completions"

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.provider.APIKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	stream := &StreamReader{
		Chunks: make(chan string, 10),
		Err:    make(chan error, 1),
		Done:   make(chan struct{}),
	}

	go c.readStream(resp.Body, stream)

	return stream, nil
}

func (c *OpenAIClient) readStream(body io.Reader, stream *StreamReader) {
	defer close(stream.Done)
	defer close(stream.Chunks)

	decoder := json.NewDecoder(body)
	for {
		var delta OpenAIDelta
		if err := decoder.Decode(&delta); err != nil {
			if err == io.EOF {
				break
			}
			stream.Err <- fmt.Errorf("failed to decode stream: %w", err)
			return
		}

		if len(delta.Choices) > 0 {
			if s := contentString(delta.Choices[0].Delta.Content); s != "" {
				stream.Chunks <- s
			}
		}

		// Check for finish
		if len(delta.Choices) > 0 && delta.Choices[0].FinishReason != "" {
			break
		}
	}
}

func contentString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func (c *OpenAIClient) GetModel() string {
	return c.provider.Model
}

func (c *OpenAIClient) GetProviderType() string {
	return c.provider.ProviderType
}

func (c *OpenAIClient) convertRequest(req ChatRequest) OpenAIRequest {
	messages := make([]OpenAIMessage, len(req.Messages))
	for i, msg := range req.Messages {
		content := interface{}(msg.Content)
		if len(msg.ContentParts) > 0 {
			parts := make([]ContentPart, len(msg.ContentParts))
			copy(parts, msg.ContentParts)
			content = parts
		}
		messages[i] = OpenAIMessage{
			Role:    msg.Role,
			Content: content,
			Name:    msg.Name,
		}
	}

	openAIReq := OpenAIRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
		Stop:        req.Stop,
	}
	if req.ResponseFormat != nil {
		openAIReq.ResponseFormat = &OpenAIResponseFormat{Type: req.ResponseFormat.Type, JSONSchema: req.ResponseFormat.JSONSchema}
	}
	if len(req.Tools) > 0 {
		openAIReq.Tools = make([]OpenAITool, len(req.Tools))
		for i, t := range req.Tools {
			openAIReq.Tools[i] = OpenAITool{Type: t.Type, Function: OpenAIFunctionDef{Name: t.Function.Name, Description: t.Function.Description, Parameters: t.Function.Parameters}}
		}
	}
	if req.ToolChoice != "" {
		openAIReq.ToolChoice = req.ToolChoice
	}

	// Set defaults
	if openAIReq.MaxTokens == 0 {
		openAIReq.MaxTokens = 2048
	}
	if openAIReq.Temperature == 0 {
		openAIReq.Temperature = 0.7
	}

	return openAIReq
}

func (c *OpenAIClient) convertResponse(resp OpenAIResponse) *ChatResponse {
	if len(resp.Choices) == 0 {
		return &ChatResponse{
			Model: resp.Model,
		}
	}

	choice := resp.Choices[0]
	msg := choice.Message
	var toolCalls []ToolCall
	for _, tc := range msg.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{ID: tc.ID, Type: tc.Type, Function: FunctionCall{Name: tc.Function.Name, Arguments: tc.Function.Arguments}})
	}
	return &ChatResponse{
		Content:      contentString(msg.Content),
		Model:        resp.Model,
		FinishReason: choice.FinishReason,
		ToolCalls:    toolCalls,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
}

// OpenAI API types

type OpenAIRequest struct {
	Model          string                `json:"model"`
	Messages       []OpenAIMessage       `json:"messages"`
	MaxTokens      int                   `json:"max_tokens,omitempty"`
	Temperature    float64               `json:"temperature,omitempty"`
	TopP           float64               `json:"top_p,omitempty"`
	Stream         bool                  `json:"stream,omitempty"`
	Stop           []string              `json:"stop,omitempty"`
	Tools          []OpenAITool          `json:"tools,omitempty"`
	ToolChoice     interface{}           `json:"tool_choice,omitempty"`
	ResponseFormat *OpenAIResponseFormat `json:"response_format,omitempty"`
}

type OpenAIResponseFormat struct {
	Type       string      `json:"type"`
	JSONSchema interface{} `json:"json_schema,omitempty"`
}

type OpenAIMessage struct {
	Role      string           `json:"role"`
	Content   interface{}      `json:"content"`
	Name      string           `json:"name,omitempty"`
	ToolCalls []OpenAIToolCall `json:"tool_calls,omitempty"`
}

type OpenAITool struct {
	Type     string            `json:"type"`
	Function OpenAIFunctionDef `json:"function"`
}

type OpenAIFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type OpenAIToolCall struct {
	ID       string                `json:"id"`
	Type     string                `json:"type"`
	Function OpenAIFunctionCallDef `json:"function"`
}

type OpenAIFunctionCallDef struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
}

type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type OpenAIDelta struct {
	Choices []OpenAIDeltaChoice `json:"choices"`
}

type OpenAIDeltaChoice struct {
	Index        int           `json:"index"`
	Delta        OpenAIMessage `json:"delta"`
	FinishReason string        `json:"finish_reason,omitempty"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
