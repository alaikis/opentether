# Enhanced Soul Evolution Specification

## Overview

Enhanced user profile (Soul) evolution system with implicit feedback and multi-dimensional persona.

## Features

### 1. Implicit Feedback Collection
- Session length tracking
- Follow-up question rate measurement
- Query complexity analysis
- Satisfaction signal detection (via follow-up patterns)

### 2. Multi-Dimensional Persona
- **Professional Level**: Beginner/Intermediate/Expert based on query complexity
- **Reply Style**: Concise/Detailed/Technical based on interaction patterns
- **Preferred Metrics**: Frequently used metrics tracking
- **Language Preference**: Dynamic language preference

### 3. Evolution Triggers
- **Basic Evolution**: Query count >= 5
- **Deep Evolution**: Query count >= 20
- **Style Evolution**: Detected on significant pattern change

### 4. Manual Override
- Admin can lock certain dimensions
- User can manually set preferences
- Override takes priority over evolved values

## Data Model

```go
type UserSoul struct {
    UserID             string
    Persona            string // AI assistant description
    Human              string // User description
    LanguagePreference string
    
    // Enhanced dimensions
    ProfessionalLevel  string // beginner/intermediate/expert
    ReplyStyle         string // concise/detailed/technical
    PreferredMetrics   []string
    Confidence         map[string]float64
    
    // Implicit feedback
    AvgSessionLength   int     // queries per session
    FollowUpRate       float64 // ratio of follow-up questions
    QueryComplexity    float64 // average query complexity score
    
    UpdatedAt          time.Time
}

type EvolutionEvent struct {
    ID          string
    UserID      string
    Dimension   string
    OldValue    string
    NewValue    string
    Trigger     string // basic_evolution/deep_evolution/style_change/manual
    Confidence  float64
    CreatedAt   time.Time
}
```

## Evolution Logic

```go
func (m *LettaMemory) evolveSoulAdvanced(userID string, p *UserSoul) {
    totalQueries := p.getQueryCount()
    
    // Basic evolution: query count >= 5
    if totalQueries >= 5 {
        p.evolveFromQueryHistory()
    }
    
    // Deep evolution: query count >= 20
    if totalQueries >= 20 {
        p.evolveFromImplicitFeedback()
    }
    
    // Style evolution: detect significant pattern change
    if p.detectStyleShift() {
        p.evolveReplyStyle()
    }
}
```