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

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

// MCPService MCP 协议集成服务
type MCPService struct {
	db        *gorm.DB
	servers   map[string]*MCPServer
	mu        sync.RWMutex
}

type MCPServer struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Command     string            `json:"command"`      // 执行命令，如 "npx", "python"
	Args        []string          `json:"args"`         // 参数，如 ["-y", "@modelcontextprotocol/server-filesystem", "./data"]
	Env         map[string]string `json:"env"`          // 环境变量
	Status      string            `json:"status"`       // running, stopped, error
	Process     *exec.Cmd         `json:"-"`
	Stdin       *os.File          `json:"-"`
	Stdout      *os.File          `json:"-"`
	Stderr      *os.File          `json:"-"`
	Tools       []MCPTool         `json:"tools"`        // 可用工具
	Resources   []MCPResource     `json:"resources"`    // 可用资源
	Prompts     []MCPPrompt       `json:"prompts"`      // 可用提示词
	LastError   string            `json:"last_error"`
	StartedAt   time.Time         `json:"started_at"`
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

// MCPConfig MCP 服务器配置模型
type MCPConfig struct {
	ID          string            `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name        string            `json:"name" gorm:"type:varchar(100)"`
	Command     string            `json:"command" gorm:"type:varchar(500)"`
	Args        string            `json:"args" gorm:"type:text"` // JSON array
	Env         string            `json:"env" gorm:"type:text"`   // JSON object
	Enabled     bool              `json:"enabled" gorm:"default:true"`
	Status      string            `json:"status" gorm:"type:varchar(20)"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// SaveToDB 保存配置到数据库
func (s *MCPService) SaveToDB(config *MCPConfig) error {
	return s.db.Create(config).Error
}

// GetConfigs 获取所有 MCP 配置
func (s *MCPService) GetConfigs() ([]MCPConfig, error) {
	var configs []MCPConfig
	err := s.db.Where("enabled = ?", true).Find(&configs).Error
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
		StartedAt: time.Now(),
	}

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
				// 解析 tools
				if tools, ok := capabilities["tools"].(map[string]interface{}); ok {
					if list, ok := tools["list"].(bool); ok && list {
						// 发送 tools/list 请求
						s.listTools(server)
					}
				}
			}
		}
	}

	// 发送 initialized 通知
	notif := MCPRequest{
		JSONRPC: "2.0",
		Method:  "initialized",
	}
	server.Stdin.Write([]byte(notif.String() + "\n"))
}

// sendRequest 发送请求并等待响应
func (s *MCPService) sendRequest(server *MCPServer, req MCPRequest, timeout time.Duration) (*MCPResponse, error) {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// 写入请求
	_, err = server.Stdin.Write(append(reqBytes, '\n'))
	if err != nil {
		return nil, err
	}

	// 读取响应
	respCh := make(chan *MCPResponse, 1)
	errCh := make(chan error, 1)

	go func() {
		reader := bufio.NewReader(server.Stdout)
		line, err := reader.ReadString('\n')
		if err != nil {
			errCh <- err
			return
		}

		var resp MCPResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			errCh <- err
			return
		}
		respCh <- &resp
	}()

	select {
	case resp := <-respCh:
		return resp, nil
	case err := <-errCh:
		return nil, err
	case <-time.After(timeout):
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
