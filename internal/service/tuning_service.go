package service

import (
	"errors"

	"github.com/alaikis/opentether/internal/models"
	"github.com/alaikis/opentether/internal/tuning"
	"gorm.io/gorm"
)

type TuningService struct {
	db       *gorm.DB
	optimizer tuning.Optimizer
	history   tuning.TuningHistory
	ruleOpt   tuning.RuleBasedOptimizer
	jobs     map[string]*tuning.TuningJob
}

func NewTuningService(db *gorm.DB) *TuningService {
	dataDir := "data/tuning"
	return &TuningService{
		db:       db,
		optimizer: tuning.NewInMemoryOptimizer(),
		history:   tuning.NewInMemoryTuningHistory(dataDir),
		ruleOpt:   tuning.NewInMemoryRuleOptimizer(),
		jobs:      make(map[string]*tuning.TuningJob),
	}
}

func (s *TuningService) CreateJob(job *tuning.TuningJob) error {
	if job == nil || job.ID == "" {
		return errors.New("invalid job")
	}
	s.jobs[job.ID] = job
	return nil
}

func (s *TuningService) StartJob(id string) error {
	job, ok := s.jobs[id]
	if !ok {
		return errors.New("job not found")
	}
	iteration, err := s.optimizer.Optimize(nil, job)
	if err != nil {
		return err
	}
	return s.history.RecordIteration(iteration)
}

func (s *TuningService) ListJobs() ([]*tuning.TuningJob, error) {
	result := make([]*tuning.TuningJob, 0, len(s.jobs))
	for _, j := range s.jobs {
		result = append(result, j)
	}
	return result, nil
}

func (s *TuningService) ListIterations(jobID string) ([]*tuning.TuningIteration, error) {
	return s.history.History(jobID)
}

func (s *TuningService) Rollback(jobID, iterationStr string) error {
	// In a real implementation, parse iterationStr to int
	return nil
}

func (s *TuningService) GetSuggestions() ([]*tuning.ParameterValue, error) {
	return s.ruleOpt.Suggest(nil, map[string]float64{"latency_ms": 6000})
}

func (s *TuningService) GetBestProvider(providers []*models.Provider, metrics map[string]float64) (*models.Provider, error) {
	type providerSuggester interface {
		SuggestProvider([]*models.Provider, map[string]float64) (*models.Provider, error)
	}
	if ps, ok := s.ruleOpt.(providerSuggester); ok {
		return ps.SuggestProvider(providers, metrics)
	}
	if len(providers) == 0 {
		return nil, errors.New("no providers")
	}
	return providers[0], nil
}

func (s *TuningService) RecordIteration(iteration *tuning.TuningIteration) error {
	return s.history.RecordIteration(iteration)
}
