package observability

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type InMemoryMetricCollector struct {
	mu      sync.RWMutex
	metrics map[string]*MetricDefinition
	values  map[string][]*MetricValue
}

func NewInMemoryMetricCollector() *InMemoryMetricCollector {
	return &InMemoryMetricCollector{
		metrics: make(map[string]*MetricDefinition),
		values:  make(map[string][]*MetricValue),
	}
}

func (c *InMemoryMetricCollector) Register(def *MetricDefinition) error {
	if def == nil || def.ID == "" || def.Name == "" {
		return errors.New("invalid metric definition")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	def.CreatedAt = time.Now()
	c.metrics[def.ID] = def
	return nil
}

func (c *InMemoryMetricCollector) Record(ctx context.Context, metricID string, value float64, labels map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.metrics[metricID]; !ok {
		return errors.New("metric not found: " + metricID)
	}
	entry := &MetricValue{
		MetricID: metricID,
		Labels:   labels,
		Value:    value,
		Ts:       time.Now(),
	}
	c.values[metricID] = append(c.values[metricID], entry)
	return nil
}

func (c *InMemoryMetricCollector) Query(metricID string, start, end time.Time) ([]*MetricValue, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	all, ok := c.values[metricID]
	if !ok {
		return []*MetricValue{}, nil
	}
	var result []*MetricValue
	for _, v := range all {
		if (v.Ts.Equal(start) || v.Ts.After(start)) && (v.Ts.Equal(end) || v.Ts.Before(end)) {
			result = append(result, v)
		}
	}
	return result, nil
}

func (c *InMemoryMetricCollector) List() ([]*MetricDefinition, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]*MetricDefinition, 0, len(c.metrics))
	for _, m := range c.metrics {
		result = append(result, m)
	}
	return result, nil
}

type InMemoryHookRegistry struct {
	mu     sync.RWMutex
	hooks  map[string]HookFunc
	events chan hookEvent
}

type hookEvent struct {
	name   string
	value  float64
	labels map[string]string
}

func NewInMemoryHookRegistry() *InMemoryHookRegistry {
	return &InMemoryHookRegistry{
		hooks:  make(map[string]HookFunc),
		events: make(chan hookEvent, 1024),
	}
}

func (r *InMemoryHookRegistry) Register(name string, fn HookFunc) error {
	if name == "" || fn == nil {
		return errors.New("invalid hook registration")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks[name] = fn
	return nil
}

func (r *InMemoryHookRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.hooks, name)
	return nil
}

func (r *InMemoryHookRegistry) Emit(ctx context.Context, name string, value float64, labels map[string]string) {
	r.mu.RLock()
	fn, ok := r.hooks[name]
	r.mu.RUnlock()
	if !ok {
		return
	}
	go fn(ctx, name, value, labels)
}

func (r *InMemoryHookRegistry) StartConsuming() {
	go func() {
		for evt := range r.events {
			r.Emit(context.Background(), evt.name, evt.value, evt.labels)
		}
	}()
}

type FileBackedAlertRuleEngine struct {
	mu        sync.RWMutex
	rules     map[string]*AlertRule
	alerts    map[string]*AlertEvent
	dataDir   string
	collector MetricCollector
}

func NewFileBackedAlertRuleEngine(dataDir string) *FileBackedAlertRuleEngine {
	return &FileBackedAlertRuleEngine{
		rules:   make(map[string]*AlertRule),
		alerts:  make(map[string]*AlertEvent),
		dataDir: dataDir,
	}
}

func (e *FileBackedAlertRuleEngine) CreateRule(rule *AlertRule) error {
	if rule == nil || rule.ID == "" || rule.Name == "" {
		return errors.New("invalid alert rule")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	rule.CreatedAt = time.Now()
	e.rules[rule.ID] = rule
	return e.persist()
}

func (e *FileBackedAlertRuleEngine) UpdateRule(id string, rule *AlertRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.rules[id]; !ok {
		return errors.New("rule not found")
	}
	rule.ID = id
	e.rules[id] = rule
	return e.persist()
}

func (e *FileBackedAlertRuleEngine) SetCollector(collector MetricCollector) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.collector = collector
}

func (e *FileBackedAlertRuleEngine) DeleteRule(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.rules, id)
	return e.persist()
}

func (e *FileBackedAlertRuleEngine) ListRules() ([]*AlertRule, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]*AlertRule, 0, len(e.rules))
	for _, r := range e.rules {
		result = append(result, r)
	}
	return result, nil
}

func (e *FileBackedAlertRuleEngine) Evaluate(now time.Time) ([]*AlertEvent, error) {
	e.mu.RLock()
	rules := make([]*AlertRule, 0, len(e.rules))
	for _, r := range e.rules {
		if r.Enabled {
			rules = append(rules, r)
		}
	}
	collector := e.collector
	e.mu.RUnlock()

	if collector == nil {
		return []*AlertEvent{}, nil
	}

	var fired []*AlertEvent
	for _, rule := range rules {
		duration, err := time.ParseDuration(rule.Window)
		if err != nil {
			continue
		}
		start := now.Add(-duration)
		values, err := collector.Query(rule.MetricID, start, now)
		if err != nil {
			continue
		}

		for _, v := range values {
			violated := false
			switch rule.Condition {
			case "gt":
				violated = v.Value > rule.Threshold
			case "gte":
				violated = v.Value >= rule.Threshold
			case "lt":
				violated = v.Value < rule.Threshold
			case "lte":
				violated = v.Value <= rule.Threshold
			case "eq":
				violated = v.Value == rule.Threshold
			}

			if violated {
				alertID := fmt.Sprintf("%s_%d", rule.ID, v.Ts.UnixNano())
				alert := &AlertEvent{
					ID:       alertID,
					RuleID:   rule.ID,
					RuleName: rule.Name,
					Severity: rule.Severity,
					Message:  fmt.Sprintf("Metric %s violated threshold: %f %s %f", rule.MetricID, v.Value, rule.Condition, rule.Threshold),
					Value:    v.Value,
					FiredAt:  now,
				}
				e.mu.Lock()
				e.alerts[alertID] = alert
				e.mu.Unlock()
				fired = append(fired, alert)
			}
		}
	}
	return fired, nil
}

func (e *FileBackedAlertRuleEngine) AckAlert(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	alert, ok := e.alerts[id]
	if !ok {
		return errors.New("alert not found")
	}
	alert.Resolved = true
	now := time.Now()
	alert.ResolvedAt = &now
	return e.persist()
}

func (e *FileBackedAlertRuleEngine) persist() error {
	if e.dataDir == "" {
		return nil
	}
	_ = os.MkdirAll(e.dataDir, 0755)
	rulesData, _ := json.Marshal(e.rules)
	if err := os.WriteFile(filepath.Join(e.dataDir, "alert_rules.json"), rulesData, 0644); err != nil {
		return err
	}
	alertsData, _ := json.Marshal(e.alerts)
	return os.WriteFile(filepath.Join(e.dataDir, "alert_events.json"), alertsData, 0644)
}
