package service

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/alaikis/opentether/internal/llm"
	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

type AgentTaskService struct {
	db     *gorm.DB
	agent  *AgentService
	cancel map[string]chan struct{}
}

func NewAgentTaskService(db *gorm.DB) *AgentTaskService {
	return &AgentTaskService{db: db, cancel: make(map[string]chan struct{})}
}
func (s *AgentTaskService) SetAgentService(agent *AgentService) { s.agent = agent }

func (s *AgentTaskService) CancelGraph(graphID string) error {
	s.db.Model(&models.AgentTaskGraph{}).Where("id = ?", graphID).Update("status", "cancelled")
	return nil
}

func (s *AgentTaskService) CreateGraph(userID, conversationID, goal string) (*models.AgentTaskGraph, error) {
	goal = stripEnvironmentDetails(goal)
	graph := &models.AgentTaskGraph{UserID: userID, ConversationID: conversationID, Goal: goal, Status: "pending"}
	plan := s.planWithLLM(context.Background(), goal)
	if len(plan) == 0 {
		plan = s.defaultPlan(goal)
	}
	b, _ := json.Marshal(plan)
	graph.PlanJSON = string(b)
	if err := s.db.Create(graph).Error; err != nil {
		return nil, err
	}
	for _, node := range plan {
		n := models.AgentTaskNode{GraphID: graph.ID, Type: node.Type, Name: node.Name, InputJSON: node.InputJSON, DependsOnJSON: node.DependsOnJSON, Status: "pending"}
		_ = s.db.Create(&n).Error
	}
	go s.RunGraph(graph.ID)
	return graph, nil
}

type plannedNode struct {
	Type          string   `json:"type"`
	Name          string   `json:"name"`
	Query         string   `json:"query,omitempty"`
	InputJSON     string   `json:"input_json"`
	DependsOn     []string `json:"depends_on,omitempty"`
	DependsOnJSON string   `json:"depends_on_json"`
}

func (s *AgentTaskService) planWithLLM(ctx context.Context, goal string) []plannedNode {
	var provider models.Provider
	if err := s.db.Where("enabled = ?", true).Order("priority ASC").First(&provider).Error; err != nil {
		return nil
	}
	client, err := llm.NewClient(&provider)
	if err != nil {
		return nil
	}
	prompt := `你是任务规划器。把用户目标拆成可执行 DAG 节点。只返回 JSON 数组，不要解释。
每个节点字段: name,type,query,depends_on。
type 只能是 agent,text2sql,report,summary,notify。
原则: 数据查询先于图表/报告/通知；summary 通常依赖全部前置节点。
用户目标:` + goal
	resp, err := llm.ChatCompletionWithRetry(client, ctx, llm.ChatRequest{Model: provider.Model, Messages: []llm.Message{{Role: "user", Content: prompt}}, MaxTokens: 2048, Temperature: 0.1}, 2)
	if err != nil || resp == nil || strings.TrimSpace(resp.Content) == "" {
		return nil
	}
	text := strings.TrimSpace(resp.Content)
	if i := strings.Index(text, "["); i >= 0 {
		if j := strings.LastIndex(text, "]"); j >= i {
			text = text[i : j+1]
		}
	}
	var nodes []plannedNode
	if json.Unmarshal([]byte(text), &nodes) != nil || len(nodes) == 0 {
		return nil
	}
	for i := range nodes {
		if nodes[i].Type == "" {
			nodes[i].Type = "agent"
		}
		if nodes[i].Name == "" {
			nodes[i].Name = shortTaskName(nodes[i].Query)
		}
		input := map[string]string{"query": nodes[i].Query}
		if input["query"] == "" {
			input["query"] = goal
		}
		b, _ := json.Marshal(input)
		d, _ := json.Marshal(nodes[i].DependsOn)
		nodes[i].InputJSON = string(b)
		nodes[i].DependsOnJSON = string(d)
	}
	return nodes
}

func (s *AgentTaskService) defaultPlan(goal string) []plannedNode {
	goal = stripEnvironmentDetails(goal)
	parts := splitGoalIntoTasks(goal)
	plan := make([]plannedNode, 0, len(parts)+1)
	for _, part := range parts {
		input := map[string]string{"query": part}
		b, _ := json.Marshal(input)
		plan = append(plan, plannedNode{Type: "agent", Name: shortTaskName(part), InputJSON: string(b)})
	}
	input := map[string]string{"query": goal}
	b, _ := json.Marshal(input)
	plan = append(plan, plannedNode{Type: "summary", Name: "汇总结果", InputJSON: string(b), DependsOnJSON: "[]"})
	return plan
}

