package agent

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

type PromptVersion struct {
	ID          string
	Key         string
	Version     string
	Content     string
	Metrics     PromptMetrics
	Status      string
	CreatedAt   time.Time
	ActivatedAt *time.Time
}

type PromptMetrics struct {
	TotalRequests  int
	SuccessCount   int
	FailureCount   int
	AvgLatencyMs   float64
	SuccessRate    float64
}

type ABTest struct {
	ID           string
	Key          string
	VersionA     string
	VersionB     string
	TrafficRatio float64
	StartTime    time.Time
	EndTime      *time.Time
	Status       string
	ResultsA     PromptMetrics
	ResultsB     PromptMetrics
}

type PromptEvolution struct {
	mu           sync.RWMutex
	versions     map[string]map[string]*PromptVersion
	abTests      map[string]*ABTest
	history      []PromptMetrics
	thresholdMin float64
	thresholdMax float64
	baseThreshold float64
}

func NewPromptEvolution() *PromptEvolution {
	return &PromptEvolution{
		versions:       make(map[string]map[string]*PromptVersion),
		abTests:        make(map[string]*ABTest),
		thresholdMin:   0.6,
		thresholdMax:   0.95,
		baseThreshold:  0.8,
	}
}

func (p *PromptEvolution) RegisterPrompt(key, version, content string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.versions[key]; !ok {
		p.versions[key] = make(map[string]*PromptVersion)
	}

	p.versions[key][version] = &PromptVersion{
		ID:        fmt.Sprintf("%s_%s", key, version),
		Key:       key,
		Version:   version,
		Content:   content,
		Status:    "active",
		CreatedAt: time.Now(),
	}

	log.Printf("[PromptEvolution] Registered prompt %s v%s", key, version)
}

func (p *PromptEvolution) RecordRequest(key, version string, latencyMs int64, success bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if versions, ok := p.versions[key]; ok {
		if v, ok := versions[version]; ok {
			v.Metrics.TotalRequests++
			if success {
				v.Metrics.SuccessCount++
			} else {
				v.Metrics.FailureCount++
			}

			prevLatency := v.Metrics.AvgLatencyMs
			if v.Metrics.TotalRequests == 1 {
				v.Metrics.AvgLatencyMs = float64(latencyMs)
			} else {
				v.Metrics.AvgLatencyMs = prevLatency + (float64(latencyMs)-prevLatency)/float64(v.Metrics.TotalRequests)
			}

			if v.Metrics.TotalRequests > 0 {
				v.Metrics.SuccessRate = float64(v.Metrics.SuccessCount) / float64(v.Metrics.TotalRequests)
			}
		}
	}

	p.history = append(p.history, PromptMetrics{})
	if len(p.history) > 1000 {
		p.history = p.history[1:]
	}
}

func (p *PromptEvolution) CalculateDynamicThreshold() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.history) < 10 {
		return p.baseThreshold
	}

	recentSuccessRate := 0.0
	count := 0
	start := len(p.history) - 10
	if start < 0 {
		start = 0
	}
	for i := start; i < len(p.history); i++ {
		if p.history[i].TotalRequests > 0 {
			recentSuccessRate += p.history[i].SuccessRate
			count++
		}
	}

	if count == 0 {
		return p.baseThreshold
	}

	recentSuccessRate /= float64(count)
	adjustment := (recentSuccessRate - 0.7) * 0.1
	threshold := p.baseThreshold + adjustment

	if threshold < p.thresholdMin {
		threshold = p.thresholdMin
	}
	if threshold > p.thresholdMax {
		threshold = p.thresholdMax
	}

	return threshold
}

func (p *PromptEvolution) ShouldUpgrade(key string) (bool, string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	versions, ok := p.versions[key]
	if !ok || len(versions) == 0 {
		return false, ""
	}

	currentVersion := ""
	var currentMetrics PromptMetrics
	for version, v := range versions {
		if v.Status == "active" {
			currentVersion = version
			currentMetrics = v.Metrics
			break
		}
	}

	if currentVersion == "" {
		return false, ""
	}

	dynamicThreshold := p.CalculateDynamicThresholdLocked()

	if currentMetrics.TotalRequests >= 10 && currentMetrics.SuccessRate > dynamicThreshold {
		for version, v := range versions {
			if version != currentVersion && v.Status == "testing" {
				if v.Metrics.SuccessRate > currentMetrics.SuccessRate {
					return true, version
				}
			}
		}
	}

	return false, ""
}

