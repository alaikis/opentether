package tuning

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/alaikis/opentether/internal/models"
)

func TestInMemoryOptimizerContinuousBounds(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	opt := NewInMemoryOptimizer()
	min, max := 0.0, 100.0
	job := &TuningJob{
		ID: "job1",
		Parameters: []*Parameter{
			{Name: "rate", Type: ParameterTypeContinuous, Min: &min, Max: &max},
		},
	}
	for i := 0; i < 20; i++ {
		it, err := opt.Optimize(nil, job)
		if err != nil {
			t.Fatalf("optimize: %v", err)
		}
		for _, v := range it.Values {
			fv := v.Value.(float64)
			if fv < min || fv > max {
				t.Errorf("value out of bounds: %v", fv)
			}
		}
	}
}

func TestInMemoryOptimizerImprovesOverIterations(t *testing.T) {
	rand.Seed(42)
	opt := NewInMemoryOptimizer()
	min, max := 0.0, 10.0
	job := &TuningJob{
		ID: "job2",
		Parameters: []*Parameter{
			{Name: "x", Type: ParameterTypeContinuous, Min: &min, Max: &max},
		},
	}
	var scores []float64
	for i := 0; i < 10; i++ {
		it, err := opt.Optimize(nil, job)
		if err != nil {
			t.Fatalf("optimize: %v", err)
		}
		x := it.Values[0].Value.(float64)
		it.Score = -math.Pow(x-7.5, 2)
		if err := opt.Update(nil, job.ID, it); err != nil {
			t.Fatalf("update: %v", err)
		}
		scores = append(scores, it.Score)
	}
	best := scores[0]
	for _, s := range scores {
		if s > best {
			best = s
		}
	}
	if best <= scores[0] {
		t.Errorf("expected some improvement, best=%v first=%v", best, scores[0])
	}
}

func TestInMemoryRuleOptimizerTimeoutSuggestion(t *testing.T) {
	o := NewInMemoryRuleOptimizer()
	params := []*ParameterValue{
		{Name: "timeout", Value: 5000},
	}
	metrics := map[string]float64{"latency_ms": 6000}
	result, err := o.Suggest(params, metrics)
	if err != nil {
		t.Fatalf("suggest: %v", err)
	}
	for _, p := range result {
		if p.Name == "timeout" && p.Value != 30000 {
			t.Errorf("expected timeout 30000, got %v", p.Value)
		}
	}
}

func TestInMemoryRuleOptimizerProviderSuggestion(t *testing.T) {
	o := NewInMemoryRuleOptimizer()
	providers := []*models.Provider{
		{ID: "p1", ProviderName: "Slow"},
		{ID: "p2", ProviderName: "Fast"},
	}
	metrics := map[string]float64{
		"provider_latency_p1": 6000,
		"provider_latency_p2": 1000,
	}
	result, err := o.SuggestProvider(providers, metrics)
	if err != nil {
		t.Fatalf("suggest provider: %v", err)
	}
	if result.ID != "p2" {
		t.Errorf("expected p2, got %s", result.ID)
	}
}

func TestInMemoryTuningHistoryRecordAndRetrieve(t *testing.T) {
	h := NewInMemoryTuningHistory("")
	it := &TuningIteration{
		ID:        "it1",
		JobID:     "job3",
		Values:    []*ParameterValue{{Name: "x", Value: 1.0}},
		Score:     10,
		Iteration: 1,
		CreatedAt: time.Now(),
	}
	if err := h.RecordIteration(it); err != nil {
		t.Fatalf("record: %v", err)
	}
	history, err := h.History("job3")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 iteration, got %d", len(history))
	}
}

func TestInMemoryTuningHistoryRollback(t *testing.T) {
	h := NewInMemoryTuningHistory("")
	h.RecordIteration(&TuningIteration{
		ID: "it1", JobID: "job4", Values: []*ParameterValue{{Name: "x", Value: 1.0}}, Iteration: 1,
	})
	h.RecordIteration(&TuningIteration{
		ID: "it2", JobID: "job4", Values: []*ParameterValue{{Name: "x", Value: 2.0}}, Iteration: 2,
	})
	v, err := h.Rollback("job4", 1)
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if v.Value != 1.0 {
		t.Errorf("expected 1.0, got %v", v.Value)
	}
}

func TestParameterValueGeneration(t *testing.T) {
	opt := NewInMemoryOptimizer()
	min, max := 0.0, 10.0
	step := 1.0
	job := &TuningJob{
		ID: "job5",
		Parameters: []*Parameter{
			{Name: "cont", Type: ParameterTypeContinuous, Min: &min, Max: &max, Step: &step},
			{Name: "disc", Type: ParameterTypeDiscrete, Options: []string{"a", "b", "c"}},
			{Name: "cat", Type: ParameterTypeCategorical, Options: []string{"x", "y"}},
		},
	}
	for i := 0; i < 10; i++ {
		it, err := opt.Optimize(nil, job)
		if err != nil {
			t.Fatalf("optimize: %v", err)
		}
		foundCont, foundDisc, foundCat := false, false, false
		for _, v := range it.Values {
			switch v.Name {
			case "cont":
				fv := v.Value.(float64)
				if fv < min || fv > max {
					t.Errorf("continuous out of bounds: %v", fv)
				}
				foundCont = true
			case "disc":
				foundDisc = true
				valid := false
				for _, o := range []string{"a", "b", "c"} {
					if v.Value == o {
						valid = true
						break
					}
				}
				if !valid {
					t.Errorf("discrete invalid value: %v", v.Value)
				}
			case "cat":
				foundCat = true
				valid := false
				for _, o := range []string{"x", "y"} {
					if v.Value == o {
						valid = true
						break
					}
				}
				if !valid {
					t.Errorf("categorical invalid value: %v", v.Value)
				}
			}
		}
		if !foundCont || !foundDisc || !foundCat {
			t.Errorf("missing parameter values")
		}
	}
}
