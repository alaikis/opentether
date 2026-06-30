package agent

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type SystemObserver struct {
	mu       sync.RWMutex

	skillStats       map[string]*SkillStat
	dataSourceHealth map[string]*DataSourceHealth
	llmMetrics       map[string]*LLMQualityMetrics
	userBehaviors    map[string]*UserBehavior

	collectionTicker *time.Ticker
	stopCh           chan struct{}
}

type SkillStat struct {
	SkillID      string
	TotalCalls   int
	SuccessCalls int
	FailedCalls  int
	TotalLatency int64
	LastUsedAt   time.Time
}

func (s *SkillStat) AvgLatencyMs() float64 {
	if s.TotalCalls == 0 {
		return 0
	}
	return float64(s.TotalLatency) / float64(s.TotalCalls)
}

func (s *SkillStat) SuccessRate() float64 {
	if s.TotalCalls == 0 {
		return 0
	}
	return float64(s.SuccessCalls) / float64(s.TotalCalls)
}

type DataSourceHealth struct {
	DataSourceID      string
	TotalQueries      int
	SuccessQueries    int
	TimeoutQueries    int
	ConnectionErrors  int
	TotalLatency      int64
	LastChecked       time.Time
}

func (d *DataSourceHealth) AvgLatencyMs() float64 {
	if d.TotalQueries == 0 {
		return 0
	}
	return float64(d.TotalLatency) / float64(d.TotalQueries)
}

func (d *DataSourceHealth) TimeoutRate() float64 {
	if d.TotalQueries == 0 {
		return 0
	}
	return float64(d.TimeoutQueries) / float64(d.TotalQueries)
}

type LLMQualityMetrics struct {
	ProviderID   string
	TotalCalls   int
	SuccessCalls int
	ErrorCalls   int
	TotalTokens  int
	TotalLatency int64
	LastCallAt   time.Time
}

func (l *LLMQualityMetrics) AvgLatencyMs() float64 {
	if l.TotalCalls == 0 {
		return 0
	}
	return float64(l.TotalLatency) / float64(l.TotalCalls)
}

func (l *LLMQualityMetrics) ErrorRate() float64 {
	if l.TotalCalls == 0 {
		return 0
	}
	return float64(l.ErrorCalls) / float64(l.TotalCalls)
}

type UserBehavior struct {
	UserID            string
	TotalQueries      int
	ActiveSessions    int
	TotalSessionLength int
	FollowUpQuestions int
	QueryTypes        map[string]int
	ActiveHours       map[int]int
	FirstActiveAt     time.Time
	LastActiveAt      time.Time
}

func (u *UserBehavior) AvgSessionLength() float64 {
	if u.ActiveSessions == 0 {
		return 0
	}
	return float64(u.TotalSessionLength) / float64(u.ActiveSessions)
}

func (u *UserBehavior) FollowUpRate() float64 {
	if u.TotalQueries == 0 {
		return 0
	}
	return float64(u.FollowUpQuestions) / float64(u.TotalQueries)
}

func NewSystemObserver() *SystemObserver {
	return &SystemObserver{
		skillStats:        make(map[string]*SkillStat),
		dataSourceHealth:  make(map[string]*DataSourceHealth),
		llmMetrics:        make(map[string]*LLMQualityMetrics),
		userBehaviors:     make(map[string]*UserBehavior),
		stopCh:            make(chan struct{}),
	}
}

func (o *SystemObserver) Start() {
	o.collectionTicker = time.NewTicker(1 * time.Minute)
	go func() {
		for {
			select {
			case <-o.stopCh:
				o.collectionTicker.Stop()
				o.persistStats()
				return
			case <-o.collectionTicker.C:
				o.persistStats()
			}
		}
	}()
	log.Printf("[Observer] System observer started")
}

func (o *SystemObserver) Stop() {
	close(o.stopCh)
	log.Printf("[Observer] System observer stopped")
}