func (p *PromptEvolution) CalculateDynamicThresholdLocked() float64 {
	if len(p.history) < 10 {
		return p.baseThreshold
	}

	recentSuccessRate := 0.0
	count := 0
	start := len(p.history) - 10
	if start < 0 {
		start = 0
	}
	for i := start; i < len(p.history); i++ {
		if p.history[i].TotalRequests > 0 {
			recentSuccessRate += p.history[i].SuccessRate
			count++
		}
	}

	if count == 0 {
		return p.baseThreshold
	}

	recentSuccessRate /= float64(count)
	adjustment := (recentSuccessRate - 0.7) * 0.1
	threshold := p.baseThreshold + adjustment

	if threshold < p.thresholdMin {
		threshold = p.thresholdMin
	}
	if threshold > p.thresholdMax {
		threshold = p.thresholdMax
	}

	return threshold
}

func (p *PromptEvolution) StartABTest(key, versionA, versionB string, trafficRatio float64) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	testID := fmt.Sprintf("ab_%d", time.Now().UnixNano())
	test := &ABTest{
		ID:           testID,
		Key:          key,
		VersionA:     versionA,
		VersionB:     versionB,
		TrafficRatio: trafficRatio,
		StartTime:    time.Now(),
		Status:       "running",
	}

	p.abTests[testID] = test

	if versions, ok := p.versions[key]; ok {
		if v, ok := versions[versionA]; ok {
			v.Status = "testing"
		}
		if v, ok := versions[versionB]; ok {
			v.Status = "testing"
		}
	}

	log.Printf("[PromptEvolution] Started A/B test %s: %s vs %s (%.0f%% traffic to B)",
		testID, versionA, versionB, trafficRatio*100)

	return testID
}

func (p *PromptEvolution) RecordABTestResult(testID string, version string, latencyMs int64, success bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	test, ok := p.abTests[testID]
	if !ok || test.Status != "running" {
		return
	}

	if version == test.VersionA {
		test.ResultsA.TotalRequests++
		if success {
			test.ResultsA.SuccessCount++
		} else {
			test.ResultsA.FailureCount++
		}
	} else if version == test.VersionB {
		test.ResultsB.TotalRequests++
		if success {
			test.ResultsB.SuccessCount++
		} else {
			test.ResultsB.FailureCount++
		}
	}

	if test.ResultsA.TotalRequests > 0 {
		test.ResultsA.SuccessRate = float64(test.ResultsA.SuccessCount) / float64(test.ResultsA.TotalRequests)
	}
	if test.ResultsB.TotalRequests > 0 {
		test.ResultsB.SuccessRate = float64(test.ResultsB.SuccessCount) / float64(test.ResultsB.TotalRequests)
	}
}

func (p *PromptEvolution) GetABTestResult(testID string) map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	test, ok := p.abTests[testID]
	if !ok {
		return nil
	}

	return map[string]interface{}{
		"test_id":     test.ID,
		"status":      test.Status,
		"version_a":   test.VersionA,
		"version_b":   test.VersionB,
		"results_a":   test.ResultsA,
		"results_b":   test.ResultsB,
		"winner":      p.calculateABWinner(test),
		"confidence":  p.calculateABConfidence(test),
	}
}

func (p *PromptEvolution) calculateABWinner(test *ABTest) string {
	if test.ResultsA.TotalRequests == 0 || test.ResultsB.TotalRequests == 0 {
		return "insufficient_data"
	}

	if test.ResultsA.SuccessRate > test.ResultsB.SuccessRate {
		return test.VersionA
	} else if test.ResultsB.SuccessRate > test.ResultsA.SuccessRate {
		return test.VersionB
	}

	return "tie"
}

