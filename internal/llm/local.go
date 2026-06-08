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

// LocalClient implements Client for local LLM servers (Ollama, LM Studio, etc.)
type LocalClient struct {
	provider   *models.Provider
	httpClient *http.Client
}

// NewLocalClient creates a new local LLM client
func NewLocalClient(provider *models.Provider) *LocalClient {
	return &LocalClient{
		provider:   provider,
		httpClient: &http.Client{},
	}
}

func (c *LocalClient) ChatCompletion(ctx interface{}, req ChatRequest) (*ChatResponse, error) {
	httpCtx := context.Background()
	if ctx != nil {
		if c, ok := ctx.(context.Context); ok {
			httpCtx = c
		}
	}

	// Convert to Ollama format
	ollamaReq := c.convertRequest(req)
	jsonData, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build URL - Ollama uses /api/chat endpoint
	url := c.provider.APIBase
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += "api/chat"

	httpReq, err := http.NewRequestWithContext(httpCtx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("Local API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result OllamaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return c.convertResponse(result), nil
}

func (c *LocalClient) ChatCompletionStream(ctx interface{}, req ChatRequest) (*StreamReader, error) {
	req.Stream = true

	ollamaReq := c.convertRequest(req)
	jsonData, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.provider.APIBase
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += "api/chat"

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

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

func (c *LocalClient) readStream(body io.Reader, stream *StreamReader) {
	defer close(stream.Done)
	defer close(stream.Chunks)

	decoder := json.NewDecoder(body)
	for {
		var delta OllamaResponse
		if err := decoder.Decode(&delta); err != nil {
			if err == io.EOF {
				break
			}
			stream.Err <- fmt.Errorf("failed to decode stream: %w", err)
			return
		}

		if delta.Message.Content != "" {
			stream.Chunks <- delta.Message.Content
		}

		// Check for done
		if delta.Done {
			break
		}
	}
}

func (c *LocalClient) GetModel() string {
	return c.provider.Model
}

func (c *LocalClient) GetProviderType() string {
	return c.provider.ProviderType
}

func (c *LocalClient) convertRequest(req ChatRequest) OllamaRequest {
	// Convert messages to Ollama format
	messages := make([]OllamaMessage, len(req.Messages))
	for i, msg := range req.Messages {
		role := msg.Role
		// Ollama uses "assistant" not "system" for assistant role
		// Map "system" to "user" if there's no dedicated system role
		if role == "system" {
			role = "system"
		}
		messages[i] = OllamaMessage{
			Role:    role,
			Content: msg.Content,
		}
	}

	model := req.Model
	if model == "" {
		model = "llama2"
	}

	return OllamaRequest{
		Model:       model,
		Messages:    messages,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		Stop:        req.Stop,
	}
}

func (c *LocalClient) convertResponse(resp OllamaResponse) *ChatResponse {
	return &ChatResponse{
		Content:      resp.Message.Content,
		Model:        resp.Model,
		FinishReason: "stop",
		Usage: Usage{
			PromptTokens:     0, // Ollama doesn't return token counts
			CompletionTokens: 0,
			TotalTokens:      0,
		},
	}
}

// Ollama API types

type OllamaRequest struct {
	Model       string          `json:"model"`
	Messages    []OllamaMessage `json:"messages"`
	Stream      bool            `json:"stream,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
}

type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaResponse struct {
	Model           string        `json:"model"`
	Message         OllamaMessage `json:"message"`
	Done            bool          `json:"done"`
	Context         []int         `json:"context,omitempty"`
	TotalDuration   int64         `json:"total_duration,omitempty"`
	LoadDuration    int64         `json:"load_duration,omitempty"`
	PromptEvalCount int           `json:"prompt_eval_count,omitempty"`
	EvalCount       int           `json:"eval_count,omitempty"`
}
