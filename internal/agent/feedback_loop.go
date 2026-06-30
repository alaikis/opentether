package agent

import (
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	ObservationTypeQueryResult   = "query_result"
	ObservationTypeSkillUsage    = "skill_usage"
	ObservationTypeLLMLatency    = "llm_latency"
	ObservationTypeUserFeedback  = "user_feedback"
	ObservationTypeImplicitSignal = "implicit_signal"

	ActionTypeGenerateHint     = "generate_improvement_hint"
	ActionTypeUpdateSoul       = "update_soul"
	ActionTypeUpdatePromptVer  = "update_prompt_version"
	ActionTypeAlert            = "alert"
	ActionTypeRecordMetric     = "record_metric"
)

type FeedbackLoop struct {
	mu           sync.RWMutex
	collectCh    chan *Observation
	processCh    chan []*Observation
	stopCh       chan struct{}

	batchSize    int
	batchTimeout time.Duration
	actions      map[string]ActionHandler

	processedCount int64
	errorCount     int64
}

type Observation struct {
	ID        string
	Type      string
	Payload   map[string]interface{}
	Timestamp time.Time
	UserID    string
	TraceID   string
}

type Insight struct {
	ID         string
	ObsIDs     []string
	Category   string
	Confidence float64
	Content    string
	CreatedAt  time.Time
}

type UpgradeAction struct {
	ID         string
	Type       string
	Payload    map[string]interface{}
	Status     string
	CreatedAt  time.Time
	ExecutedAt *time.Time
}

type ActionHandler func(action *UpgradeAction) error

func NewFeedbackLoop(batchSize int, batchTimeout time.Duration) *FeedbackLoop {
	loop := &FeedbackLoop{
		collectCh:    make(chan *Observation, 1000),
		processCh:    make(chan []*Observation, 10),
		stopCh:       make(chan struct{}),
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		actions:      make(map[string]ActionHandler),
	}

	loop.registerDefaultActions()
	return loop
}

func (f *FeedbackLoop) registerDefaultActions() {
	f.RegisterAction(ActionTypeGenerateHint, func(action *UpgradeAction) error {
		log.Printf("[FeedbackLoop] Generating improvement hint: %s", action.Payload)
		return nil
	})

	f.RegisterAction(ActionTypeUpdateSoul, func(action *UpgradeAction) error {
		log.Printf("[FeedbackLoop] Updating soul: %s", action.Payload)
		return nil
	})

	f.RegisterAction(ActionTypeUpdatePromptVer, func(action *UpgradeAction) error {
		log.Printf("[FeedbackLoop] Updating prompt version: %s", action.Payload)
		return nil
	})

	f.RegisterAction(ActionTypeAlert, func(action *UpgradeAction) error {
		log.Printf("[FeedbackLoop] Alert: %s", action.Payload)
		return nil
	})

	f.RegisterAction(ActionTypeRecordMetric, func(action *UpgradeAction) error {
		metricType, _ := action.Payload["metric_type"].(string)
		value, _ := action.Payload["value"].(float64)
		log.Printf("[FeedbackLoop] Metric recorded: %s = %.2f", metricType, value)
		return nil
	})
}

func (f *FeedbackLoop) RegisterAction(actionType string, handler ActionHandler) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.actions[actionType] = handler
}

func (f *FeedbackLoop) Start() {
	go f.batchCollector()
	go f.processor()
	log.Printf("[FeedbackLoop] Started with batch size %d, timeout %v",
		f.batchSize, f.batchTimeout)
}

func (f *FeedbackLoop) Stop() {
	close(f.stopCh)
	log.Printf("[FeedbackLoop] Stopped. Processed: %d, Errors: %d",
		f.processedCount, f.errorCount)
}

func (f *FeedbackLoop) Collect(obs *Observation) bool {
	select {
	case f.collectCh <- obs:
		return true
	default:
		log.Printf("[FeedbackLoop] Channel full, dropping observation")
		return false
	}
}

func (f *FeedbackLoop) CollectAsync(obs *Observation) {
	go func() {
		f.Collect(obs)
	}()
}

func (f *FeedbackLoop) batchCollector() {
	ticker := time.NewTicker(f.batchTimeout)
	defer ticker.Stop()

	batch := make([]*Observation, 0, f.batchSize)

	for {
		select {
		case <-f.stopCh:
			if len(batch) > 0 {
				f.processCh <- batch
			}
			return
		case obs := <-f.collectCh:
			batch = append(batch, obs)
			if len(batch) >= f.batchSize {
				f.processCh <- batch
				batch = make([]*Observation, 0, f.batchSize)
			}
		case <-ticker.C:
			if len(batch) > 0 {
				f.processCh <- batch
				batch = make([]*Observation, 0, f.batchSize)
			}
		}
	}
}

func (f *FeedbackLoop) processor() {
	for {
		select {
		case <-f.stopCh:
			return
		case batch := <-f.processCh:
			f.processBatch(batch)
		}
	}
}

func (f *FeedbackLoop) processBatch(batch []*Observation) {
	insights := f.generateInsights(batch)

	for _, insight := range insights {
		actions := f.createActions(insight)
		for _, action := range actions {
			f.executeAction(action)
		}
	}

	f.mu.Lock()
	f.processedCount += int64(len(batch))
	f.mu.Unlock()
}

