package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/alaikis/opentether/internal/agent"
	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

// MCPService MCP 协议集成服务
type MCPService struct {
	db      *gorm.DB
	servers map[string]*MCPServer
	mu      sync.RWMutex
}

type MCPServer struct {
	ID        string                       `json:"id"`
	Name      string                       `json:"name"`
	Command   string                       `json:"command"` // 执行命令，如 "npx", "python"
	Args      []string                     `json:"args"`    // 参数，如 ["-y", "@modelcontextprotocol/server-filesystem", "./data"]
	Env       map[string]string            `json:"env"`     // 环境变量
	Status    string                       `json:"status"`  // running, stopped, error
	Process   *exec.Cmd                    `json:"-"`
	Stdin     *os.File                     `json:"-"`
	Stdout    *os.File                     `json:"-"`
	Stderr    *os.File                     `json:"-"`
	writeMu   sync.Mutex                   `json:"-"`
	pendingMu sync.Mutex                   `json:"-"`
	pending   map[string]chan *MCPResponse `json:"-"`
	Tools     []MCPTool                    `json:"tools"`     // 可用工具
	Resources []MCPResource                `json:"resources"` // 可用资源
	Prompts   []MCPPrompt                  `json:"prompts"`   // 可用提示词
	LastError string                       `json:"last_error"`
	StartedAt time.Time                    `json:"started_at"`
}

type MCPTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mime_type"`
}

type MCPPrompt struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Arguments   []string `json:"arguments"`
}

type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type MCPError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func NewMCPService(db *gorm.DB) *MCPService {
	return &MCPService{
		db:      db,
		servers: make(map[string]*MCPServer),
	}
}

// StartEnabledServers starts all enabled MCP configs. It is safe to call at boot.
func (s *MCPService) StartEnabledServers() {
	configs, err := s.GetConfigs()
	if err != nil {
		return
	}
	for _, cfg := range configs {
		if cfg.Enabled {
			_ = s.StartServer(cfg.ID)
		}
	}
}

// StopAll stops all running MCP servers.
func (s *MCPService) StopAll() {
	s.mu.RLock()
	ids := make([]string, 0, len(s.servers))
	for id := range s.servers {
		ids = append(ids, id)
	}
	s.mu.RUnlock()
	for _, id := range ids {
		_ = s.StopServer(id)
	}
}

// MCPConfig MCP 服务器配置模型（由 models.MCPConfig 持久化，此处保留类型别名兼容 handler/service API）
type MCPConfig = models.MCPConfig

// SaveToDB 保存配置到数据库
func (s *MCPService) SaveToDB(config *MCPConfig) error {
	return s.db.Create(config).Error
}

// GetConfigs 获取所有 MCP 配置
func (s *MCPService) GetConfigs() ([]MCPConfig, error) {
	var configs []MCPConfig
	err := s.db.Where("enabled = ?", true).Order("created_at DESC").Find(&configs).Error
	return configs, err
}

// StartServer 启动 MCP 服务器
func (s *MCPService) StartServer(configID string) error {
	var config MCPConfig
	if err := s.db.First(&config, configID).Error; err != nil {
		return fmt.Errorf("MCP 配置不存在: %w", err)
	}

	// 检查是否已启动
	s.mu.Lock()
	if server, exists := s.servers[configID]; exists && server.Status == "running" {
		s.mu.Unlock()
		return nil // 已经启动
	}
	s.mu.Unlock()

	// 解析参数
	var args []string
	if config.Args != "" {
		json.Unmarshal([]byte(config.Args), &args)
	}

	// 解析环境变量
	envMap := make(map[string]string)
	if config.Env != "" {
		json.Unmarshal([]byte(config.Env), &envMap)
	}

	// 创建命令
	cmd := exec.Command(config.Command, args...)
	cmd.Env = os.Environ()
	for k, v := range envMap {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// 创建管道
	stdin, stdout, stderr, err := createPipes(cmd)
	if err != nil {
		return fmt.Errorf("创建管道失败: %w", err)
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 MCP 服务器失败: %w", err)
	}

	server := &MCPServer{
		ID:        configID,
		Name:      config.Name,
		Command:   config.Command,
		Args:      args,
		Env:       envMap,
		Status:    "running",
		Process:   cmd,
		Stdin:     stdin,
		Stdout:    stdout,
		Stderr:    stderr,
		pending:   make(map[string]chan *MCPResponse),
		StartedAt: time.Now(),
	}
	go s.readLoop(server)

	s.mu.Lock()
	s.servers[configID] = server
	s.mu.Unlock()

	// 更新数据库状态
	s.db.Model(&config).Update("status", "running")

	// 异步初始化 MCP (获取工具列表)
	go s.initializeServer(server)

	return nil
}

// createPipes 创建进程管道
func createPipes(cmd *exec.Cmd) (*os.File, *os.File, *os.File, error) {
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		return nil, nil, nil, err
	}

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return nil, nil, nil, err
	}

	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		return nil, nil, nil, err
	}

	cmd.Stdin = stdinR
	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW

	return stdinW, stdoutR, stderrR, nil
}

