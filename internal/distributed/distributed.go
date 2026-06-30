package distributed

import (
	"context"
	"time"
)

type NodeStatus string

const (
	NodeStatusOnline  NodeStatus = "online"
	NodeStatusOffline NodeStatus = "offline"
	NodeStatusBusy    NodeStatus = "busy"
)

type NodeInfo struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Host       string            `json:"host"`
	Port       int               `json:"port"`
	Version    string            `json:"version"`
	Status     NodeStatus        `json:"status"`
	Tags       []string          `json:"tags"`
	Load       float64           `json:"load"`
	LastSeen   time.Time         `json:"last_seen"`
	Registered time.Time         `json:"registered"`
	Metadata   map[string]string `json:"metadata"`
}

type TaskAssignment struct {
	TaskID    string                 `json:"task_id"`
	NodeID    string                 `json:"node_id"`
	Payload   map[string]interface{} `json:"payload"`
	Priority  int                    `json:"priority"`
	Timeout   string                 `json:"timeout"`
	CreatedAt time.Time              `json:"created_at"`
}

type TaskResult struct {
	TaskID    string                 `json:"task_id"`
	NodeID    string                 `json:"node_id"`
	Status    string                 `json:"status"`
	Output    map[string]interface{} `json:"output"`
	Error     string                 `json:"error,omitempty"`
	Duration  string                 `json:"duration"`
	FinishedAt time.Time             `json:"finished_at"`
}

type GossipMessage struct {
	Type      string                 `json:"type"`
	FromNode  string                 `json:"from_node"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
}

type GossipProtocol interface {
	Join(ctx context.Context, bootstrapNodes []string) error
	Leave(ctx context.Context) error
	Broadcast(msg *GossipMessage) error
	OnMessage(handler func(msg *GossipMessage))
	Peers() ([]*NodeInfo, error)
}

type TaskDistributor interface {
	SubmitTask(ctx context.Context, assignment *TaskAssignment) error
	CancelTask(ctx context.Context, taskID string) error
	RescheduleTask(ctx context.Context, taskID string) error
	TaskResults(ctx context.Context, taskID string) ([]*TaskResult, error)
}

type FailureDetector interface {
	RegisterHeartbeat(nodeID string) error
	RecordFailure(nodeID string) error
	DetectFailures(now time.Time) ([]string, error)
	IsAlive(nodeID string) bool
}
