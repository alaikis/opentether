package tuning

import (
	"context"
	"time"
)

type ParameterType string

const (
	ParameterTypeContinuous ParameterType = "continuous"
	ParameterTypeDiscrete   ParameterType = "discrete"
	ParameterTypeCategorical ParameterType = "categorical"
)

type Parameter struct {
	Name        string        `json:"name"`
	Type        ParameterType `json:"type"`
	Min         *float64      `json:"min,omitempty"`
	Max         *float64      `json:"max,omitempty"`
	Step        *float64      `json:"step,omitempty"`
	Options     []string      `json:"options,omitempty"`
	Default     interface{}   `json:"default"`
}

type ParameterValue struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

type TuningObjective struct {
	Name string `json:"name"`
	Goal string `json:"goal"`
}

type TuningJob struct {
	ID          string            `json:"id"`
	Parameters  []*Parameter      `json:"parameters"`
	Objective   *TuningObjective  `json:"objective"`
	Status      string            `json:"status"`
	BestValue   *ParameterValue   `json:"best_value,omitempty"`
	BestScore   float64           `json:"best_score"`
	Iterations  int               `json:"iterations"`
	MaxIterations int             `json:"max_iterations"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type TuningIteration struct {
	ID         string            `json:"id"`
	JobID      string            `json:"job_id"`
	Values     []*ParameterValue `json:"values"`
	Score      float64           `json:"score"`
	Metrics    map[string]float64 `json:"metrics"`
	Iteration  int               `json:"iteration"`
	CreatedAt  time.Time         `json:"created_at"`
}

type Optimizer interface {
	Optimize(ctx context.Context, job *TuningJob) (*TuningIteration, error)
	Update(ctx context.Context, jobID string, iteration *TuningIteration) error
	Best(ctx context.Context, jobID string) (*TuningIteration, error)
}

type RuleBasedOptimizer interface {
	Suggest(params []*ParameterValue, metrics map[string]float64) ([]*ParameterValue, error)
}

type TuningHistory interface {
	RecordIteration(iteration *TuningIteration) error
	History(jobID string) ([]*TuningIteration, error)
	Rollback(jobID string, iteration int) (*ParameterValue, error)
}