func (o *SystemObserver) RecordSkillCall(skillID string, latencyMs int64, success bool) {
	o.mu.Lock()
	defer o.mu.Unlock()

	stat, ok := o.skillStats[skillID]
	if !ok {
		stat = &SkillStat{SkillID: skillID}
		o.skillStats[skillID] = stat
	}
	stat.TotalCalls++
	stat.TotalLatency += latencyMs
	stat.LastUsedAt = time.Now()
	if success {
		stat.SuccessCalls++
	} else {
		stat.FailedCalls++
	}
}

func (o *SystemObserver) RecordDataSourceQuery(dataSourceID string, latencyMs int64, timeout, connectionError bool) {
	o.mu.Lock()
	defer o.mu.Unlock()

	health, ok := o.dataSourceHealth[dataSourceID]
	if !ok {
		health = &DataSourceHealth{DataSourceID: dataSourceID}
		o.dataSourceHealth[dataSourceID] = health
	}
	health.TotalQueries++
	health.TotalLatency += latencyMs
	health.LastChecked = time.Now()

	if timeout {
		health.TimeoutQueries++
	}
	if connectionError {
		health.ConnectionErrors++
	}
	if !timeout && !connectionError {
		health.SuccessQueries++
	}
}

func (o *SystemObserver) RecordLLMCall(providerID string, latencyMs int64, tokens int, success bool, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	metrics, ok := o.llmMetrics[providerID]
	if !ok {
		metrics = &LLMQualityMetrics{ProviderID: providerID}
		o.llmMetrics[providerID] = metrics
	}
	metrics.TotalCalls++
	metrics.TotalLatency += latencyMs
	metrics.TotalTokens += tokens
	metrics.LastCallAt = time.Now()

	if success {
		metrics.SuccessCalls++
	} else {
		metrics.ErrorCalls++
	}
}

func (o *SystemObserver) RecordUserQuery(userID string, queryType string, isFollowUp bool) {
	o.mu.Lock()
	defer o.mu.Unlock()

	behavior, ok := o.userBehaviors[userID]
	if !ok {
		behavior = &UserBehavior{
			UserID:     userID,
			QueryTypes: make(map[string]int),
			ActiveHours: make(map[int]int),
		}
		behavior.FirstActiveAt = time.Now()
		o.userBehaviors[userID] = behavior
	}

	behavior.TotalQueries++
	behavior.LastActiveAt = time.Now()
	behavior.ActiveHours[time.Now().Hour()]++

	if isFollowUp {
		behavior.FollowUpQuestions++
	}

	if queryType != "" {
		behavior.QueryTypes[queryType]++
	}
}

func (o *SystemObserver) RecordUserSession(userID string, sessionLength int) {
	o.mu.Lock()
	defer o.mu.Unlock()

	behavior, ok := o.userBehaviors[userID]
	if !ok {
		behavior = &UserBehavior{
			UserID:     userID,
			QueryTypes: make(map[string]int),
			ActiveHours: make(map[int]int),
		}
		behavior.FirstActiveAt = time.Now()
		o.userBehaviors[userID] = behavior
	}

	behavior.ActiveSessions++
	behavior.TotalSessionLength += sessionLength
	behavior.LastActiveAt = time.Now()
}

func (o *SystemObserver) GetSkillStats(skillID string) *SkillStat {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if stat, ok := o.skillStats[skillID]; ok {
		result := *stat
		return &result
	}
	return nil
}

func (o *SystemObserver) GetAllSkillStats() map[string]SkillStat {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make(map[string]SkillStat)
	for id, stat := range o.skillStats {
		result[id] = *stat
	}
	return result
}

func (o *SystemObserver) GetDataSourceHealth(dataSourceID string) *DataSourceHealth {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if health, ok := o.dataSourceHealth[dataSourceID]; ok {
		result := *health
		return &result
	}
	return nil
}

func (o *SystemObserver) GetAllDataSourceHealth() map[string]DataSourceHealth {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make(map[string]DataSourceHealth)
	for id, health := range o.dataSourceHealth {
		result[id] = *health
	}
	return result
}

