package service

import (
	"errors"
	"time"

	"github.com/alaikis/opentether/internal/distributed"
	"gorm.io/gorm"
)

type DistributedService struct {
	db           *gorm.DB
	gossip       distributed.GossipProtocol
	distributor  distributed.TaskDistributor
	failureDet   distributed.FailureDetector
	nodes        map[string]*distributed.NodeInfo
	taskResults  map[string][]*distributed.TaskResult
}

func NewDistributedService(db *gorm.DB) *DistributedService {
	return &DistributedService{
		db:          db,
		gossip:      distributed.NewInMemoryGossipProtocol(),
		distributor: distributed.NewInMemoryTaskDistributor(),
		failureDet:  distributed.NewInMemoryFailureDetector(),
		nodes:       make(map[string]*distributed.NodeInfo),
		taskResults: make(map[string][]*distributed.TaskResult),
	}
}

func (s *DistributedService) RegisterNode(node *distributed.NodeInfo) error {
	if node == nil || node.ID == "" {
		return errors.New("invalid node")
	}
	s.nodes[node.ID] = node
	_ = s.failureDet.RegisterHeartbeat(node.ID)
	_ = s.gossip.Broadcast(&distributed.GossipMessage{
		Type:      "node_join",
		FromNode:  node.ID,
		Payload:   map[string]interface{}{"node": node},
		Timestamp: time.Now(),
	})
	return nil
}

func (s *DistributedService) DeregisterNode(id string) error {
	delete(s.nodes, id)
	return nil
}

func (s *DistributedService) ListNodes() ([]*distributed.NodeInfo, error) {
	result := make([]*distributed.NodeInfo, 0, len(s.nodes))
	for _, n := range s.nodes {
		result = append(result, n)
	}
	return result, nil
}

func (s *DistributedService) SubmitTask(assignment *distributed.TaskAssignment) error {
	return s.distributor.SubmitTask(nil, assignment)
}

func (s *DistributedService) CancelTask(taskID string) error {
	return s.distributor.CancelTask(nil, taskID)
}

func (s *DistributedService) GetTaskResults(taskID string) ([]*distributed.TaskResult, error) {
	return s.distributor.TaskResults(nil, taskID)
}

func (s *DistributedService) ListTasks() ([]*distributed.TaskAssignment, error) {
	return []*distributed.TaskAssignment{}, nil
}

func (s *DistributedService) RecordHeartbeat(nodeID string) error {
	return s.failureDet.RegisterHeartbeat(nodeID)
}

func (s *DistributedService) DetectFailures() ([]string, error) {
	return s.failureDet.DetectFailures(time.Now())
}