func (p *PromptEvolution) calculateABConfidence(test *ABTest) float64 {
	if test.ResultsA.TotalRequests < 10 || test.ResultsB.TotalRequests < 10 {
		return 0.0
	}

	p1 := test.ResultsA.SuccessRate
	p2 := test.ResultsB.SuccessRate
	n1 := float64(test.ResultsA.TotalRequests)
	n2 := float64(test.ResultsB.TotalRequests)

	pooled := (p1*n1 + p2*n2) / (n1 + n2)
	se := math.Sqrt(pooled * (1 - pooled) * (1/n1 + 1/n2))

	if se == 0 {
		return 0.0
	}

	z := math.Abs(p1-p2) / se

	confidence := 0.0
	if z >= 1.96 {
		confidence = 0.95
	} else if z >= 1.645 {
		confidence = 0.90
	} else if z >= 1.28 {
		confidence = 0.80
	} else {
		confidence = z / 1.96 * 0.95
	}

	return confidence
}

func (p *PromptEvolution) PromoteVersion(key, version string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	versions, ok := p.versions[key]
	if !ok {
		return fmt.Errorf("prompt key not found: %s", key)
	}

	for _, pv := range versions {
		if pv.Status == "active" {
			pv.Status = "archived"
		}
	}

	if v, ok := versions[version]; ok {
		v.Status = "active"
		now := time.Now()
		v.ActivatedAt = &now
		log.Printf("[PromptEvolution] Promoted %s to version %s", key, version)
		return nil
	}

	return fmt.Errorf("version not found: %s", version)
}

func (p *PromptEvolution) GetVersion(key, version string) *PromptVersion {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if versions, ok := p.versions[key]; ok {
		if v, ok := versions[version]; ok {
			return v
		}
	}
	return nil
}

func (p *PromptEvolution) GetAllVersions(key string) map[string]PromptVersion {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]PromptVersion)
	if versions, ok := p.versions[key]; ok {
		for v, pv := range versions {
			result[v] = *pv
		}
	}
	return result
}

func (p *PromptEvolution) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	totalVersions := 0
	activeVersions := 0
	testingVersions := 0

	for _, versions := range p.versions {
		for _, v := range versions {
			totalVersions++
			if v.Status == "active" {
				activeVersions++
			} else if v.Status == "testing" {
				testingVersions++
			}
		}
	}

	return map[string]interface{}{
		"total_versions":     totalVersions,
		"active_versions":    activeVersions,
		"testing_versions":   testingVersions,
		"ab_tests_running":   p.countRunningABTests(),
		"dynamic_threshold":  p.CalculateDynamicThresholdLocked(),
		"history_size":       len(p.history),
	}
}

func (p *PromptEvolution) GetAllVersionsGlobal() map[string]map[string]PromptVersion {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]map[string]PromptVersion)
	for key, versions := range p.versions {
		keyVersions := make(map[string]PromptVersion)
		for v, pv := range versions {
			keyVersions[v] = *pv
		}
		result[key] = keyVersions
	}
	return result
}

func (p *PromptEvolution) countRunningABTests() int {
	count := 0
	for _, test := range p.abTests {
		if test.Status == "running" {
			count++
		}
	}
	return count
}

var _ = fmt.Sprintf

type PromptEvolutionAPI struct {
	evolution *PromptEvolution
}

func NewPromptEvolutionAPI(evolution *PromptEvolution) *PromptEvolutionAPI {
	return &PromptEvolutionAPI{evolution: evolution}
}

func (a *PromptEvolutionAPI) RegisterPrompt(key, version, content string) {
	a.evolution.RegisterPrompt(key, version, content)
}

func (a *PromptEvolutionAPI) RecordRequest(key, version string, latencyMs int64, success bool) {
	a.evolution.RecordRequest(key, version, latencyMs, success)
}

func (a *PromptEvolutionAPI) ShouldUpgrade(key string) (bool, string) {
	return a.evolution.ShouldUpgrade(key)
}

func (a *PromptEvolutionAPI) GetStats() map[string]interface{} {
	return a.evolution.GetStats()
}

func (a *PromptEvolutionAPI) GetVersions(key string) map[string]PromptVersion {
	return a.evolution.GetAllVersions(key)
}