func (o *SystemObserver) GetLLMMetrics(providerID string) *LLMQualityMetrics {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if metrics, ok := o.llmMetrics[providerID]; ok {
		result := *metrics
		return &result
	}
	return nil
}

func (o *SystemObserver) GetAllLLMMetrics() map[string]LLMQualityMetrics {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make(map[string]LLMQualityMetrics)
	for id, metrics := range o.llmMetrics {
		result[id] = *metrics
	}
	return result
}

func (o *SystemObserver) GetUserBehavior(userID string) *UserBehavior {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if behavior, ok := o.userBehaviors[userID]; ok {
		result := *behavior
		result.QueryTypes = make(map[string]int)
		for k, v := range behavior.QueryTypes {
			result.QueryTypes[k] = v
		}
		result.ActiveHours = make(map[int]int)
		for k, v := range behavior.ActiveHours {
			result.ActiveHours[k] = v
		}
		return &result
	}
	return nil
}

func (o *SystemObserver) GetAllUserBehaviors() map[string]UserBehavior {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make(map[string]UserBehavior)
	for id, behavior := range o.userBehaviors {
		behaviorCopy := *behavior
		behaviorCopy.QueryTypes = make(map[string]int)
		for k, v := range behavior.QueryTypes {
			behaviorCopy.QueryTypes[k] = v
		}
		behaviorCopy.ActiveHours = make(map[int]int)
		for k, v := range behavior.ActiveHours {
			behaviorCopy.ActiveHours[k] = v
		}
		result[id] = behaviorCopy
	}
	return result
}

func (o *SystemObserver) persistStats() {
	o.mu.RLock()
	defer o.mu.RUnlock()

	log.Printf("[Observer] Stats checkpoint: %d skills, %d datasources, %d providers, %d users",
		len(o.skillStats), len(o.dataSourceHealth), len(o.llmMetrics), len(o.userBehaviors))
}

func (o *SystemObserver) GetSystemHealth() map[string]interface{} {
	o.mu.RLock()
	defer o.mu.RUnlock()

	health := map[string]interface{}{
		"timestamp":       time.Now(),
		"tracked_skills":  len(o.skillStats),
		"tracked_sources": len(o.dataSourceHealth),
		"tracked_providers": len(o.llmMetrics),
		"tracked_users":   len(o.userBehaviors),
	}

	var totalSkillCalls, totalSkillSuccess int
	var totalDSQueries, totalDSSuccess int
	var totalLLMCalls, totalLLMSuccess int

	for _, s := range o.skillStats {
		totalSkillCalls += s.TotalCalls
		totalSkillSuccess += s.SuccessCalls
	}
	for _, d := range o.dataSourceHealth {
		totalDSQueries += d.TotalQueries
		totalDSSuccess += d.SuccessQueries
	}
	for _, l := range o.llmMetrics {
		totalLLMCalls += l.TotalCalls
		totalLLMSuccess += l.SuccessCalls
	}

	health["skill_success_rate"] = 0.0
	if totalSkillCalls > 0 {
		health["skill_success_rate"] = float64(totalSkillSuccess) / float64(totalSkillCalls)
	}

	health["datasource_success_rate"] = 0.0
	if totalDSQueries > 0 {
		health["datasource_success_rate"] = float64(totalDSSuccess) / float64(totalDSQueries)
	}

	health["llm_success_rate"] = 0.0
	if totalLLMCalls > 0 {
		health["llm_success_rate"] = float64(totalLLMSuccess) / float64(totalLLMCalls)
	}

	return health
}

