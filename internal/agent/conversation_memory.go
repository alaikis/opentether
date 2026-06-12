package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

const (
	conversationWindowMessages = 10
	maxConversationSummaryLen  = 1200
	maxTaskMemoryCount         = 6
	maxTaskSummaryLen          = 500
	warmIdleAfter              = 30 * time.Minute
	archiveAfter               = 6 * time.Hour
	memoryArchiveDir           = "data/memory/conversations"
)

type ConversationMemoryContext struct {
	State          ConversationWorkingState
	RecentMessages []models.Message
	Route          TopicRouteResult
	ArchiveSnippet string
}

type ConversationWorkingState struct {
	Summary      string              `json:"summary"`
	ActiveTaskID string              `json:"active_task_id"`
	Status       string              `json:"status"`
	ArchivePath  string              `json:"archive_path"`
	LastActiveAt time.Time           `json:"last_active_at"`
	Entities     map[string]string   `json:"entities"`
	Tasks        []TaskWorkingMemory `json:"tasks"`
}

type TaskWorkingMemory struct {
	ID            string            `json:"id"`
	Topic         string            `json:"topic"`
	Status        string            `json:"status"`
	Entities      map[string]string `json:"entities"`
	LastIntent    string            `json:"last_intent"`
	LastMetric    string            `json:"last_metric"`
	LastTimeRange string            `json:"last_time_range"`
	Summary       string            `json:"summary"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type TopicRouteResult struct {
	Action     string              `json:"action"` // continue, switch, resume, fork, clarify
	FromTaskID string              `json:"from_task_id"`
	ToTaskID   string              `json:"to_task_id"`
	Reason     string              `json:"reason"`
	Candidates []TaskWorkingMemory `json:"candidates,omitempty"`
	Extracted  conversationFacts   `json:"extracted"`
}

func (m *MemoryManager) LoadConversationMemory(userID, conversationID, query string, limit int) (*ConversationMemoryContext, error) {
	if m == nil || m.db == nil || conversationID == "" {
		return &ConversationMemoryContext{State: newConversationWorkingState()}, nil
	}
	if limit <= 0 {
		limit = conversationWindowMessages
	}

	state, err := m.getConversationWorkingState(userID, conversationID)
	if err != nil {
		return nil, err
	}

	var recentDesc []models.Message
	if err := m.db.Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		Limit(limit).
		Find(&recentDesc).Error; err != nil {
		return nil, err
	}

	recent := make([]models.Message, 0, len(recentDesc))
	for i := len(recentDesc) - 1; i >= 0; i-- {
		recent = append(recent, recentDesc[i])
	}

	previousStatus := state.Status
	state = applyMemoryLifecycle(state)
	if previousStatus != state.Status {
		if state.Status == "archived" && state.ArchivePath == "" {
			if path, archiveErr := appendConversationArchive(conversationID, userID, state, "[system] 上下文长时间未活跃，自动封存。", ""); archiveErr == nil {
				state.ArchivePath = path
			}
		}
		updates := map[string]interface{}{"status": state.Status, "archive_path": state.ArchivePath}
		if state.Status == "archived" {
			now := time.Now()
			updates["archived_at"] = &now
		}
		_ = m.db.Model(&models.ConversationState{}).Where("conversation_id = ? AND user_id = ?", conversationID, userID).Updates(updates).Error
	}
	route := routeTopic(state, query)
	state = applyRouteToState(state, route)

	archiveSnippet := ""
	if route.Action == "resume" && state.ArchivePath != "" {
		archiveSnippet = readArchiveSnippet(state.ArchivePath, 1200)
	}

	return &ConversationMemoryContext{State: state, RecentMessages: recent, Route: route, ArchiveSnippet: archiveSnippet}, nil
}

func (m *MemoryManager) UpdateConversationMemory(user *UserContext, conversationID, userQuery, assistantReply string) error {
	return m.UpdateConversationMemoryWithSummary(user, conversationID, userQuery, assistantReply, "")
}

func (m *MemoryManager) UpdateConversationMemoryWithSummary(user *UserContext, conversationID, userQuery, assistantReply, compressedSummary string) error {
	if m == nil || m.db == nil || user == nil || conversationID == "" {
		return nil
	}

	state, err := m.getConversationWorkingState(user.UserID, conversationID)
	if err != nil {
		return err
	}

	state = applyMemoryLifecycle(state)
	route := routeTopic(state, userQuery)
	state = applyRouteToState(state, route)

	facts := extractConversationFacts(userQuery + "\n" + assistantReply)
	mergeEntities(state.Entities, facts.Entities)

	task := buildTaskFromFacts(facts, userQuery, assistantReply)
	state = upsertTask(state, task)
	state = applyRouteToState(state, route)
	if strings.TrimSpace(compressedSummary) != "" {
		state.Summary = truncateRunes(compressedSummary, maxConversationSummaryLen)
	} else {
		state.Summary = compressConversationSummary(state.Summary, userQuery, assistantReply)
	}
	if len([]rune(state.Summary)) > 600 || state.ArchivePath != "" {
		if path, archiveErr := appendConversationArchive(conversationID, user.UserID, state, userQuery, assistantReply); archiveErr == nil && path != "" {
			state.ArchivePath = path
		}
	}
	state.Tasks = compactTasks(state.Tasks, maxTaskMemoryCount)

	payloadEntities, _ := json.Marshal(state.Entities)
	payloadTasks, _ := json.Marshal(state.Tasks)

	var model models.ConversationState
	err = m.db.Where("conversation_id = ?", conversationID).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		model = models.ConversationState{
			ConversationID: conversationID,
			UserID:         user.UserID,
			Status:         "active",
			Summary:        state.Summary,
			ActiveTaskID:   state.ActiveTaskID,
			EntitiesJSON:   string(payloadEntities),
			TasksJSON:      string(payloadTasks),
			ArchivePath:    state.ArchivePath,
			LastActiveAt:   time.Now(),
			Version:        1,
		}
		return m.db.Create(&model).Error
	}
	if err != nil {
		return err
	}

	model.UserID = user.UserID
	model.Status = "active"
	model.Summary = state.Summary
	model.ActiveTaskID = state.ActiveTaskID
	model.EntitiesJSON = string(payloadEntities)
	model.TasksJSON = string(payloadTasks)
	model.ArchivePath = state.ArchivePath
	model.LastActiveAt = time.Now()
	model.ArchivedAt = nil
	model.Version++
	return m.db.Save(&model).Error
}

func (m *MemoryManager) getConversationWorkingState(userID, conversationID string) (ConversationWorkingState, error) {
	state := newConversationWorkingState()
	var model models.ConversationState
	err := m.db.Where("conversation_id = ?", conversationID).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	if model.UserID != "" && userID != "" && model.UserID != userID {
		return state, nil
	}

	state.Summary = model.Summary
	state.ActiveTaskID = model.ActiveTaskID
	state.Status = model.Status
	state.ArchivePath = model.ArchivePath
	state.LastActiveAt = model.LastActiveAt
	if model.EntitiesJSON != "" {
		_ = json.Unmarshal([]byte(model.EntitiesJSON), &state.Entities)
	}
	if model.TasksJSON != "" {
		_ = json.Unmarshal([]byte(model.TasksJSON), &state.Tasks)
	}
	if state.Entities == nil {
		state.Entities = map[string]string{}
	}
	return state, nil
}

func newConversationWorkingState() ConversationWorkingState {
	return ConversationWorkingState{Status: "active", Entities: map[string]string{}, Tasks: []TaskWorkingMemory{}, LastActiveAt: time.Now()}
}

func BuildConversationMemoryPrompt(ctx *ConversationMemoryContext, query string) string {
	if ctx == nil {
		return ""
	}
	var sb strings.Builder
	if ctx.Route.Action != "" {
		sb.WriteString("## 话题路由\n")
		sb.WriteString(fmt.Sprintf("- 动作: %s\n- 原任务: %s\n- 目标任务: %s\n- 原因: %s\n\n", ctx.Route.Action, ctx.Route.FromTaskID, ctx.Route.ToTaskID, ctx.Route.Reason))
	}

	if ctx.State.Summary != "" || len(ctx.State.Entities) > 0 || len(ctx.State.Tasks) > 0 {
		sb.WriteString("## 当前对话短期记忆（仅限本 conversation_id）\n")
		if ctx.State.Summary != "" {
			sb.WriteString("- 摘要: " + truncateRunes(ctx.State.Summary, 500) + "\n")
		}
		if len(ctx.State.Entities) > 0 {
			sb.WriteString("- 活跃实体: ")
			parts := make([]string, 0, len(ctx.State.Entities))
			for k, v := range ctx.State.Entities {
				parts = append(parts, fmt.Sprintf("%s=%s", k, v))
			}
			sb.WriteString(strings.Join(parts, ", ") + "\n")
		}
		if ctx.State.Status != "" {
			sb.WriteString("- 上下文状态: " + ctx.State.Status + "\n")
		}
		if ctx.State.ActiveTaskID != "" {
			sb.WriteString("- 当前任务: " + ctx.State.ActiveTaskID + "\n")
		}
		for _, task := range ctx.State.Tasks {
			if task.ID == ctx.State.ActiveTaskID || isTaskRelevant(task, query) {
				sb.WriteString(fmt.Sprintf("  - 任务[%s] topic=%s metric=%s time=%s summary=%s\n", task.ID, task.Topic, task.LastMetric, task.LastTimeRange, truncateRunes(task.Summary, 220)))
			}
		}
		sb.WriteString("\n")
	}

	if ctx.ArchiveSnippet != "" {
		sb.WriteString("## 召回的归档上下文片段（仅相关时使用）\n")
		sb.WriteString(ctx.ArchiveSnippet + "\n\n")
	}

	if len(ctx.RecentMessages) > 0 {
		sb.WriteString("## 最近对话窗口（按时间顺序，最多最近几轮）\n")
		for _, msg := range ctx.RecentMessages {
			role := "用户"
			if msg.Role == "assistant" {
				role = "助手"
			} else if msg.Role == "system" {
				role = "系统"
			}
			sb.WriteString(fmt.Sprintf("%s: %s\n", role, truncateRunes(msg.Content, 240)))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func applyMemoryLifecycle(state ConversationWorkingState) ConversationWorkingState {
	if state.Status == "" {
		state.Status = "active"
	}
	if state.LastActiveAt.IsZero() {
		return state
	}
	idle := time.Since(state.LastActiveAt)
	if idle >= archiveAfter {
		state.Status = "archived"
		for i := range state.Tasks {
			if state.Tasks[i].Status == "active" || state.Tasks[i].Status == "paused" {
				state.Tasks[i].Status = "archived"
			}
		}
	} else if idle >= warmIdleAfter {
		state.Status = "warm_idle"
		for i := range state.Tasks {
			if state.Tasks[i].Status == "active" {
				state.Tasks[i].Status = "paused"
			}
		}
	}
	return state
}

func routeTopic(state ConversationWorkingState, query string) TopicRouteResult {
	facts := extractConversationFacts(query)
	active := findTaskByID(state.Tasks, state.ActiveTaskID)
	result := TopicRouteResult{Action: "switch", FromTaskID: state.ActiveTaskID, Extracted: facts, Reason: "检测到新话题或新实体"}

	if strings.TrimSpace(query) == "" {
		result.Action = "continue"
		result.ToTaskID = state.ActiveTaskID
		result.Reason = "空输入，保持当前话题"
		return result
	}

	if hasMultipleTopics(query) {
		result.Action = "fork"
		result.Reason = "检测到多个话题并列出现"
		return result
	}

	if hasPronounReference(query) {
		if active != nil {
			result.Action = "continue"
			result.ToTaskID = active.ID
			result.Reason = "指代词命中当前活跃任务"
			return result
		}
		candidates := relevantTasks(state.Tasks, facts)
		if len(candidates) == 1 {
			result.Action = "resume"
			result.ToTaskID = candidates[0].ID
			result.Candidates = candidates
			result.Reason = "指代词命中唯一历史任务"
			return result
		}
		if len(candidates) > 1 {
			result.Action = "clarify"
			result.Candidates = candidates
			result.Reason = "指代词存在多个候选话题，需要澄清"
			return result
		}
	}

	if active != nil && isTaskRelevantToFacts(*active, facts) && !hasEntityConflict(*active, facts) {
		result.Action = "continue"
		result.ToTaskID = active.ID
		result.Reason = "当前输入与活跃任务匹配"
		return result
	}

	for _, task := range state.Tasks {
		if task.ID == state.ActiveTaskID {
			continue
		}
		if isTaskRelevantToFacts(task, facts) && !hasEntityConflict(task, facts) {
			result.Action = "resume"
			result.ToTaskID = task.ID
			result.Reason = "当前输入命中历史话题，恢复该任务"
			return result
		}
	}

	newTask := buildTaskFromFacts(facts, query, "")
	result.ToTaskID = newTask.ID
	return result
}

func applyRouteToState(state ConversationWorkingState, route TopicRouteResult) ConversationWorkingState {
	switch route.Action {
	case "continue":
		state.Status = "active"
		if route.ToTaskID != "" {
			state.ActiveTaskID = route.ToTaskID
		}
	case "resume", "switch", "fork":
		for i := range state.Tasks {
			if state.Tasks[i].ID == state.ActiveTaskID && state.Tasks[i].ID != route.ToTaskID {
				state.Tasks[i].Status = "paused"
			}
			if state.Tasks[i].ID == route.ToTaskID {
				state.Tasks[i].Status = "active"
			}
		}
		if route.ToTaskID != "" {
			state.ActiveTaskID = route.ToTaskID
		}
		state.Status = "active"
	case "clarify":
		state.Status = "active"
	}
	return state
}

func findTaskByID(tasks []TaskWorkingMemory, id string) *TaskWorkingMemory {
	for i := range tasks {
		if tasks[i].ID == id {
			return &tasks[i]
		}
	}
	return nil
}

func relevantTasks(tasks []TaskWorkingMemory, facts conversationFacts) []TaskWorkingMemory {
	var result []TaskWorkingMemory
	for _, task := range tasks {
		if isTaskRelevantToFacts(task, facts) {
			result = append(result, task)
		}
	}
	return result
}

func hasEntityConflict(task TaskWorkingMemory, facts conversationFacts) bool {
	for k, v := range facts.Entities {
		if v != "" && task.Entities[k] != "" && task.Entities[k] != v {
			return true
		}
	}
	return false
}

func hasMultipleTopics(query string) bool {
	separators := []string{"另外", "顺便", "同时", "还有", "并且", "以及", "，另外", ";", "；"}
	for _, sep := range separators {
		if strings.Contains(query, sep) {
			return true
		}
	}
	return false
}

func hasPronounReference(query string) bool {
	for _, token := range []string{"他", "她", "它", "该员工", "这个人", "此人", "刚才", "上面", "这个", "该客户", "该产品"} {
		if strings.Contains(query, token) {
			return true
		}
	}
	return false
}

type conversationFacts struct {
	Entities  map[string]string
	Topic     string
	Metric    string
	TimeRange string
	Intent    string
}

func extractConversationFacts(text string) conversationFacts {
	facts := conversationFacts{Entities: map[string]string{}}
	if employee := extractEmployeeName(text); employee != "" {
		facts.Entities["employee"] = employee
	}
	facts.Topic = detectTopic(text)
	facts.Metric = detectMetric(text)
	facts.TimeRange = detectTimeRange(text)
	facts.Intent = detectIntent(text)
	return facts
}

func extractEmployeeName(text string) string {
	patterns := []string{
		`已找到员工\s*([\p{Han}A-Za-z0-9_·]{2,20})`,
		`员工\s*[:：]?\s*([\p{Han}A-Za-z0-9_·]{2,20})`,
		`([\p{Han}]{2,4})(?:当前|现在|本月|上月|上个季度|上季度|这个季度|的)?(?:卖了|卖|出了|出|订单|销售额|业绩)`,
	}
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		if m := re.FindStringSubmatch(text); len(m) > 1 {
			return strings.Trim(m[1], " ，。:：-\n\t")
		}
	}
	trimmed := strings.TrimSpace(text)
	if utf8.RuneCountInString(trimmed) >= 2 && utf8.RuneCountInString(trimmed) <= 4 && regexp.MustCompile(`^[\p{Han}]+$`).MatchString(trimmed) {
		return trimmed
	}
	return ""
}

func detectTopic(text string) string {
	switch {
	case strings.Contains(text, "订单") || strings.Contains(text, "出单") || strings.Contains(text, "销售") || strings.Contains(text, "卖"):
		return "sales"
	case strings.Contains(text, "库存") || strings.Contains(text, "仓库"):
		return "inventory"
	case strings.Contains(text, "员工") || strings.Contains(text, "部门") || strings.Contains(text, "职位"):
		return "employee"
	case strings.Contains(text, "客户"):
		return "customer"
	default:
		return "general"
	}
}

func detectMetric(text string) string {
	switch {
	case strings.Contains(text, "销售额") || strings.Contains(text, "金额"):
		return "销售额"
	case strings.Contains(text, "订单") || strings.Contains(text, "出单") || strings.Contains(text, "多少单"):
		return "订单数"
	case strings.Contains(text, "库存"):
		return "库存"
	default:
		return ""
	}
}

func detectTimeRange(text string) string {
	for _, token := range []string{"上个季度", "上季度", "本季度", "这个季度", "上个月", "上月", "本月", "当前", "现在", "今天", "昨天", "今年", "去年"} {
		if strings.Contains(text, token) {
			return token
		}
	}
	return ""
}

func detectIntent(text string) string {
	if strings.Contains(text, "多少") || strings.Contains(text, "查询") || strings.Contains(text, "销售额") || strings.Contains(text, "订单") || strings.Contains(text, "卖") || strings.Contains(text, "出") {
		return "query"
	}
	return "chat"
}

func mergeEntities(dst, src map[string]string) {
	if dst == nil {
		return
	}
	for k, v := range src {
		if strings.TrimSpace(v) != "" {
			dst[k] = v
		}
	}
}

func buildTaskFromFacts(facts conversationFacts, query, reply string) TaskWorkingMemory {
	topic := facts.Topic
	if topic == "" {
		topic = "general"
	}
	idParts := []string{topic}
	if employee := facts.Entities["employee"]; employee != "" {
		idParts = append(idParts, sanitizeTaskID(employee))
	}
	id := strings.Join(idParts, ":")
	return TaskWorkingMemory{
		ID:            id,
		Topic:         topic,
		Status:        "active",
		Entities:      facts.Entities,
		LastIntent:    facts.Intent,
		LastMetric:    facts.Metric,
		LastTimeRange: facts.TimeRange,
		Summary:       truncateRunes(fmt.Sprintf("用户: %s；助手: %s", query, reply), maxTaskSummaryLen),
		UpdatedAt:     time.Now(),
	}
}

func upsertTask(state ConversationWorkingState, task TaskWorkingMemory) ConversationWorkingState {
	if state.Entities == nil {
		state.Entities = map[string]string{}
	}
	if task.ID == "" {
		return state
	}
	updated := false
	for i := range state.Tasks {
		if state.Tasks[i].ID == task.ID {
			mergeEntities(state.Tasks[i].Entities, task.Entities)
			if task.LastIntent != "" {
				state.Tasks[i].LastIntent = task.LastIntent
			}
			if task.LastMetric != "" {
				state.Tasks[i].LastMetric = task.LastMetric
			}
			if task.LastTimeRange != "" {
				state.Tasks[i].LastTimeRange = task.LastTimeRange
			}
			state.Tasks[i].Summary = task.Summary
			state.Tasks[i].Status = "active"
			state.Tasks[i].UpdatedAt = task.UpdatedAt
			updated = true
			break
		}
	}
	if !updated {
		state.Tasks = append(state.Tasks, task)
	}
	state.ActiveTaskID = task.ID
	return state
}

func focusConversationState(state ConversationWorkingState, query string) ConversationWorkingState {
	if len(state.Tasks) == 0 || query == "" {
		return state
	}
	facts := extractConversationFacts(query)
	for _, task := range state.Tasks {
		if isTaskRelevantToFacts(task, facts) {
			state.ActiveTaskID = task.ID
			mergeEntities(state.Entities, task.Entities)
			return state
		}
	}
	if hasPronounReference(query) && state.ActiveTaskID != "" {
		for _, task := range state.Tasks {
			if task.ID == state.ActiveTaskID {
				mergeEntities(state.Entities, task.Entities)
				return state
			}
		}
	}
	return state
}

func isTaskRelevant(task TaskWorkingMemory, query string) bool {
	return isTaskRelevantToFacts(task, extractConversationFacts(query))
}

func isTaskRelevantToFacts(task TaskWorkingMemory, facts conversationFacts) bool {
	if facts.Topic != "" && facts.Topic != "general" && facts.Topic == task.Topic {
		if employee := facts.Entities["employee"]; employee == "" || employee == task.Entities["employee"] {
			return true
		}
	}
	for k, v := range facts.Entities {
		if v != "" && task.Entities[k] == v {
			return true
		}
	}
	return false
}

func compactTasks(tasks []TaskWorkingMemory, limit int) []TaskWorkingMemory {
	if limit <= 0 || len(tasks) <= limit {
		return tasks
	}
	for i := 0; i < len(tasks)-1; i++ {
		for j := i + 1; j < len(tasks); j++ {
			if tasks[j].UpdatedAt.After(tasks[i].UpdatedAt) {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}
	return tasks[:limit]
}

func compressConversationSummary(oldSummary, query, reply string) string {
	line := fmt.Sprintf("- %s | %s", truncateRunes(query, 120), truncateRunes(reply, 180))
	if oldSummary == "" {
		return line
	}
	combined := oldSummary + "\n" + line
	return truncateFromEnd(combined, maxConversationSummaryLen)
}

func truncateRunes(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if max <= 0 || len(r) <= max {
		return strings.TrimSpace(s)
	}
	return string(r[:max]) + "..."
}

func truncateFromEnd(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if max <= 0 || len(r) <= max {
		return strings.TrimSpace(s)
	}
	return "..." + string(r[len(r)-max:])
}

func sanitizeTaskID(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, ":", "_")
	return s
}

func appendConversationArchive(conversationID, userID string, state ConversationWorkingState, userQuery, assistantReply string) (string, error) {
	if conversationID == "" {
		return "", nil
	}
	if err := os.MkdirAll(memoryArchiveDir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(memoryArchiveDir, sanitizeTaskID(conversationID)+".md")
	entry := formatConversationArchiveEntry(userID, state, userQuery, assistantReply)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(entry); err != nil {
		return "", err
	}
	return path, nil
}

func formatConversationArchiveEntry(userID string, state ConversationWorkingState, userQuery, assistantReply string) string {
	entitiesJSON, _ := json.Marshal(state.Entities)
	tasksJSON, _ := json.Marshal(state.Tasks)
	return fmt.Sprintf("\n\n---\n\n## Memory Turn %s\n\n- UserID: %s\n- Status: %s\n- ActiveTask: %s\n- Entities: %s\n\n### Task Index\n\n```json\n%s\n```\n\n### Summary Snapshot\n\n%s\n\n### User\n\n%s\n\n### Assistant\n\n%s\n", time.Now().Format(time.RFC3339), userID, state.Status, state.ActiveTaskID, string(entitiesJSON), string(tasksJSON), truncateRunes(state.Summary, 800), truncateRunes(userQuery, 600), truncateRunes(assistantReply, 1000))
}

func readArchiveSnippet(path string, max int) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	runes := []rune(string(data))
	if max <= 0 || len(runes) <= max {
		return string(data)
	}
	return string(runes[len(runes)-max:])
}
