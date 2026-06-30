# Feedback Loop Specification

## Overview

Unified feedback collection and processing framework that integrates all feedback sources in the system.

## Features

### 1. Observation Collection
- Non-blocking channel-based collection
- Support for various observation types:
  - `query_result`: Query execution result (success/failure)
  - `skill_usage`: Skill execution result
  - `llm_latency`: LLM call latency measurement
  - `user_feedback`: Explicit user feedback
  - `implicit_signal`: Implicit signals (session length, follow-up rate)

### 2. Insight Processing
- Background goroutine for async processing
- Configurable batch size and interval
- Backpressure protection with bounded channel

### 3. Upgrade Action Framework
- Pluggable action handlers
- Support for:
  - `generate_improvement_hint`: Create improvement suggestions
  - `update_soul`: Evolve user persona
  - `update_prompt_version`: Trigger prompt version update
  - `alert`: Send notifications

## Data Model

```go
type Observation struct {
    ID        string
    Type      string
    Payload   map[string]interface{}
    Timestamp time.Time
    UserID    string
    TraceID   string
}

type Insight struct {
    ID         string
    ObsIDs     []string
    Category   string
    Confidence float64
    Content    string
    CreatedAt  time.Time
}

type UpgradeAction struct {
    ID          string
    Type        string
    Payload     map[string]interface{}
    Status      string
    CreatedAt   time.Time
    ExecutedAt  *time.Time
}
```

## Processing Pipeline

1. Observation → Channel (non-blocking)
2. BatchCollector → Time/Size triggered batches
3. InsightProcessor → Generate insights
4. ActionDispatcher → Execute upgrade actions