func splitGoalIntoTasks(goal string) []string {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return []string{""}
	}
	seps := []string{"？", "?", "；", ";", "然后", "并且"}
	parts := []string{goal}
	for _, sep := range seps {
		var next []string
		for _, part := range parts {
			next = append(next, strings.Split(part, sep)...)
		}
		parts = next
	}
	out := []string{}
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{goal}
	}
	return out
}

func shortTaskName(text string) string {
	r := []rune(strings.TrimSpace(text))
	if len(r) > 16 {
		return string(r[:16]) + "..."
	}
	return string(r)
}

func (s *AgentTaskService) RunGraph(graphID string) {
	var graph models.AgentTaskGraph
	if err := s.db.First(&graph, "id = ?", graphID).Error; err != nil {
		return
	}
	s.db.Model(&graph).Updates(map[string]interface{}{"status": "running"})
	var nodes []models.AgentTaskNode
	s.db.Where("graph_id = ?", graphID).Order("created_at ASC").Find(&nodes)
	completed := map[string]bool{}
	executionGroups := map[string][]*models.AgentTaskNode{}
	for i := range nodes {
		if nodes[i].ExecutionGroup != "" {
			executionGroups[nodes[i].ExecutionGroup] = append(executionGroups[nodes[i].ExecutionGroup], &nodes[i])
		}
	}
	for {
		progress := false
		var parallelNodes []*models.AgentTaskNode
		for i := range nodes {
			node := &nodes[i]
			if node.Status == "completed" || node.Status == "skipped" {
				completed[node.Name] = true
				continue
			}
			if node.Status == "running" || node.Status == "review" {
				continue
			}
			if !depsCompleted(node.DependsOnJSON, completed) {
				continue
			}
			parallelNodes = append(parallelNodes, node)
		}
		if len(parallelNodes) == 0 {
			break
		}
		progress = true
		var wg sync.WaitGroup
		for _, node := range parallelNodes {
			if node.SubgraphID != "" {
				wg.Add(1)
				go func(n *models.AgentTaskNode) {
					defer wg.Done()
					if s.executeSubgraph(&graph, n, completed) {
						completed[n.Name] = true
					}
				}(node)
				continue
			}
			if !s.evaluateCondition(node, &graph, completed) {
				s.db.Model(node).Updates(map[string]interface{}{"status": "skipped", "summary": "条件不满足，已跳过"})
				completed[node.Name] = true
				continue
			}
			wg.Add(1)
			go func(n *models.AgentTaskNode) {
				defer wg.Done()
				s.executeNodeWithRetry(&graph, n, &completed)
				if n.Status == "failed" {
					var retryCfg map[string]interface{}
					_ = json.Unmarshal([]byte(n.RetryConfigJSON), &retryCfg)
					maxRetries := 3
					if v, ok := retryCfg["max_retries"].(float64); ok {
						maxRetries = int(v)
					}
					if n.RetryCount < maxRetries {
						backoff := time.Duration(n.RetryCount+1) * 2 * time.Second
						time.Sleep(backoff)
						n.RetryCount++
						n.Status = "pending"
						s.db.Model(n).Updates(map[string]interface{}{"status": "pending", "retry_count": n.RetryCount, "error": ""})
						completed[n.Name] = false
						return
					}
					s.db.Model(&graph).Updates(map[string]interface{}{"status": "failed", "error": n.Error})
				}
				completed[n.Name] = true
			}(node)
		}
		wg.Wait()
		if !progress {
			break
		}
	}
	var pending int64
	s.db.Model(&models.AgentTaskNode{}).Where("graph_id = ? AND status IN ?", graph.ID, []string{"pending", "running"}).Count(&pending)
	if pending > 0 {
		s.db.Model(&graph).Updates(map[string]interface{}{"status": "failed", "error": "存在未满足依赖的节点"})
		return
	}
	s.db.Model(&graph).Updates(map[string]interface{}{"status": "completed", "summary": s.collectSummary(graph.ID)})
	s.notifyWebhook(&graph)
}

func (s *AgentTaskService) evaluateCondition(node *models.AgentTaskNode, graph *models.AgentTaskGraph, completed map[string]bool) bool {
	if strings.TrimSpace(node.ConditionJSON) == "" {
		return true
	}
	var cond map[string]interface{}
	if json.Unmarshal([]byte(node.ConditionJSON), &cond) != nil {
		return true
	}
	if dep, ok := cond["depends_on"].(string); ok && dep != "" && !completed[dep] {
		return false
	}
	_ = cond
	return true
}