func (f *FeedbackLoop) generateInsights(batch []*Observation) []*Insight {
	insightsMap := make(map[string]*Insight)

	for _, obs := range batch {
		key := fmt.Sprintf("%s_%s", obs.Type, obs.UserID)
		if insight, ok := insightsMap[key]; ok {
			insight.ObsIDs = append(insight.ObsIDs, obs.ID)
		} else {
			insightsMap[key] = &Insight{
				ID:         generateID(),
				ObsIDs:     []string{obs.ID},
				Category:   obs.Type,
				Confidence: 0.5,
				Content:    fmt.Sprintf("Observation batch: %d items", len(obs.Payload)),
				CreatedAt:  time.Now(),
			}
		}
	}

	insights := make([]*Insight, 0, len(insightsMap))
	for _, insight := range insightsMap {
		insights = append(insights, insight)
	}
	return insights
}

func (f *FeedbackLoop) createActions(insight *Insight) []*UpgradeAction {
	var actions []*UpgradeAction

	switch insight.Category {
	case ObservationTypeQueryResult:
		payload := insight.GetPayload()
		if success, ok := payload["success"].(bool); ok && !success {
			actions = append(actions, &UpgradeAction{
				ID:        generateID(),
				Type:      ActionTypeGenerateHint,
				Payload:   map[string]interface{}{"insight_id": insight.ID},
				Status:    "pending",
				CreatedAt: time.Now(),
			})
		}

	case ObservationTypeLLMLatency:
		payload := insight.GetPayload()
		if latency, ok := payload["latency_ms"].(float64); ok && latency > 10000 {
			actions = append(actions, &UpgradeAction{
				ID:        generateID(),
				Type:      ActionTypeAlert,
				Payload:   map[string]interface{}{"type": "high_latency", "value": latency},
				Status:    "pending",
				CreatedAt: time.Now(),
			})
		}

	case ObservationTypeImplicitSignal:
		actions = append(actions, &UpgradeAction{
			ID:        generateID(),
			Type:      ActionTypeUpdateSoul,
			Payload:   map[string]interface{}{"insight_id": insight.ID},
			Status:    "pending",
			CreatedAt: time.Now(),
		})
	}

	return actions
}

func (i *Insight) GetPayload() map[string]interface{} {
	result := make(map[string]interface{})
	result["content"] = i.Content
	result["category"] = i.Category
	result["confidence"] = i.Confidence
	return result
}

func (f *FeedbackLoop) executeAction(action *UpgradeAction) {
	f.mu.RLock()
	handler, ok := f.actions[action.Type]
	f.mu.RUnlock()

	if !ok {
		log.Printf("[FeedbackLoop] No handler for action type: %s", action.Type)
		f.mu.Lock()
		f.errorCount++
		f.mu.Unlock()
		return
	}

	if err := handler(action); err != nil {
		log.Printf("[FeedbackLoop] Action failed: %v", err)
		f.mu.Lock()
		f.errorCount++
		f.mu.Unlock()
	} else {
		now := time.Now()
		action.ExecutedAt = &now
		action.Status = "completed"
	}
}

func (f *FeedbackLoop) GetStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return map[string]interface{}{
		"processed_count":    f.processedCount,
		"error_count":        f.errorCount,
		"channel_buffer":     len(f.collectCh),
		"batch_size":         f.batchSize,
		"batch_timeout_sec":  f.batchTimeout.Seconds(),
		"registered_actions": len(f.actions),
	}
}

func generateID() string {
	return fmt.Sprintf("%d_%d", time.Now().UnixNano(), time.Now().Unix()%10000)
}

var _ = log.Printf
var _ = fmt.Sprintf

type FeedbackLoopAPI struct {
	loop *FeedbackLoop
}

func NewFeedbackLoopAPI(loop *FeedbackLoop) *FeedbackLoopAPI {
	return &FeedbackLoopAPI{loop: loop}
}

func (a *FeedbackLoopAPI) GetStats() map[string]interface{} {
	return a.loop.GetStats()
}

func (a *FeedbackLoopAPI) RecordQueryResult(userID, traceID string, success bool, latencyMs int64) {
	a.loop.CollectAsync(&Observation{
		ID:        generateID(),
		Type:      ObservationTypeQueryResult,
		Payload:   map[string]interface{}{"success": success, "latency_ms": latencyMs},
		Timestamp: time.Now(),
		UserID:    userID,
		TraceID:   traceID,
	})
}

func (a *FeedbackLoopAPI) RecordUserFeedback(userID, feedback string) {
	a.loop.CollectAsync(&Observation{
		ID:        generateID(),
		Type:      ObservationTypeUserFeedback,
		Payload:   map[string]interface{}{"feedback": feedback},
		Timestamp: time.Now(),
		UserID:    userID,
	})
}

func (a *FeedbackLoopAPI) RecordLLMLatency(userID, traceID string, latencyMs int64, tokens int) {
	a.loop.CollectAsync(&Observation{
		ID:        generateID(),
		Type:      ObservationTypeLLMLatency,
		Payload:   map[string]interface{}{"latency_ms": latencyMs, "tokens": tokens},
		Timestamp: time.Now(),
		UserID:    userID,
		TraceID:   traceID,
	})
}