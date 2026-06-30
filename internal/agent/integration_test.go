package agent

import (
	"context"
	"testing"
	"time"
)

func TestVerificationLoop_Basic(t *testing.T) {
	v := NewVerificationLoop(nil)

	result := v.VerifyResult("text2sql", "SELECT * FROM users", "Error: syntax error")
	if result.Passed {
		t.Error("Expected verification to fail for syntax error")
	}
	if !result.ShouldRetry {
		t.Error("Expected retry flag to be set")
	}
}

func TestVerificationLoop_SuccessCase(t *testing.T) {
	v := NewVerificationLoop(nil)

	output := `[{"id": 1, "name": "John"}, {"id": 2, "name": "Jane"}]`
	result := v.VerifyResult("text2sql", "SELECT * FROM users", output)

	if !result.Passed {
		t.Errorf("Expected verification to pass, got: %+v", result.Checks)
	}
}

func TestVerificationLoop_RetryStrategy(t *testing.T) {
	v := NewVerificationLoop(nil)

	delay := v.GetRetryDelay("text2sql", "timeout", 0)
	if delay < 500 {
		t.Errorf("Expected base delay >= 500, got %d", delay)
	}

	shouldRetry := v.ShouldRetry("text2sql", "timeout", 2)
	if !shouldRetry {
		t.Error("Expected should retry on timeout")
	}

	shouldRetry = v.ShouldRetry("text2sql", "timeout", 5)
	if shouldRetry {
		t.Error("Expected no retry after max attempts")
	}
}

func TestVerificationLoop_FailureAnalysis(t *testing.T) {
	v := NewVerificationLoop(nil)

	v.VerifyResult("text2sql", "invalid query", "Error: timeout")
	v.VerifyResult("text2sql", "bad query", "Error: timeout")

	stats := v.GetStats()
	if stats["total_verifications"].(int) < 2 {
		t.Errorf("Expected at least 2 verifications, got %+v", stats)
	}
}

func TestVerificationLoop_SuggestOptimization(t *testing.T) {
	v := NewVerificationLoop(nil)

	v.VerifyResult("text2sql", "query1", "Error: timeout")
	v.VerifyResult("text2sql", "query2", "Error: timeout")

	suggestion := v.SuggestOptimization("text2sql")
	if suggestion == "" {
		t.Error("Expected non-empty suggestion")
	}

	t.Logf("Suggestion: %s", suggestion)
}

func TestDistributedCache_Local(t *testing.T) {
	cache := NewDistributedCache(true)

	ctx := context.Background()

	cache.Set(ctx, "test_key", "test_value", 10*time.Second)
	val, found := cache.Get(ctx, "test_key")

	if !found {
		t.Error("Expected to find cached value")
	}
	if val != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", val)
	}

	cache.Delete(ctx, "test_key")
	val, found = cache.Get(ctx, "test_key")

	if found {
		t.Error("Expected value to be deleted")
	}
}

func TestDistributedCache_Fallback(t *testing.T) {
	cache := NewDistributedCache(false)

	ctx := context.Background()

	cache.Set(ctx, "fallback_key", "fallback_value", 10*time.Second)
	val, found := cache.Get(ctx, "fallback_key")

	if !found {
		t.Error("Expected to find fallback value")
	}
	if val != "fallback_value" {
		t.Errorf("Expected 'fallback_value', got '%s'", val)
	}
}

func TestDistributedCache_Multi(t *testing.T) {
	cache := NewDistributedCache(true)

	ctx := context.Background()

	cache.Set(ctx, "key1", "val1", 10*time.Second)
	cache.Set(ctx, "key2", "val2", 10*time.Second)
	cache.Set(ctx, "key3", "val3", 10*time.Second)

	results := cache.GetMulti(ctx, []string{"key1", "key2", "missing"})

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	if results["key1"] != "val1" {
		t.Errorf("Expected val1, got %s", results["key1"])
	}
	if results["key2"] != "val2" {
		t.Errorf("Expected val2, got %s", results["key2"])
	}
}

func TestSemanticCache(t *testing.T) {
	cache := NewSemanticCache()

	ctx := context.Background()

	testQuery := "show me sales data"
	testResponse := "Sales data: $10000"

	err := cache.Set(ctx, testQuery, testResponse, 5*time.Minute)
	if err != nil {
		t.Errorf("Failed to set cache: %v", err)
	}

	val, found := cache.Get(ctx, testQuery)
	if !found {
		t.Error("Expected to find cached semantic query")
	}
	if val != testResponse {
		t.Errorf("Expected response match, got %s", val)
	}
}

