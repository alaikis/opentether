package mcp

import (
	"context"
	"encoding/json"
	"time"
)

type ServerStatus string

const (
	ServerStatusRunning ServerStatus = "running"
	ServerStatusStopped ServerStatus = "stopped"
	ServerStatusError   ServerStatus = "error"
)

type MCPServerConfig struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Transport string            `json:"transport"`
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Enabled   bool              `json:"enabled"`
	AutoStart bool              `json:"auto_start"`
	Status    ServerStatus      `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type MCPTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
	ServerID    string          `json:"server_id"`
}

type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ServerID    string `json:"server_id"`
}

type MCPPrompt struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ServerID    string `json:"server_id"`
}

type ServerRegistry interface {
	Register(config *MCPServerConfig) error
	Unregister(id string) error
	Start(id string) error
	Stop(id string) error
	Get(id string) (*MCPServerConfig, error)
	List() ([]*MCPServerConfig, error)
}

type ToolDiscovery interface {
	DiscoverTools(ctx context.Context, serverID string) ([]*MCPTool, error)
	DiscoverResources(ctx context.Context, serverID string) ([]*MCPResource, error)
	DiscoverPrompts(ctx context.Context, serverID string) ([]*MCPPrompt, error)
	ListAllTools() ([]*MCPTool, error)
}

type ToolInvoker interface {
	CallTool(ctx context.Context, serverID, toolName string, arguments map[string]interface{}) (json.RawMessage, error)
	ReadResource(ctx context.Context, uri string) (json.RawMessage, error)
}
