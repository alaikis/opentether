package agent

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

// ============================================
// AgentHub - Master 侧独立 Agent 管理中心
//
// 管理所有注册的独立 Agent 节点:
// - 配对码验证与注册
// - 心跳监控
// - 任务分发（pull 模式）
// - 结果回收
// ============================================

// AgentHub 管理所有独立 Agent 节点
type AgentHub struct {
	db     *gorm.DB
	agents map[string]*AgentNode
	mu     sync.RWMutex
}

// AgentNode 一个已配对的独立 Agent 节点
type AgentNode struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Version    string    `json:"version"`
	Host       string    `json:"host"`   // Agent 所在机器
	Status     string    `json:"status"` // online, offline, busy
	LastSeen   time.Time `json:"last_seen"`
	Registered time.Time `json:"registered"`
	Tags       []string  `json:"tags"` // 标签: prod, test, gpu
}

func NewAgentHub(db *gorm.DB) *AgentHub {
	return &AgentHub{
		db:     db,
		agents: make(map[string]*AgentNode),
	}
}

// GeneratePairingCode 生成配对码（有效期 10 分钟）
func (h *AgentHub) GeneratePairingCode() (*models.AgentPairing, error) {
	code := models.GeneratePairingCode()
	pairing := &models.AgentPairing{
		Code:      code,
		Status:    "pending",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := h.db.Create(pairing).Error; err != nil {
		return nil, err
	}
	return pairing, nil
}

// RegisterAgent Agent 注册（通过配对码）
func (h *AgentHub) RegisterAgent(code, name, version, host string) (*AgentNode, error) {
	var pairing models.AgentPairing
	if err := h.db.Where("code = ? AND status = ?", code, "pending").First(&pairing).Error; err != nil {
		return nil, ErrInvalidPairingCode
	}

	if time.Now().After(pairing.ExpiresAt) {
		pairing.Status = "expired"
		h.db.Save(&pairing)
		return nil, ErrPairingCodeExpired
	}

	node := &AgentNode{
		ID:         models.GenerateID(),
		Name:       name,
		Version:    version,
		Host:       host,
		Status:     "online",
		LastSeen:   time.Now(),
		Registered: time.Now(),
	}

	pairing.AgentID = node.ID
	pairing.Status = "paired"
	now := time.Now()
	pairing.PairedAt = &now
	h.db.Save(&pairing)

	h.mu.Lock()
	h.agents[node.ID] = node
	h.mu.Unlock()

	return node, nil
}

// Heartbeat Agent 心跳
func (h *AgentHub) Heartbeat(agentID string) error {
	h.mu.Lock()
	node, ok := h.agents[agentID]
	if ok {
		node.LastSeen = time.Now()
		node.Status = "online"
	}
	h.mu.Unlock()

	if !ok {
		return ErrAgentNotFound
	}
	return nil
}

// SubmitTask Master 提交任务到指定 Agent
func (h *AgentHub) SubmitTask(agentID, taskType, payload string) (*models.AgentTask, error) {
	task := &models.AgentTask{
		ID:       models.GenerateID(),
		AgentID:  agentID,
		TaskType: taskType,
		Payload:  payload,
		Status:   "pending",
	}
	if err := h.db.Create(task).Error; err != nil {
		return nil, err
	}
	return task, nil
}

// PollTask Agent 轮询获取任务（pull 模式）
func (h *AgentHub) PollTask(agentID string) (*models.AgentTask, error) {
	var task models.AgentTask
	err := h.db.Where("agent_id = ? AND status = ?", agentID, "pending").
		Order("created_at ASC").
		First(&task).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	now := time.Now()
	task.Status = "running"
	task.StartedAt = &now
	h.db.Save(&task)

	h.mu.Lock()
	if node, ok := h.agents[agentID]; ok {
		node.Status = "busy"
	}
	h.mu.Unlock()

	return &task, nil
}

// ReportTaskResult Agent 上报任务结果
func (h *AgentHub) ReportTaskResult(agentID, taskID, result, errorMsg string) error {
	var task models.AgentTask
	if err := h.db.Where("id = ? AND agent_id = ?", taskID, agentID).First(&task).Error; err != nil {
		return err
	}

	now := time.Now()
	task.CompletedAt = &now
	if errorMsg != "" {
		task.Status = "failed"
		task.Error = errorMsg
	} else {
		task.Status = "done"
		task.Result = result
	}

	if err := h.db.Save(&task).Error; err != nil {
		return err
	}

	h.mu.Lock()
	if node, ok := h.agents[agentID]; ok {
		node.Status = "online"
	}
	h.mu.Unlock()

	return nil
}

// ListOnlineAgents 列出在线 Agent
func (h *AgentHub) ListOnlineAgents() []*AgentNode {
	h.mu.RLock()
	defer h.mu.RUnlock()

	agents := make([]*AgentNode, 0)
	for _, agent := range h.agents {
		agents = append(agents, agent)
	}
	return agents
}

// ListTasks 列出任务列表
func (h *AgentHub) ListTasks(agentID string, status string, limit int) ([]models.AgentTask, error) {
	var tasks []models.AgentTask
	query := h.db.Model(&models.AgentTask{})
	if agentID != "" {
		query = query.Where("agent_id = ?", agentID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if limit <= 0 {
		limit = 50
	}
	err := query.Order("created_at DESC").Limit(limit).Find(&tasks).Error
	return tasks, err
}

// ScanOfflineAgents 扫描离线 Agent（心跳超时 60 秒）
func (h *AgentHub) ScanOfflineAgents() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	for id, agent := range h.agents {
		if now.Sub(agent.LastSeen) > 60*time.Second {
			agent.Status = "offline"
			log.Printf("[AgentHub] Agent %s (%s) 离线", id, agent.Name)
		}
	}
}

// GetAgent 获取 Agent 信息
func (h *AgentHub) GetAgent(agentID string) (*AgentNode, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	node, ok := h.agents[agentID]
	if !ok {
		return nil, ErrAgentNotFound
	}
	return node, nil
}

var (
	ErrInvalidPairingCode = fmt.Errorf("配对码无效")
	ErrPairingCodeExpired = fmt.Errorf("配对码已过期")
	ErrAgentNotFound      = fmt.Errorf("Agent 不存在")
)