func (o *SystemObserver) ExportPrometheus() string {
	o.mu.RLock()
	defer o.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("# HELP wisehoof_skill_calls_total Total skill calls\n")
	sb.WriteString("# TYPE wisehoof_skill_calls_total counter\n")
	for skillID, stat := range o.skillStats {
		sb.WriteString(fmt.Sprintf("wisehoof_skill_calls_total{skill_id=\"%s\"} %d\n", skillID, stat.TotalCalls))
	}

	sb.WriteString("# HELP wisehoof_skill_success_rate Skill success rate\n")
	sb.WriteString("# TYPE wisehoof_skill_success_rate gauge\n")
	for skillID, stat := range o.skillStats {
		sb.WriteString(fmt.Sprintf("wisehoof_skill_success_rate{skill_id=\"%s\"} %.4f\n", skillID, stat.SuccessRate()))
	}

	sb.WriteString("# HELP wisehoof_skill_latency_ms Average skill latency in milliseconds\n")
	sb.WriteString("# TYPE wisehoof_skill_latency_ms gauge\n")
	for skillID, stat := range o.skillStats {
		sb.WriteString(fmt.Sprintf("wisehoof_skill_latency_ms{skill_id=\"%s\"} %.2f\n", skillID, stat.AvgLatencyMs()))
	}

	sb.WriteString("# HELP wisehoof_datasource_queries_total Total datasource queries\n")
	sb.WriteString("# TYPE wisehoof_datasource_queries_total counter\n")
	for dsID, health := range o.dataSourceHealth {
		sb.WriteString(fmt.Sprintf("wisehoof_datasource_queries_total{data_source_id=\"%s\"} %d\n", dsID, health.TotalQueries))
	}

	sb.WriteString("# HELP wisehoof_datasource_timeout_rate Datasource timeout rate\n")
	sb.WriteString("# TYPE wisehoof_datasource_timeout_rate gauge\n")
	for dsID, health := range o.dataSourceHealth {
		sb.WriteString(fmt.Sprintf("wisehoof_datasource_timeout_rate{data_source_id=\"%s\"} %.4f\n", dsID, health.TimeoutRate()))
	}

	sb.WriteString("# HELP wisehoof_llm_calls_total Total LLM calls\n")
	sb.WriteString("# TYPE wisehoof_llm_calls_total counter\n")
	for providerID, metrics := range o.llmMetrics {
		sb.WriteString(fmt.Sprintf("wisehoof_llm_calls_total{provider_id=\"%s\"} %d\n", providerID, metrics.TotalCalls))
	}

	sb.WriteString("# HELP wisehoof_llm_error_rate LLM error rate\n")
	sb.WriteString("# TYPE wisehoof_llm_error_rate gauge\n")
	for providerID, metrics := range o.llmMetrics {
		sb.WriteString(fmt.Sprintf("wisehoof_llm_error_rate{provider_id=\"%s\"} %.4f\n", providerID, metrics.ErrorRate()))
	}

	sb.WriteString("# HELP wisehoof_user_queries_total Total user queries\n")
	sb.WriteString("# TYPE wisehoof_user_queries_total counter\n")
	for userID, behavior := range o.userBehaviors {
		sb.WriteString(fmt.Sprintf("wisehoof_user_queries_total{user_id=\"%s\"} %d\n", userID, behavior.TotalQueries))
	}

	return sb.String()
}

var _ = fmt.Sprintf

type ObserverAPI struct {
	observer *SystemObserver
}

func NewObserverAPI(observer *SystemObserver) *ObserverAPI {
	return &ObserverAPI{observer: observer}
}

func (a *ObserverAPI) GetSkillsStats() map[string]SkillStat {
	return a.observer.GetAllSkillStats()
}

func (a *ObserverAPI) GetDataSourcesHealth() map[string]DataSourceHealth {
	return a.observer.GetAllDataSourceHealth()
}

func (a *ObserverAPI) GetLLMMetrics() map[string]LLMQualityMetrics {
	return a.observer.GetAllLLMMetrics()
}

func (a *ObserverAPI) GetUserActivity(userID string) *UserBehavior {
	return a.observer.GetUserBehavior(userID)
}

func (a *ObserverAPI) GetSystemHealth() map[string]interface{} {
	return a.observer.GetSystemHealth()
}