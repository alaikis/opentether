package distributed

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"
)

type InMemoryGossipProtocol struct {
	mu       sync.RWMutex
	peers    map[string]*NodeInfo
	messages chan *GossipMessage
}

func NewInMemoryGossipProtocol() *InMemoryGossipProtocol {
	return &InMemoryGossipProtocol{
		peers:    make(map[string]*NodeInfo),
		messages: make(chan *GossipMessage, 1024),
	}
}

func (g *InMemoryGossipProtocol) Join(ctx context.Context, bootstrapNodes []string) error {
	for _, addr := range bootstrapNodes {
		peer := &NodeInfo{
			ID:         addr,
			Host:       addr,
			Port:       0,
			Status:     NodeStatusOnline,
			Registered: time.Now(),
			LastSeen:   time.Now(),
		}
		g.mu.Lock()
		g.peers[addr] = peer
		g.mu.Unlock()
	}
	go g.processMessages(ctx)
	return nil
}

func (g *InMemoryGossipProtocol) Leave(ctx context.Context) error {
	close(g.messages)
	return nil
}

func (g *InMemoryGossipProtocol) Broadcast(msg *GossipMessage) error {
	if msg == nil {
		return errors.New("nil message")
	}
	msg.Timestamp = time.Now()
	select {
	case g.messages <- msg:
		return nil
	default:
		return errors.New("message queue full")
	}
}

func (g *InMemoryGossipProtocol) OnMessage(handler func(msg *GossipMessage)) {
	go func() {
		for msg := range g.messages {
			handler(msg)
		}
	}()
}

func (g *InMemoryGossipProtocol) Peers() ([]*NodeInfo, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := make([]*NodeInfo, 0, len(g.peers))
	for _, p := range g.peers {
		result = append(result, p)
	}
	return result, nil
}

func (g *InMemoryGossipProtocol) processMessages(ctx context.Context) {
	for msg := range g.messages {
		g.mu.Lock()
		if peer, ok := g.peers[msg.FromNode]; ok {
			peer.LastSeen = time.Now()
		}
		g.mu.Unlock()
	}
}

type InMemoryTaskDistributor struct {
	mu        sync.RWMutex
	tasks     map[string]*TaskAssignment
	results   map[string][]*TaskResult
	pending   map[string]context.CancelFunc
}

func NewInMemoryTaskDistributor() *InMemoryTaskDistributor {
	return &InMemoryTaskDistributor{
		tasks:   make(map[string]*TaskAssignment),
		results: make(map[string][]*TaskResult),
		pending: make(map[string]context.CancelFunc),
	}
}

func (d *InMemoryTaskDistributor) SubmitTask(ctx context.Context, assignment *TaskAssignment) error {
	if assignment == nil || assignment.TaskID == "" {
		return errors.New("invalid task assignment")
	}
	assignment.CreatedAt = time.Now()
	d.mu.Lock()
	d.tasks[assignment.TaskID] = assignment
	d.results[assignment.TaskID] = []*TaskResult{}
	ctx, cancel := context.WithCancel(ctx)
	d.pending[assignment.TaskID] = cancel
	d.mu.Unlock()
	go d.simulateExecution(ctx, assignment)
	return nil
}

func (d *InMemoryTaskDistributor) CancelTask(ctx context.Context, taskID string) error {
	d.mu.RLock()
	cancel, ok := d.pending[taskID]
	d.mu.RUnlock()
	if ok {
		cancel()
	}
	return nil
}

func (d *InMemoryTaskDistributor) RescheduleTask(ctx context.Context, taskID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	assignment, ok := d.tasks[taskID]
	if !ok {
		return errors.New("task not found")
	}
	delete(d.pending, taskID)
	cctx, cancel := context.WithCancel(ctx)
	d.pending[taskID] = cancel
	go d.simulateExecution(cctx, assignment)
	return nil
}

func (d *InMemoryTaskDistributor) TaskResults(ctx context.Context, taskID string) ([]*TaskResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	results, ok := d.results[taskID]
	if !ok {
		return []*TaskResult{}, nil
	}
	return results, nil
}

func (d *InMemoryTaskDistributor) simulateExecution(ctx context.Context, assignment *TaskAssignment) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(100 * time.Millisecond):
		d.mu.Lock()
		defer d.mu.Unlock()
		d.results[assignment.TaskID] = append(d.results[assignment.TaskID], &TaskResult{
			TaskID:    assignment.TaskID,
			NodeID:    assignment.NodeID,
			Status:    "success",
			Output:    map[string]interface{}{"result": "ok"},
			Duration:  "100ms",
			FinishedAt: time.Now(),
		})
		delete(d.pending, assignment.TaskID)
	}
}

type InMemoryFailureDetector struct {
	mu       sync.RWMutex
	heartbeats map[string]time.Time
	failures  map[string]int
}

func NewInMemoryFailureDetector() *InMemoryFailureDetector {
	return &InMemoryFailureDetector{
		heartbeats: make(map[string]time.Time),
		failures:  make(map[string]int),
	}
}

func (d *InMemoryFailureDetector) RegisterHeartbeat(nodeID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.heartbeats[nodeID] = time.Now()
	return nil
}

func (d *InMemoryFailureDetector) RecordFailure(nodeID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.failures[nodeID]++
	return nil
}

func (d *InMemoryFailureDetector) DetectFailures(now time.Time) ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var failed []string
	for nodeID, lastSeen := range d.heartbeats {
		if now.Sub(lastSeen) > 30*time.Second {
			failed = append(failed, nodeID)
		}
	}
	return failed, nil
}

func (d *InMemoryFailureDetector) IsAlive(nodeID string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	lastSeen, ok := d.heartbeats[nodeID]
	if !ok {
		return false
	}
	return time.Since(lastSeen) < 30*time.Second
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