func (s *AgentTaskService) executeSubgraph(graph *models.AgentTaskGraph, node *models.AgentTaskNode, completed map[string]bool) bool {
	var subNodes []models.AgentTaskNode
	s.db.Where("graph_id = ?", node.SubgraphID).Order("created_at ASC").Find(&subNodes)
	subCompleted := map[string]bool{}
	for _, sn := range subNodes {
		sn := sn
		now := time.Now()
		s.db.Model(&sn).Updates(map[string]interface{}{"status": "running", "started_at": &now})
		summary, _, errText := s.executeNode(graph, &sn)
		finish := time.Now()
		if errText != "" {
			s.db.Model(&sn).Updates(map[string]interface{}{"status": "failed", "error": errText, "finished_at": &finish})
			s.db.Model(node).Updates(map[string]interface{}{"status": "failed", "summary": summary, "error": errText, "finished_at": &finish})
			return false
		}
		s.db.Model(&sn).Updates(map[string]interface{}{"status": "completed", "summary": summary, "finished_at": &finish})
		subCompleted[sn.Name] = true
	}
	s.db.Model(node).Updates(map[string]interface{}{"status": "completed", "summary": s.collectSummary(node.SubgraphID)})
	return true
}

func (s *AgentTaskService) executeNodeWithRetry(graph *models.AgentTaskGraph, node *models.AgentTaskNode, completed *map[string]bool) {
	checkpoint, _ := json.Marshal(map[string]interface{}{"completed": *completed, "node_status": node.Status, "graph_status": graph.Status})
	now := time.Now()
	s.db.Model(node).Updates(map[string]interface{}{"status": "running", "started_at": &now, "checkpoint_json": string(checkpoint)})
	summary, output, errText := s.executeNode(graph, node)
	finish := time.Now()
	status := "completed"
	if errText != "" {
		status = "failed"
	}
	s.db.Model(node).Updates(map[string]interface{}{"status": status, "summary": summary, "error": errText, "finished_at": &finish, "checkpoint_json": string(checkpoint)})
	_ = s.db.Create(&models.AgentTaskOutput{NodeID: node.ID, Type: node.Type, ContentJSON: output, Summary: summary}).Error
}

func (s *AgentTaskService) InsertNode(graphID string, node *models.AgentTaskNode) (*models.AgentTaskNode, error) {
	node.GraphID = graphID
	node.Status = "pending"
	if err := s.db.Create(node).Error; err != nil {
		return nil, err
	}
	return node, nil
}

func (s *AgentTaskService) ReviewNode(nodeID string, approved bool, comment string) (*models.AgentTaskNode, error) {
	var node models.AgentTaskNode
	if err := s.db.First(&node, "id = ?", nodeID).Error; err != nil {
		return nil, err
	}
	if approved {
		s.db.Model(&node).Updates(map[string]interface{}{"review_status": "approved", "review_comment": comment, "status": "pending"})
		go s.RunGraph(node.GraphID)
	} else {
		s.db.Model(&node).Updates(map[string]interface{}{"review_status": "rejected", "review_comment": comment, "status": "review"})
	}
	return &node, nil
}

func (s *AgentTaskService) ResumeFromCheckpoint(graphID, nodeID string) error {
	var node models.AgentTaskNode
	if err := s.db.First(&node, "id = ?", nodeID).Error; err != nil {
		return err
	}
	var checkpoint map[string]interface{}
	if json.Unmarshal([]byte(node.CheckpointJSON), &checkpoint) == nil {
		completed, _ := checkpoint["completed"].(map[string]interface{})
		_ = completed
	}
	s.db.Model(&node).Updates(map[string]interface{}{"status": "pending", "error": "", "checkedpoint_json": node.CheckpointJSON})
	go s.RunGraph(graphID)
	return nil
}

func (s *AgentTaskService) GetGraphVisualization(graphID string) (map[string]interface{}, error) {
	graph, nodes, outputs, err := s.GetGraph(graphID)
	if err != nil {
		return nil, err
	}
	_ = outputs
	vis := map[string]interface{}{"graph": graph, "dags": map[string]interface{}{"nodes": []map[string]interface{}{}, "edges": []map[string]interface{}{}}}
	var nodeList []map[string]interface{}
	var edgeList []map[string]interface{}
	for _, n := range nodes {
		nodeList = append(nodeList, map[string]interface{}{"id": n.ID, "name": n.Name, "type": n.Type, "status": n.Status, "execution_group": n.ExecutionGroup, "subgraph_id": n.SubgraphID, "retry_count": n.RetryCount})
		var deps []string
		_ = json.Unmarshal([]byte(n.DependsOnJSON), &deps)
		for _, dep := range deps {
			for _, dn := range nodes {
				if dn.Name == dep {
					edgeList = append(edgeList, map[string]interface{}{"from": dn.ID, "to": n.ID, "label": dep})
				}
			}
		}
	}
	vis["dags"].(map[string]interface{})["nodes"] = nodeList
	vis["dags"].(map[string]interface{})["edges"] = edgeList
	return vis, nil
}

