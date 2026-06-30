package agent

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type MetricsRegistry struct {
	counters   map[string]*int64
	histograms map[string]*histogram
	mu         sync.RWMutex
	started    time.Time
	counterMu  sync.Mutex
}

type histogram struct {
	mu       sync.Mutex
	values   []float64
	count    int64
	sum      float64
	maxSize  int
	countMu  sync.Mutex
	sumMu    sync.Mutex
}

func NewMetricsRegistry() *MetricsRegistry {
	return &MetricsRegistry{
		counters:   make(map[string]*int64),
		histograms: make(map[string]*histogram),
		started:    time.Now(),
	}
}

func (m *MetricsRegistry) IncCounter(name string, labels ...Label) {
	counter := m.getOrCreateCounter(name, labels)
	m.counterMu.Lock()
	*counter++
	m.counterMu.Unlock()
}

func (m *MetricsRegistry) Add(name string, value int64, labels ...Label) {
	counter := m.getOrCreateCounter(name, labels)
	m.counterMu.Lock()
	*counter += value
	m.counterMu.Unlock()
}

func (m *MetricsRegistry) RecordValue(name string, value float64, labels ...Label) {
	key := metricKey(name, labels)
	m.mu.RLock()
	hist, ok := m.histograms[key]
	m.mu.RUnlock()

	if !ok {
		hist = &histogram{maxSize: 1000}
		m.mu.Lock()
		m.histograms[key] = hist
		m.mu.Unlock()
	}

	hist.mu.Lock()
	hist.values = append(hist.values, value)
	if len(hist.values) > hist.maxSize {
		hist.values = hist.values[1:]
	}
	hist.count++
	hist.sum += value
	hist.mu.Unlock()
}

func (m *MetricsRegistry) getOrCreateCounter(name string, labels []Label) *int64 {
	key := metricKey(name, labels)
	m.mu.RLock()
	counter, ok := m.counters[key]
	m.mu.RUnlock()

	if ok {
		return counter
	}

	var c int64
	counter = &c
	m.mu.Lock()
	m.counters[key] = counter
	m.mu.Unlock()
	return counter
}

func (m *MetricsRegistry) GetCounter(name string, labels ...Label) int64 {
	key := metricKey(name, labels)
	m.mu.RLock()
	defer m.mu.RUnlock()
	if counter, ok := m.counters[key]; ok {
		m.counterMu.Lock()
		val := *counter
		m.counterMu.Unlock()
		return val
	}
	return 0
}

func (m *MetricsRegistry) GetHistogram(name string, labels ...Label) (count int64, avg, p50, p95, max float64) {
	key := metricKey(name, labels)
	m.mu.RLock()
	hist, ok := m.histograms[key]
	m.mu.RUnlock()

	if !ok {
		return
	}

	hist.mu.Lock()
	count = hist.count
	if count == 0 {
		hist.mu.Unlock()
		return
	}

	values := make([]float64, len(hist.values))
	copy(values, hist.values)
	hist.mu.Unlock()

	sum := hist.sum
	avg = sum / float64(count)

	sortFloat64(values)
	if len(values) > 0 {
		max = values[len(values)-1]
		p50 = values[len(values)/2]
		p95 = values[int(float64(len(values))*0.95)]
	}

	return
}

func (m *MetricsRegistry) GetAll() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := map[string]interface{}{
		"uptime_seconds": time.Since(m.started).Seconds(),
		"counters":       make(map[string]int64),
		"histograms":     make(map[string]interface{}),
	}

	for key, counter := range m.counters {
		m.counterMu.Lock()
		result["counters"].(map[string]int64)[key] = *counter
		m.counterMu.Unlock()
	}

	for key, hist := range m.histograms {
		hist.mu.Lock()
		count := hist.count
		if count > 0 {
			result["histograms"].(map[string]interface{})[key] = map[string]interface{}{
				"count": count,
				"sum":   hist.sum,
				"avg":   hist.sum / float64(count),
			}
		}
		hist.mu.Unlock()
	}

	return result
}

func metricKey(name string, labels []Label) string {
	if len(labels) == 0 {
		return name
	}
	result := name
	for _, l := range labels {
		result += fmt.Sprintf(",%s=%s", l.Name, l.Value)
	}
	return result
}