func TestFallbackCache_Eviction(t *testing.T) {
	cache := NewFallbackCache(3)

	cache.Set("k1", "v1", time.Hour)
	cache.Set("k2", "v2", time.Hour)
	cache.Set("k3", "v3", time.Hour)

	if _, found := cache.Get("k1"); !found {
		t.Error("Expected k1 to exist")
	}

	cache.Set("k4", "v4", time.Hour)

	if _, found := cache.Get("k4"); !found {
		t.Error("Expected k4 to exist after eviction")
	}
}

func TestVerifyRule_Patterns(t *testing.T) {
	v := NewVerificationLoop(nil)

	tests := []struct {
		checkType string
		pattern   string
		output    string
		expected  bool
	}{
		{"error_marker", "(?i)error", "Some error occurred", false},
		{"error_marker", "(?i)error", "Success!", true},
		{"empty_result", "无数据", "无数据返回", false},
		{"empty_result", "无数据", "Found 100 records", true},
		{"json_valid", "^\\s*[\\[\\{]", "{\"key\": \"value\"}", true},
		{"json_valid", "^\\s*[\\[\\{]", "not json", false},
	}

	for _, tt := range tests {
		rule := &VerifyRule{
			ID:         "test",
			CheckType:  tt.checkType,
			Pattern:    tt.pattern,
			IsRequired: true,
		}

		result := v.executeCheck(rule, tt.output)
		if result.Passed != tt.expected {
			t.Errorf("CheckType %s: expected %v, got %v", tt.checkType, tt.expected, result.Passed)
		}
	}
}

func TestRetryConditions(t *testing.T) {
	v := NewVerificationLoop(nil)

	delay1 := v.GetRetryDelay("api_caller", "rate limit exceeded", 0)
	delay2 := v.GetRetryDelay("api_caller", "rate limit exceeded", 1)

	if delay2 <= delay1 {
		t.Errorf("Expected exponential backoff: delay1=%d, delay2=%d", delay1, delay2)
	}

	shouldRetry := v.ShouldRetry("api_caller", "429 Too Many Requests", 1)
	if !shouldRetry {
		t.Error("Expected retry for 429 status")
	}
}

func TestMetricsRegistry(t *testing.T) {
	registry := NewMetricsRegistry()

	registry.IncCounter("test_counter")
	if registry.GetCounter("test_counter") != 1 {
		t.Error("Expected counter to be 1")
	}

	registry.Add("test_counter", 5)
	if registry.GetCounter("test_counter") != 6 {
		t.Errorf("Expected counter to be 6, got %d", registry.GetCounter("test_counter"))
	}

	registry.RecordValue("test_histogram", 10.0)
	registry.RecordValue("test_histogram", 20.0)
	registry.RecordValue("test_histogram", 30.0)

	count, avg, p50, p95, max := registry.GetHistogram("test_histogram")
	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
	if avg != 20.0 {
		t.Errorf("Expected avg 20.0, got %f", avg)
	}
	if p50 != 20.0 {
		t.Errorf("Expected p50 20.0, got %f", p50)
	}
	if max != 30.0 {
		t.Errorf("Expected max 30.0, got %f", max)
	}
	_ = p95
}

func TestTelemetryDirect(t *testing.T) {
	telemetry := &Telemetry{
		registry: NewMetricsRegistry(),
	}

	telemetry.IncCounter("test_metric", Label{Name: "env", Value: "test"})
	if telemetry.registry.GetCounter("test_metric", Label{Name: "env", Value: "test"}) != 1 {
		t.Error("Expected metric to be recorded")
	}

	telemetry.RecordLatency("operation", 100.0)
	telemetry.RecordLatency("operation", 200.0)

	count, avg, _, _, _ := telemetry.registry.GetHistogram("operation")
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
	if avg != 150.0 {
		t.Errorf("Expected avg 150.0, got %f", avg)
	}

	metrics := telemetry.GetMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be non-nil")
	}
}

func TestTelemetryGlobal(t *testing.T) {
	InitTelemetry(TelemetryConfig{
		Enabled:     true,
		ServiceName: "test-service",
	})

	tel := GetTelemetry()
	if tel == nil {
		t.Error("Expected non-nil telemetry")
	}

	tel.IncCounter("init_test")
	if tel.registry.GetCounter("init_test") != 1 {
		t.Error("Expected counter to be 1 after init")
	}
}