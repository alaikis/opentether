package agent

import (
	"encoding/json"
	"errors"
	"sort"
	"sync"
)

type DAGNode struct {
	ID         string
	Task       SubTask
	Children   []*DAGNode
	Dependents []*DAGNode
}

type TaskGraph struct {
	mu      sync.RWMutex
	nodes   map[string]*DAGNode
	entry   *DAGNode
	version int
}

func NewTaskGraph(root *SubTask) *TaskGraph {
	g := &TaskGraph{
		nodes: make(map[string]*DAGNode),
	}
	if root != nil {
		g.entry = g.buildDAG(nil, *root)
	}
	return g
}

func (g *TaskGraph) buildDAG(parent *DAGNode, task SubTask) *DAGNode {
	node := &DAGNode{
		ID:       taskLabelToID(task.Label),
		Task:     task,
		Children: make([]*DAGNode, 0, len(task.Children)),
	}
	if _, exists := g.nodes[node.ID]; !exists {
		g.nodes[node.ID] = node
	}
	if parent != nil {
		node.Dependents = append(node.Dependents, parent)
		parent.Children = append(parent.Children, node)
	}
	for _, child := range task.Children {
		g.buildDAG(node, *child)
	}
	return node
}

func taskLabelToID(label string) string {
	return label
}

func (g *TaskGraph) AddTask(task SubTask, deps []string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	id := taskLabelToID(task.Label)
	if _, exists := g.nodes[id]; exists {
		return errors.New("task already exists: " + id)
	}

	node := &DAGNode{
		ID:         id,
		Task:       task,
		Children:   make([]*DAGNode, 0),
		Dependents: make([]*DAGNode, 0),
	}
	g.nodes[id] = node

	for _, depID := range deps {
		dep, exists := g.nodes[depID]
		if !exists {
			continue
		}
		node.Dependents = append(node.Dependents, dep)
		dep.Children = append(dep.Children, node)
	}

	if g.entry == nil {
		g.entry = node
	}

	g.version++
	return nil
}

func (g *TaskGraph) UpdateStatus(id, status string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	node, exists := g.nodes[id]
	if !exists {
		return
	}
	node.Task.Status = status
	g.version++
}

func (g *TaskGraph) TopologicalSort() []SubTask {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	var result []SubTask

	var visit func(node *DAGNode)
	visit = func(node *DAGNode) {
		if node == nil || visited[node.ID] {
			return
		}
		visited[node.ID] = true
		for _, child := range node.Children {
			visit(child)
		}
		result = append(result, node.Task)
	}

	if g.entry != nil {
		visit(g.entry)
	}

	// Ensure all nodes are visited
	for _, node := range g.nodes {
		if !visited[node.ID] {
			visit(node)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Index < result[j].Index
	})
	return result
}

func (g *TaskGraph) Dependencies(id string) ([]SubTask, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	node, exists := g.nodes[id]
	if !exists {
		return nil, errors.New("task not found: " + id)
	}

	deps := make([]SubTask, 0, len(node.Dependents))
	for _, dep := range node.Dependents {
		deps = append(deps, dep.Task)
	}
	return deps, nil
}

func (g *TaskGraph) Snapshot() *TaskGraphSnapshot {
	g.mu.RLock()
	defer g.mu.RUnlock()

	snapshot := &TaskGraphSnapshot{
		Version: g.version,
		Nodes:   make([]SubTask, 0, len(g.nodes)),
	}
	for _, node := range g.nodes {
		snapshot.Nodes = append(snapshot.Nodes, node.Task)
	}
	return snapshot
}

type TaskGraphSnapshot struct {
	Version int       `json:"version"`
	Nodes   []SubTask `json:"nodes"`
}

func (g *TaskGraph) MarshalJSON() ([]byte, error) {
	return json.Marshal(g.Snapshot())
}