func sortFloat64(s []float64) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

type Label struct {
	Name  string
	Value string
}

type Telemetry struct {
	registry *MetricsRegistry
	tracers  map[string]*Tracer
	mu       sync.RWMutex
	config   TelemetryConfig
}

type TelemetryConfig struct {
	Enabled     bool
	ServiceName string
}

var globalTelemetry *Telemetry
var telemetryOnce sync.Once

func InitTelemetry(cfg TelemetryConfig) *Telemetry {
	telemetryOnce.Do(func() {
		globalTelemetry = &Telemetry{
			registry: NewMetricsRegistry(),
			tracers:  make(map[string]*Tracer),
			config:   cfg,
		}
		log.Printf("[Telemetry] Initialized: service=%s", cfg.ServiceName)
	})
	return globalTelemetry
}

func GetTelemetry() *Telemetry {
	return globalTelemetry
}

func (t *Telemetry) RecordMetric(name string, value int64, labels ...Label) {
	if t.registry != nil {
		t.registry.Add(name, value, labels...)
	}
}

func (t *Telemetry) RecordLatency(name string, ms float64, labels ...Label) {
	if t.registry != nil {
		t.registry.RecordValue(name, ms, labels...)
	}
}

func (t *Telemetry) IncCounter(name string, labels ...Label) {
	if t.registry != nil {
		t.registry.IncCounter(name, labels...)
	}
}

func (t *Telemetry) GetMetrics() map[string]interface{} {
	if t.registry == nil {
		return nil
	}
	return t.registry.GetAll()
}

func (t *Telemetry) GetTracer(name string) *Tracer {
	t.mu.RLock()
	tracer, ok := t.tracers[name]
	t.mu.RUnlock()

	if !ok {
		tracer = &Tracer{name: name, registry: t.registry}
		t.mu.Lock()
		t.tracers[name] = tracer
		t.mu.Unlock()
	}
	return tracer
}

type Tracer struct {
	name     string
	registry *MetricsRegistry
}

func (t *Tracer) Start(ctx context.Context, name string) (context.Context, func()) {
	start := time.Now()
	return ctx, func() {
		elapsed := float64(time.Since(start).Milliseconds())
		if t.registry != nil {
			t.registry.RecordValue(t.name+"_duration_ms", elapsed,
				Label{Name: "operation", Value: name},
			)
			t.registry.IncCounter(t.name+"_total",
				Label{Name: "operation", Value: name},
			)
		}
	}
}

type OperationTimer struct {
	start     time.Time
	telemetry *Telemetry
	operation string
	labels    []Label
}

func StartTimer(ctx context.Context, operation string, labels ...Label) *OperationTimer {
	t := GetTelemetry()
	return &OperationTimer{
		start:      time.Now(),
		telemetry:  t,
		operation:  operation,
		labels:     labels,
	}
}

func (t *OperationTimer) End(status string) {
	if t.telemetry == nil {
		return
	}

	elapsed := float64(time.Since(t.start).Milliseconds())
	t.telemetry.RecordLatency(t.operation, elapsed, t.labels...)

	statusLabel := Label{Name: "status", Value: status}
	allLabels := append(t.labels, statusLabel)
	t.telemetry.IncCounter("agent_operations_total",
		append([]Label{{Name: "operation", Value: t.operation}}, allLabels...)...,
	)
}

func (t *OperationTimer) EndWithError(err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	t.End(status)
}

func RecordAgentOperation(ctx context.Context, operation string, fn func() error) error {
	telemetry := GetTelemetry()
	if telemetry == nil {
		return fn()
	}

	tracer := telemetry.GetTracer(operation)
	ctx, end := tracer.Start(ctx, operation)
	defer end()

	start := time.Now()
	err := fn()

	telemetry.RecordLatency(operation, float64(time.Since(start).Milliseconds()),
		Label{Name: "status", Value: "success"},
	)
	if err != nil {
		telemetry.RecordLatency(operation, float64(time.Since(start).Milliseconds()),
			Label{Name: "status", Value: "error"},
		)
	}

	return err
}

var _ = log.Printf