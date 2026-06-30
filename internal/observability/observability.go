package observability

import (
	"context"
	"time"
)

type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
)

type MetricDefinition struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        MetricType        `json:"type"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels"`
	Unit        string            `json:"unit"`
	CreatedAt   time.Time         `json:"created_at"`
}

type MetricValue struct {
	MetricID string            `json:"metric_id"`
	Labels   map[string]string `json:"labels"`
	Value    float64           `json:"value"`
	Ts       time.Time         `json:"ts"`
}

type AlertRule struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	MetricID    string    `json:"metric_id"`
	Condition   string    `json:"condition"`
	Threshold   float64   `json:"threshold"`
	Window      string    `json:"window"`
	Severity    string    `json:"severity"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

type AlertEvent struct {
	ID        string    `json:"id"`
	RuleID    string    `json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Value     float64   `json:"value"`
	FiredAt   time.Time `json:"fired_at"`
	Resolved  bool      `json:"resolved"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type HookFunc func(ctx context.Context, name string, value float64, labels map[string]string)

type MetricCollector interface {
	Register(def *MetricDefinition) error
	Record(ctx context.Context, metricID string, value float64, labels map[string]string) error
	Query(metricID string, start, end time.Time) ([]*MetricValue, error)
	List() ([]*MetricDefinition, error)
}

type AlertRuleEngine interface {
	CreateRule(rule *AlertRule) error
	UpdateRule(id string, rule *AlertRule) error
	DeleteRule(id string) error
	ListRules() ([]*AlertRule, error)
	Evaluate(now time.Time) ([]*AlertEvent, error)
	AckAlert(id string) error
}

type InstrumentHookRegistry interface {
	Register(name string, fn HookFunc) error
	Unregister(name string) error
	Emit(ctx context.Context, name string, value float64, labels map[string]string)
}
