package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/alaikis/opentether/internal/mcp"
	"gorm.io/gorm"
)

type MCPRegistryService struct {
	db         *gorm.DB
	registry   mcp.ServerRegistry
	discovery  mcp.ToolDiscovery
	invoker    mcp.ToolInvoker
	configDir  string
	watchers   map[string]*time.Ticker
	mu         sync.RWMutex
	invokeLogs []mcpToolInvocation
}

type mcpToolInvocation struct {
	ServerID    string
	ToolName    string
	Arguments   map[string]interface{}
	Result      string
	Error       string
	DurationMs  int64
	CalledAt    time.Time
}

func NewMCPRegistryService(db *gorm.DB) *MCPRegistryService {
	configDir := "data/mcp/configs"
	_ = os.MkdirAll(configDir, 0755)
	s := &MCPRegistryService{
		db:         db,
		registry:   mcp.NewInMemoryServerRegistry(),
		discovery:  mcp.NewInMemoryToolDiscovery(mcp.NewInMemoryServerRegistry()),
		invoker:    mcp.NewInMemoryToolInvoker(mcp.NewInMemoryServerRegistry()),
		configDir:  configDir,
		watchers:   make(map[string]*time.Ticker),
		invokeLogs: []mcpToolInvocation{},
	}
	s.discovery = mcp.NewInMemoryToolDiscovery(s.registry)
	s.invoker = mcp.NewInMemoryToolInvoker(s.registry)
	s.loadPresetServers()
	return s
}

func (s *MCPRegistryService) RegisterServer(id, name, command string, args []string, url string, headers map[string]string) error {
	if err := s.registry.Register(&mcp.MCPServerConfig{
		ID:        id,
		Name:      name,
		Command:   command,
		Args:      args,
		URL:       url,
		Headers:   headers,
		Enabled:   true,
		AutoStart: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}); err != nil {
		return err
	}
	s.startHotReload(id)
	return nil
}

func (s *MCPRegistryService) ListServers() ([]*mcp.MCPServerConfig, error) {
	return s.registry.List()
}

func (s *MCPRegistryService) StartServer(id string) error {
	return s.registry.Start(id)
}

func (s *MCPRegistryService) StopServer(id string) error {
	s.mu.Lock()
	if ticker, ok := s.watchers[id]; ok {
		ticker.Stop()
		delete(s.watchers, id)
	}
	s.mu.Unlock()
	return s.registry.Stop(id)
}

func (s *MCPRegistryService) GetServer(id string) (*mcp.MCPServerConfig, error) {
	return s.registry.Get(id)
}

func (s *MCPRegistryService) DiscoverTools(serverID string) ([]*mcp.MCPTool, error) {
	return s.discovery.DiscoverTools(nil, serverID)
}

func (s *MCPRegistryService) DiscoverAllTools() ([]*mcp.MCPTool, error) {
	return s.discovery.ListAllTools()
}

func (s *MCPRegistryService) CallTool(serverID, toolName string, arguments map[string]interface{}) ([]byte, error) {
	start := time.Now()
	result, err := s.invoker.CallTool(nil, serverID, toolName, arguments)
	duration := time.Since(start).Milliseconds()
	s.mu.Lock()
	s.invokeLogs = append(s.invokeLogs, mcpToolInvocation{
		ServerID:   serverID,
		ToolName:   toolName,
		Arguments:  arguments,
		Result:     string(result),
		Error:      errStr(err),
		DurationMs: duration,
		CalledAt:   start,
	})
	s.mu.Unlock()
	return result, err
}

func (s *MCPRegistryService) LogInvocation(serverID, toolName string, arguments map[string]interface{}, result []byte, err error) {
	s.CallTool(serverID, toolName, arguments)
}

func (s *MCPRegistryService) GetInvocationLogs() []mcpToolInvocation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	logs := make([]mcpToolInvocation, len(s.invokeLogs))
	copy(logs, s.invokeLogs)
	return logs
}

func (s *MCPRegistryService) loadPresetServers() {
	presets := []struct {
		id    string
		name  string
		cmd   string
		args  []string
	}{
		{"filesystem", "Filesystem", "npx", []string{"-y", "@modelcontextprotocol/server-filesystem", "./data"}},
		{"http", "HTTP", "npx", []string{"-y", "@modelcontextprotocol/server-http"}},
		{"database", "Database", "npx", []string{"-y", "@modelcontextprotocol/server-database"}},
	}
	for _, p := range presets {
		_ = s.registry.Register(&mcp.MCPServerConfig{
			ID:        p.id,
			Name:      p.name,
			Command:   p.cmd,
			Args:      p.args,
			Enabled:   true,
			AutoStart: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}
}

func (s *MCPRegistryService) startHotReload(id string) {
	s.mu.Lock()
	if _, ok := s.watchers[id]; ok {
		s.mu.Unlock()
		return
	}
	ticker := time.NewTicker(5 * time.Second)
	s.watchers[id] = ticker
	s.mu.Unlock()
	go func(serverID string) {
		for range ticker.C {
			path := filepath.Join(s.configDir, serverID+".json")
			if _, err := os.Stat(path); err == nil {
				s.reloadServerConfig(serverID, path)
			}
		}
	}(id)
}

func (s *MCPRegistryService) reloadServerConfig(id, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg mcp.MCPServerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return
	}
	cfg.ID = id
	_ = s.registry.Register(&cfg)
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
