package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

type InMemoryServerRegistry struct {
	mu      sync.RWMutex
	servers map[string]*MCPServerConfig
}

func NewInMemoryServerRegistry() *InMemoryServerRegistry {
	return &InMemoryServerRegistry{servers: make(map[string]*MCPServerConfig)}
}

func (r *InMemoryServerRegistry) Register(config *MCPServerConfig) error {
	if config == nil || config.ID == "" || config.Name == "" {
		return errors.New("invalid server config")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()
	r.servers[config.ID] = config
	return nil
}

func (r *InMemoryServerRegistry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.servers, id)
	return nil
}

func (r *InMemoryServerRegistry) Start(id string) error {
	r.mu.RLock()
	cfg, ok := r.servers[id]
	r.mu.RUnlock()
	if !ok {
		return errors.New("server not found")
	}
	if cfg.Transport == "stdio" && cfg.Command != "" {
		cmd := exec.Command(cfg.Command, cfg.Args...)
		cfg.Status = "running"
		r.mu.Lock()
		cfg.UpdatedAt = time.Now()
		r.mu.Unlock()
		_ = cmd.Start()
		return nil
	}
	return nil
}

func (r *InMemoryServerRegistry) Stop(id string) error {
	r.mu.RLock()
	cfg, ok := r.servers[id]
	r.mu.RUnlock()
	if !ok {
		return errors.New("server not found")
	}
	cfg.Status = "stopped"
	r.mu.Lock()
	cfg.UpdatedAt = time.Now()
	r.mu.Unlock()
	return nil
}

func (r *InMemoryServerRegistry) Get(id string) (*MCPServerConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if s, ok := r.servers[id]; ok {
		return s, nil
	}
	return nil, errors.New("server not found")
}

func (r *InMemoryServerRegistry) List() ([]*MCPServerConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*MCPServerConfig, 0, len(r.servers))
	for _, s := range r.servers {
		result = append(result, s)
	}
	return result, nil
}

type InMemoryToolDiscovery struct {
	registry ServerRegistry
	clients  map[string]http.Client
}

func NewInMemoryToolDiscovery(registry ServerRegistry) *InMemoryToolDiscovery {
	return &InMemoryToolDiscovery{
		registry: registry,
		clients:  make(map[string]http.Client),
	}
}

func (d *InMemoryToolDiscovery) DiscoverTools(ctx context.Context, serverID string) ([]*MCPTool, error) {
	cfg, err := d.registry.Get(serverID)
	if err != nil {
		return nil, err
	}
	if cfg.URL == "" {
		return []*MCPTool{}, nil
	}
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "POST", cfg.URL, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Tools []MCPTool `json:"tools"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	out := make([]*MCPTool, len(result.Tools))
	for i := range result.Tools {
		result.Tools[i].ServerID = serverID
		out[i] = &result.Tools[i]
	}
	return out, nil
}

func (d *InMemoryToolDiscovery) DiscoverResources(ctx context.Context, serverID string) ([]*MCPResource, error) {
	return []*MCPResource{}, nil
}

func (d *InMemoryToolDiscovery) DiscoverPrompts(ctx context.Context, serverID string) ([]*MCPPrompt, error) {
	return []*MCPPrompt{}, nil
}

func (d *InMemoryToolDiscovery) ListAllTools() ([]*MCPTool, error) {
	servers, err := d.registry.List()
	if err != nil {
		return nil, err
	}
	var all []*MCPTool
	for _, s := range servers {
		tools, err := d.DiscoverTools(context.Background(), s.ID)
		if err == nil {
			all = append(all, tools...)
		}
	}
	return all, nil
}

type InMemoryToolInvoker struct {
	registry ServerRegistry
	clients  map[string]*http.Client
}

func NewInMemoryToolInvoker(registry ServerRegistry) *InMemoryToolInvoker {
	return &InMemoryToolInvoker{
		registry: registry,
		clients:  make(map[string]*http.Client),
	}
}

func (i *InMemoryToolInvoker) CallTool(ctx context.Context, serverID, toolName string, arguments map[string]interface{}) (json.RawMessage, error) {
	cfg, err := i.registry.Get(serverID)
	if err != nil {
		return nil, err
	}
	if cfg.URL == "" {
		return json.RawMessage("{}"), nil
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"tool":      toolName,
		"arguments": arguments,
	})
	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "POST", cfg.URL, bytesReader(payload))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return json.RawMessage(body), nil
}

func (i *InMemoryToolInvoker) ReadResource(ctx context.Context, uri string) (json.RawMessage, error) {
	return json.RawMessage("{}"), nil
}

func bytesReader(b []byte) *bytesReaderImpl {
	return &bytesReaderImpl{data: b}
}

type bytesReaderImpl struct {
	data []byte
	idx  int
}

func (r *bytesReaderImpl) Read(p []byte) (int, error) {
	if r.idx >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.idx:])
	r.idx += n
	return n, nil
}

func (r *bytesReaderImpl) Close() error { return nil }
