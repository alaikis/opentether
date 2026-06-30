package tuning

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/alaikis/opentether/internal/models"
)

type InMemoryOptimizer struct {
	mu      sync.RWMutex
	samples map[string][]*TuningIteration
}

func NewInMemoryOptimizer() *InMemoryOptimizer {
	return &InMemoryOptimizer{
		samples: make(map[string][]*TuningIteration),
	}
}

func (o *InMemoryOptimizer) Optimize(ctx context.Context, job *TuningJob) (*TuningIteration, error) {
	if job == nil || len(job.Parameters) == 0 {
		return nil, errors.New("invalid tuning job")
	}
	values := make([]*ParameterValue, 0, len(job.Parameters))
	samples := o.samples[job.ID]
	for _, p := range job.Parameters {
		switch p.Type {
		case ParameterTypeContinuous:
			if p.Min != nil && p.Max != nil {
				v := selectNextValue(p, samples)
				if p.Step != nil {
					step := *p.Step
					v = math.Round(v/step) * step
				}
				values = append(values, &ParameterValue{Name: p.Name, Value: v})
			}
		case ParameterTypeDiscrete, ParameterTypeCategorical:
			if len(p.Options) > 0 {
				idx := int(randFloat64() * float64(len(p.Options)))
				values = append(values, &ParameterValue{Name: p.Name, Value: p.Options[idx]})
			}
		default:
			values = append(values, &ParameterValue{Name: p.Name, Value: p.Default})
		}
	}
	iteration := &TuningIteration{
		ID:        generateIterationID(),
		JobID:     job.ID,
		Values:    values,
		Score:     0,
		Iteration: job.Iterations + 1,
		CreatedAt: time.Now(),
	}
	o.mu.Lock()
	o.samples[job.ID] = append(o.samples[job.ID], iteration)
	o.mu.Unlock()
	return iteration, nil
}

func selectNextValue(param *Parameter, samples []*TuningIteration) float64 {
	if param.Min == nil || param.Max == nil {
		return 0
	}
	if len(samples) < 3 {
		return *param.Min + randFloat64()*(*param.Max-*param.Min)
	}
	bestIdx := -1
	bestScore := -math.MaxFloat64
	for i, s := range samples {
		if s.Score > bestScore {
			bestScore = s.Score
			bestIdx = i
		}
	}
	center := *param.Min + randFloat64()*(*param.Max-*param.Min)
	if bestIdx >= 0 && len(samples[bestIdx].Values) > 0 {
		for _, v := range samples[bestIdx].Values {
			if v.Name == param.Name {
				if fv, ok := v.Value.(float64); ok {
					center = fv
				}
				break
			}
		}
	}
	spread := (*param.Max - *param.Min) / float64(len(samples)+1)
	offset := (randFloat64() - 0.5) * 2 * spread
	v := center + offset
	if v < *param.Min {
		v = *param.Min
	}
	if v > *param.Max {
		v = *param.Max
	}
	return v
}

func (o *InMemoryOptimizer) Update(ctx context.Context, jobID string, iteration *TuningIteration) error {
	if iteration == nil {
		return errors.New("nil iteration")
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	samples, ok := o.samples[jobID]
	if !ok {
		return errors.New("job not found")
	}
	for _, s := range samples {
		if s.ID == iteration.ID {
			s.Score = iteration.Score
			s.Metrics = iteration.Metrics
			break
		}
	}
	return nil
}

func (o *InMemoryOptimizer) Best(ctx context.Context, jobID string) (*TuningIteration, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	samples, ok := o.samples[jobID]
	if !ok || len(samples) == 0 {
		return nil, errors.New("no iterations")
	}
	sort.Slice(samples, func(i, j int) bool {
		return samples[i].Score > samples[j].Score
	})
	return samples[0], nil
}

type InMemoryRuleOptimizer struct{}

func NewInMemoryRuleOptimizer() *InMemoryRuleOptimizer {
	return &InMemoryRuleOptimizer{}
}

func (o *InMemoryRuleOptimizer) Suggest(params []*ParameterValue, metrics map[string]float64) ([]*ParameterValue, error) {
	if len(params) == 0 {
		return nil, errors.New("no parameters")
	}
	result := make([]*ParameterValue, len(params))
	for i, p := range params {
		result[i] = &ParameterValue{Name: p.Name, Value: p.Value}
	}
	if latency, ok := metrics["latency_ms"]; ok && latency > 5000 {
		for _, p := range result {
			if p.Name == "timeout" {
				p.Value = 30000
			}
		}
	}
	if pv, ok := metrics["llm_provider"]; ok && pv != 0 {
		for _, p := range result {
			if p.Name == "llm_provider" {
				p.Value = int(pv)
			}
		}
	}
	return result, nil
}

func (o *InMemoryRuleOptimizer) SuggestProvider(providers []*models.Provider, metrics map[string]float64) (*models.Provider, error) {
	if len(providers) == 0 {
		return nil, errors.New("no providers")
	}
	threshold := 5000.0
	best := providers[0]
	bestLatency := math.MaxFloat64
	for _, p := range providers {
		key := "provider_latency_" + p.ID
		if lat, ok := metrics[key]; ok && lat < threshold {
			if lat < bestLatency {
				bestLatency = lat
				best = p
			}
		}
	}
	return best, nil
}

type InMemoryTuningHistory struct {
	mu       sync.RWMutex
	history  map[string][]*TuningIteration
	dataDir  string
}

func NewInMemoryTuningHistory(dataDir string) *InMemoryTuningHistory {
	return &InMemoryTuningHistory{
		history: make(map[string][]*TuningIteration),
		dataDir: dataDir,
	}
}

func (h *InMemoryTuningHistory) RecordIteration(iteration *TuningIteration) error {
	if iteration == nil {
		return errors.New("nil iteration")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.history[iteration.JobID] = append(h.history[iteration.JobID], iteration)
	return h.persist(iteration.JobID)
}

func (h *InMemoryTuningHistory) History(jobID string) ([]*TuningIteration, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result, ok := h.history[jobID]
	if !ok {
		return []*TuningIteration{}, nil
	}
	dup := make([]*TuningIteration, len(result))
	copy(dup, result)
	return dup, nil
}

func (h *InMemoryTuningHistory) Rollback(jobID string, iteration int) (*ParameterValue, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	history, ok := h.history[jobID]
	if !ok {
		return nil, errors.New("job not found")
	}
	for _, it := range history {
		if it.Iteration == iteration {
			if len(it.Values) == 0 {
				return nil, errors.New("no values in iteration")
			}
			return it.Values[0], nil
		}
	}
	return nil, errors.New("iteration not found")
}

func (h *InMemoryTuningHistory) persist(jobID string) error {
	if h.dataDir == "" {
		return nil
	}
	_ = os.MkdirAll(h.dataDir, 0755)
	data, _ := json.Marshal(h.history[jobID])
	return os.WriteFile(filepath.Join(h.dataDir, jobID+".json"), data, 0644)
}

func randFloat64() float64 {
	return float64(rand.Intn(10000)) / 10000.0
}

func generateIterationID() string {
	return "iter_" + time.Now().Format("20060102_150405")
}
