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

// AzureClient implements Client for Azure OpenAI API
type AzureClient struct {
	provider   *models.Provider
	httpClient *http.Client
}

// NewAzureClient creates a new Azure OpenAI client
func NewAzureClient(provider *models.Provider) *AzureClient {
	return &AzureClient{
		provider:   provider,
		httpClient: &http.Client{},
	}
}

func (c *AzureClient) ChatCompletion(ctx interface{}, req ChatRequest) (*ChatResponse, error) {
	httpCtx := context.Background()
	if ctx != nil {
		if c, ok := ctx.(context.Context); ok {
			httpCtx = c
		}
	}

	// Azure uses deployment name in the URL, not model name
	deploymentName := c.provider.Model
	if deploymentName == "" {
		deploymentName = "gpt-4"
	}

	// Build Azure API URL
	// Format: {base_url}/openai/deployments/{deployment-name}/chat/completions?api-version={version}
	apiVersion := c.getAPIVersion()
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		strings.TrimRight(c.provider.APIBase, "/"),
		deploymentName,
		apiVersion,
	)

	openAIReq := c.convertRequest(req)
	jsonData, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(httpCtx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Api-Key", c.provider.APIKey)

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
		return nil, fmt.Errorf("Azure API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result OpenAIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return c.convertResponse(result), nil
}

func (c *AzureClient) ChatCompletionStream(ctx interface{}, req ChatRequest) (*StreamReader, error) {
	req.Stream = true

	deploymentName := c.provider.Model
	if deploymentName == "" {
		deploymentName = "gpt-4"
	}

	apiVersion := c.getAPIVersion()
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		strings.TrimRight(c.provider.APIBase, "/"),
		deploymentName,
		apiVersion,
	)

	openAIReq := c.convertRequest(req)
	jsonData, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Api-Key", c.provider.APIKey)

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

func (c *AzureClient) readStream(body io.Reader, stream *StreamReader) {
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

		if len(delta.Choices) > 0 && delta.Choices[0].Delta.Content != "" {
			stream.Chunks <- delta.Choices[0].Delta.Content
		}

		if len(delta.Choices) > 0 && delta.Choices[0].FinishReason != "" {
			break
		}
	}
}

func (c *AzureClient) GetModel() string {
	return c.provider.Model
}

func (c *AzureClient) GetProviderType() string {
	return c.provider.ProviderType
}

func (c *AzureClient) getAPIVersion() string {
	// Try to get API version from Config field
	var config struct {
		APIVersion string `json:"api_version"`
	}
	if c.provider.Config != "" {
		json.Unmarshal([]byte(c.provider.Config), &config)
	}
	if config.APIVersion != "" {
		return config.APIVersion
	}
	// Default to a stable version
	return "2024-02-15"
}

func (c *AzureClient) convertRequest(req ChatRequest) OpenAIRequest {
	messages := make([]OpenAIMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
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

	if openAIReq.MaxTokens == 0 {
		openAIReq.MaxTokens = 2048
	}
	if openAIReq.Temperature == 0 {
		openAIReq.Temperature = 0.7
	}

	return openAIReq
}

func (c *AzureClient) convertResponse(resp OpenAIResponse) *ChatResponse {
	if len(resp.Choices) == 0 {
		return &ChatResponse{
			Model: resp.Model,
		}
	}

	choice := resp.Choices[0]
	return &ChatResponse{
		Content:      choice.Message.Content,
		Model:        resp.Model,
		FinishReason: choice.FinishReason,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
}
