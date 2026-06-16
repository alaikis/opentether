package llm

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	defaultMaxRetries  = 3
	defaultBackoffBase = 500 * time.Millisecond
	defaultBackoffMax  = 8 * time.Second
)

func isRetryableHTTPError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	retryable := []string{
		"timeout", "connection refused", "connection reset",
		"eof", "broken pipe", "no route to host",
		"too many requests", "rate limit", "429",
		"internal server error", "500", "502", "503", "504",
		"service unavailable", "temporarily unavailable",
	}
	for _, kw := range retryable {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}

func backoffDuration(attempt int) time.Duration {
	d := defaultBackoffBase * time.Duration(math.Pow(2, float64(attempt)))
	if d > defaultBackoffMax {
		d = defaultBackoffMax
	}
	return d
}

// ChatCompletionWithRetry wraps a Client.ChatCompletion with exponential backoff for transient errors.
// Returns the response, or the last error after exhausting retries.
func ChatCompletionWithRetry(client Client, ctx context.Context, req ChatRequest, maxAttempts int) (*ChatResponse, error) {
	if maxAttempts <= 0 {
		maxAttempts = defaultMaxRetries
	}
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err := client.ChatCompletion(ctx, req)
		if err == nil {
			if resp != nil && strings.TrimSpace(resp.Content) != "" {
				return resp, nil
			}
			lastErr = fmt.Errorf("LLM 返回空响应")
		} else if isRetryableHTTPError(err) {
			lastErr = err
			if attempt < maxAttempts-1 {
				d := backoffDuration(attempt)
				time.Sleep(d)
				continue
			}
		} else {
			return nil, err
		}
	}
	return nil, lastErr
}
