package service

import (
	"time"

	"github.com/alaikis/opentether/internal/observability"
)

type ObservabilityService struct {
	db         interface{}
	collector  observability.MetricCollector
	hooks      observability.InstrumentHookRegistry
	alertRules observability.AlertRuleEngine
	dataDir    string
}

func NewObservabilityService(db interface{}) *ObservabilityService {
	dataDir := "data/observability"
	s := &ObservabilityService{
		db:         db,
		collector:  observability.NewInMemoryMetricCollector(),
		hooks:      observability.NewInMemoryHookRegistry(),
		alertRules: observability.NewFileBackedAlertRuleEngine(dataDir),
		dataDir:    dataDir,
	}
	s.hooks.(*observability.InMemoryHookRegistry).StartConsuming()
	return s
}

func (s *ObservabilityService) RegisterMetric(def *observability.MetricDefinition) error {
	return s.collector.Register(def)
}

func (s *ObservabilityService) ListMetrics() ([]*observability.MetricDefinition, error) {
	return s.collector.List()
}

func (s *ObservabilityService) QueryMetric(metricID string, start, end time.Time) ([]*observability.MetricValue, error) {
	return s.collector.Query(metricID, start, end)
}

func (s *ObservabilityService) CreateAlertRule(rule *observability.AlertRule) error {
	return s.alertRules.CreateRule(rule)
}

func (s *ObservabilityService) UpdateAlertRule(id string, rule *observability.AlertRule) error {
	return s.alertRules.UpdateRule(id, rule)
}

func (s *ObservabilityService) DeleteAlertRule(id string) error {
	return s.alertRules.DeleteRule(id)
}

func (s *ObservabilityService) ListAlertRules() ([]*observability.AlertRule, error) {
	return s.alertRules.ListRules()
}

func (s *ObservabilityService) ListAlertEvents() ([]*observability.AlertEvent, error) {
	return []*observability.AlertEvent{}, nil
}

func (s *ObservabilityService) AckAlert(id string) error {
	return s.alertRules.AckAlert(id)
}

func (s *ObservabilityService) RegisterHook(name string, fn observability.HookFunc) error {
	return s.hooks.Register(name, fn)
}