func depsCompleted(raw string, completed map[string]bool) bool {
	var deps []string
	_ = json.Unmarshal([]byte(raw), &deps)
	for _, dep := range deps {
		if dep != "" && !completed[dep] {
			return false
		}
	}
	return true
}

func (s *AgentTaskService) executeNode(graph *models.AgentTaskGraph, node *models.AgentTaskNode) (string, string, string) {
	if node.Type == "summary" {
		return s.collectSummary(graph.ID), "{}", ""
	}
	if s.agent == nil {
		return "", "{}", "agent service not configured"
	}
	var input map[string]string
	_ = json.Unmarshal([]byte(node.InputJSON), &input)
	query := input["query"]
	resp, err := s.agent.Chat(graph.UserID, query, graph.ConversationID, "")
	if err != nil {
		return "", "{}", err.Error()
	}
	b, _ := json.Marshal(resp)
	msg, _ := resp["message"].(string)
	if len(msg) > 1000 {
		msg = msg[:1000] + "..."
	}
	return msg, string(b), ""
}

func (s *AgentTaskService) notifyWebhook(graph *models.AgentTaskGraph) {
	if s.db == nil {
		return
	}
	var svc WebhookService
	svc.db = s.db
	svc.Deliver("task_graph."+graph.Status, map[string]interface{}{"graph_id": graph.ID, "status": graph.Status, "goal": graph.Goal, "summary": graph.Summary, "error": graph.Error})
}

func (s *AgentTaskService) collectSummary(graphID string) string {
	var outputs []models.AgentTaskOutput
	s.db.Where("node_id IN (?)", s.db.Model(&models.AgentTaskNode{}).Select("id").Where("graph_id = ?", graphID)).Order("created_at ASC").Find(&outputs)
	parts := []string{}
	for _, out := range outputs {
		if strings.TrimSpace(out.Summary) != "" {
			parts = append(parts, out.Summary)
		}
	}
	return strings.Join(parts, "\n\n")
}

func (s *AgentTaskService) RetryNode(nodeID string) (*models.AgentTaskNode, error) {
	var node models.AgentTaskNode
	if err := s.db.First(&node, "id = ?", nodeID).Error; err != nil {
		return nil, err
	}
	s.db.Model(&node).Updates(map[string]interface{}{"status": "pending", "error": "", "summary": "", "started_at": nil, "finished_at": nil})
	go s.RunGraph(node.GraphID)
	return &node, nil
}

func (s *AgentTaskService) SkipNode(nodeID string) (*models.AgentTaskNode, error) {
	var node models.AgentTaskNode
	if err := s.db.First(&node, "id = ?", nodeID).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	s.db.Model(&node).Updates(map[string]interface{}{"status": "skipped", "summary": "已跳过", "finished_at": &now})
	go s.RunGraph(node.GraphID)
	return &node, nil
}

func (s *AgentTaskService) ResumeGraph(graphID string) error {
	return s.db.Model(&models.AgentTaskGraph{}).Where("id = ?", graphID).Update("status", "pending").Error
}

func (s *AgentTaskService) GetGraph(id string) (*models.AgentTaskGraph, []models.AgentTaskNode, []models.AgentTaskOutput, error) {
	var graph models.AgentTaskGraph
	if err := s.db.First(&graph, "id = ?", id).Error; err != nil {
		return nil, nil, nil, err
	}
	var nodes []models.AgentTaskNode
	var outputs []models.AgentTaskOutput
	s.db.Where("graph_id = ?", id).Order("created_at ASC").Find(&nodes)
	ids := []string{}
	for _, n := range nodes {
		ids = append(ids, n.ID)
	}
	if len(ids) > 0 {
		s.db.Where("node_id IN ?", ids).Order("created_at ASC").Find(&outputs)
	}
	return &graph, nodes, outputs, nil
}

func (s *AgentTaskService) GetNodeHistory(graphID string) ([]models.AgentTaskNode, error) {
	var nodes []models.AgentTaskNode
	err := s.db.Where("graph_id = ?", graphID).Order("created_at ASC").Find(&nodes).Error
	if err != nil {
		return nil, err
	}
	return nodes, nil
}