// initializeServer 初始化 MCP 服务器 (获取能力列表)
func (s *MCPService) initializeServer(server *MCPServer) {
	// 发送 initialize 请求
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 发送 initialize
	initReq := MCPRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"opentether","version":"1.0.0"}}`),
		ID:      1,
	}

	resp, err := s.sendRequest(server, initReq, 5*time.Second)
	if err != nil {
		server.LastError = fmt.Sprintf("初始化失败: %v", err)
		server.Status = "error"
		return
	}

	// 解析能力
	if resp.Result != nil {
		var result map[string]interface{}
		if json.Unmarshal(resp.Result, &result) == nil {
			if capabilities, ok := result["capabilities"].(map[string]interface{}); ok {
				if _, ok := capabilities["tools"]; ok {
					s.listTools(server)
				}
				if _, ok := capabilities["resources"]; ok {
					s.listResources(server)
				}
				if _, ok := capabilities["prompts"]; ok {
					s.listPrompts(server)
				}
			}
		}
	}

	// 发送 initialized 通知
	notif := MCPRequest{
		JSONRPC: "2.0",
		Method:  "initialized",
	}
	server.writeMu.Lock()
	_, _ = server.Stdin.Write([]byte(notif.String() + "\n"))
	server.writeMu.Unlock()
}

func (s *MCPService) readLoop(server *MCPServer) {
	reader := bufio.NewReader(server.Stdout)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			server.LastError = err.Error()
			server.Status = "error"
			server.pendingMu.Lock()
			for id, ch := range server.pending {
				close(ch)
				delete(server.pending, id)
			}
			server.pendingMu.Unlock()
			return
		}

		var resp MCPResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue
		}
		id := fmt.Sprint(resp.ID)
		if id == "" || id == "<nil>" {
			continue
		}
		server.pendingMu.Lock()
		ch := server.pending[id]
		delete(server.pending, id)
		server.pendingMu.Unlock()
		if ch != nil {
			ch <- &resp
			close(ch)
		}
	}
}

// sendRequest 发送请求并等待响应。响应由单 reader loop 分发，支持并发调用。
func (s *MCPService) sendRequest(server *MCPServer, req MCPRequest, timeout time.Duration) (*MCPResponse, error) {
	if req.ID == nil {
		req.ID = time.Now().UnixNano()
	}
	id := fmt.Sprint(req.ID)
	respCh := make(chan *MCPResponse, 1)

	server.pendingMu.Lock()
	server.pending[id] = respCh
	server.pendingMu.Unlock()

	reqBytes, err := json.Marshal(req)
	if err != nil {
		server.pendingMu.Lock()
		delete(server.pending, id)
		server.pendingMu.Unlock()
		return nil, err
	}

	server.writeMu.Lock()
	_, err = server.Stdin.Write(append(reqBytes, '\n'))
	server.writeMu.Unlock()
	if err != nil {
		server.pendingMu.Lock()
		delete(server.pending, id)
		server.pendingMu.Unlock()
		return nil, err
	}

	select {
	case resp, ok := <-respCh:
		if !ok || resp == nil {
			return nil, fmt.Errorf("MCP 连接已关闭")
		}
		return resp, nil
	case <-time.After(timeout):
		server.pendingMu.Lock()
		delete(server.pending, id)
		server.pendingMu.Unlock()
		return nil, fmt.Errorf("请求超时")
	}
}

// listTools 获取工具列表
func (s *MCPService) listTools(server *MCPServer) {
	req := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		Params:  json.RawMessage("{}"),
		ID:      2,
	}

	resp, err := s.sendRequest(server, req, 5*time.Second)
	if err != nil {
		return
	}

	if resp.Result != nil {
		var result map[string]interface{}
		if json.Unmarshal(resp.Result, &result) == nil {
			if tools, ok := result["tools"].([]interface{}); ok {
				server.Tools = make([]MCPTool, 0, len(tools))
				for _, t := range tools {
					if toolMap, ok := t.(map[string]interface{}); ok {
						tool := MCPTool{
							Name:        fmt.Sprintf("%v", toolMap["name"]),
							Description: fmt.Sprintf("%v", toolMap["description"]),
						}
						if inputSchema, ok := toolMap["inputSchema"]; ok {
							tool.InputSchema, _ = json.Marshal(inputSchema)
						}
						server.Tools = append(server.Tools, tool)
					}
				}
			}
		}
	}
}

func (s *MCPService) listResources(server *MCPServer) {
	req := MCPRequest{JSONRPC: "2.0", Method: "resources/list", Params: json.RawMessage("{}"), ID: 3}
	resp, err := s.sendRequest(server, req, 5*time.Second)
	if err != nil || resp.Result == nil {
		return
	}
	var result map[string]interface{}
	if json.Unmarshal(resp.Result, &result) != nil {
		return
	}
	items, _ := result["resources"].([]interface{})
	server.Resources = make([]MCPResource, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			server.Resources = append(server.Resources, MCPResource{
				URI:         fmt.Sprintf("%v", m["uri"]),
				Name:        fmt.Sprintf("%v", m["name"]),
				Description: fmt.Sprintf("%v", m["description"]),
				MimeType:    fmt.Sprintf("%v", m["mimeType"]),
			})
		}
	}
}

func (s *MCPService) listPrompts(server *MCPServer) {
	req := MCPRequest{JSONRPC: "2.0", Method: "prompts/list", Params: json.RawMessage("{}"), ID: 4}
	resp, err := s.sendRequest(server, req, 5*time.Second)
	if err != nil || resp.Result == nil {
		return
	}
	var result map[string]interface{}
	if json.Unmarshal(resp.Result, &result) != nil {
		return
	}
	items, _ := result["prompts"].([]interface{})
	server.Prompts = make([]MCPPrompt, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			prompt := MCPPrompt{Name: fmt.Sprintf("%v", m["name"]), Description: fmt.Sprintf("%v", m["description"])}
			if args, ok := m["arguments"].([]interface{}); ok {
				for _, arg := range args {
					prompt.Arguments = append(prompt.Arguments, fmt.Sprintf("%v", arg))
				}
			}
			server.Prompts = append(server.Prompts, prompt)
		}
	}
}

// ListAvailableTools 返回所有运行中的 MCP 工具，供 Agent Loop 注入工具列表。
func (s *MCPService) ListAvailableTools() []agent.MCPRuntimeTool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []agent.MCPRuntimeTool
	for _, server := range s.servers {
		if server.Status != "running" {
			continue
		}
		for _, tool := range server.Tools {
			result = append(result, agent.MCPRuntimeTool{
				ServerID:    server.ID,
				ServerName:  server.Name,
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.InputSchema,
			})
		}
	}
	return result
}

// CallTool 调用 MCP 工具
func (s *MCPService) CallTool(serverID, toolName string, arguments map[string]interface{}) (json.RawMessage, error) {
	s.mu.RLock()
	server, exists := s.servers[serverID]
	s.mu.RUnlock()

	if !exists || server.Status != "running" {
		return nil, fmt.Errorf("MCP 服务器未运行")
	}

	argsJSON, _ := json.Marshal(arguments)
	req := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  json.RawMessage(fmt.Sprintf(`{"name":"%s","arguments":%s}`, toolName, argsJSON)),
		ID:      time.Now().UnixNano(),
	}

	resp, err := s.sendRequest(server, req, 30*time.Second)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("MCP 错误: %s", resp.Error.Message)
	}

	return resp.Result, nil
}

// StopServer 停止 MCP 服务器
func (s *MCPService) StopServer(serverID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	server, exists := s.servers[serverID]
	if !exists {
		return nil
	}

	if server.Process != nil && server.Process.Process != nil {
		server.Process.Process.Kill()
	}

	server.Status = "stopped"
	delete(s.servers, serverID)

	// 更新数据库
	var config MCPConfig
	s.db.Model(&config).Where("id = ?", serverID).Update("status", "stopped")

	return nil
}

// GetServerStatus 获取服务器状态
func (s *MCPService) GetServerStatus(serverID string) (*MCPServer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	server, exists := s.servers[serverID]
	if !exists {
		return nil, fmt.Errorf("MCP 服务器不存在")
	}

	return server, nil
}

// ListTools 列出所有可用工具
func (s *MCPService) ListTools(serverID string) ([]MCPTool, error) {
	s.mu.RLock()
	server, exists := s.servers[serverID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("MCP 服务器不存在")
	}

	return server.Tools, nil
}

// String 实现 Stringer 接口
func (r *MCPRequest) String() string {
	b, _ := json.Marshal(r)
	return string(b)
}

// AutoStartFromSkill 从 Skill 配置自动启动 MCP 服务器
func (s *MCPService) AutoStartFromSkill(skillID string) error {
	var skill models.Skill
	if err := s.db.First(&skill, skillID).Error; err != nil {
		return err
	}

	// 从配置中提取 MCP 配置
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(skill.Config), &config); err != nil {
		return err
	}

	mcpConfig, ok := config["mcp"].(string)
	if !ok || mcpConfig == "" {
		return fmt.Errorf("Skill 没有 MCP 配置")
	}

	// 解析 MCP 配置
	var mcpConf map[string]interface{}
	if err := json.Unmarshal([]byte(mcpConfig), &mcpConf); err != nil {
		// 尝试 YAML 格式解析
		return fmt.Errorf("MCP 配置格式错误")
	}

	command, _ := mcpConf["command"].(string)
	args, _ := mcpConf["args"].([]interface{})

	argsStr := "[]"
	if len(args) > 0 {
		argsBytes, _ := json.Marshal(args)
		argsStr = string(argsBytes)
	}

	// 创建 MCP 配置记录
	mcpConfigRecord := &MCPConfig{
		ID:      skill.ID,
		Name:    skill.Name + " MCP",
		Command: command,
		Args:    argsStr,
		Env:     "{}",
		Enabled: true,
	}

	// 保存到数据库
	if err := s.db.Where("id = ?", skill.ID).FirstOrCreate(mcpConfigRecord, MCPConfig{ID: skill.ID}).Error; err != nil {
		return err
	}

	// 启动服务器
	return s.StartServer(skill.ID)
}
