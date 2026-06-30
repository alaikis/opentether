# System Observer Specification

## Overview

Unified system-level monitoring component that collects runtime metrics for Skill, DataSource, and LLM usage.

## Features

### 1. Skill Usage Statistics
- Track total calls, successful calls, failed calls per skill
- Record average latency per skill
- Track last used timestamp per skill

### 2. DataSource Health Monitoring
- Monitor connection latency
- Track query timeout rate
- Record connection pool utilization

### 3. LLM Quality Metrics
- Track response latency
- Record error rate
- Monitor token consumption

### 4. User Behavior Profiling
- Track active periods
- Record query type distribution
- Build user activity patterns

## Data Model

```go
type SkillStat struct {
    SkillID       string
    TotalCalls    int
    SuccessCalls  int
    FailedCalls   int
    AvgLatencyMs  float64
    LastUsedAt    time.Time
}

type DataSourceHealth struct {
    DataSourceID     string
    AvgLatencyMs     float64
    TimeoutRate      float64
    ConnectionErrors int
    LastChecked      time.Time
}

type LLMQualityMetrics struct {
    ProviderID    string
    AvgLatencyMs  float64
    ErrorRate     float64
    TotalTokens   int
    LastCallAt    time.Time
}
```

## API Endpoints

- `GET /api/v1/admin/observer/skills` - Get skill usage statistics
- `GET /api/v1/admin/observer/datasources` - Get datasource health
- `GET /api/v1/admin/observer/llm` - Get LLM quality metrics
- `GET /api/v1/admin/observer/users/:id/activity` - Get user activity profile