package agent

import (
	"sync"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

type SkillPlanner struct {
	db       *gorm.DB
	skills   *SkillManager
	mu       sync.RWMutex
	planCache map[string]string
}

func NewSkillPlanner(db *gorm.DB, skills *SkillManager) *SkillPlanner {
	return &SkillPlanner{
		db:         db,
		skills:     skills,
		planCache:  make(map[string]string),
	}
}

func (p *SkillPlanner) PlanSkills(plan *MultiTaskPlan) {
	if plan == nil || len(plan.SubTasks) == 0 {
		return
	}
	if p.skills == nil {
		return
	}
	skills, err := p.skills.ListEnabledSkills()
	if err != nil || len(skills) == 0 {
		return
	}
	for i := range plan.SubTasks {
		sub := &plan.SubTasks[i]
		if sub.SkillUsed != "" {
			continue
		}
		matched := p.matchSkill(sub.Query, skills)
		if matched != nil {
			sub.SkillUsed = matched.SkillType
		}
	}
}

func (p *SkillPlanner) matchSkill(query string, skills []models.Skill) *models.Skill {
	if len(skills) == 0 {
		return nil
	}
	threshold := 0.3
	bestSkill := skills[0]
	bestScore := float64(0)
	for _, skill := range skills {
		score := p.computeSimilarity(query, skill)
		if score > bestScore {
			bestScore = score
			bestSkill = skill
		}
	}
	if bestScore >= threshold {
		return &bestSkill
	}
	return nil
}

func (p *SkillPlanner) computeSimilarity(query string, skill models.Skill) float64 {
	queryLower := toLowerASCII(query)
	skillText := toLowerASCII(skill.Name + " " + skill.Description + " " + skill.Keywords)
	wordsQ := tokenize(queryLower)
	wordsS := tokenize(skillText)
	if len(wordsQ) == 0 || len(wordsS) == 0 {
		return 0
	}
	intersection := 0
	for _, w := range wordsQ {
		if len(w) < 2 {
			continue
		}
		if stringInSlice(w, wordsS) {
			intersection++
		}
	}
	return float64(intersection * 2) / float64(len(wordsQ)+len(wordsS))
}

func toLowerASCII(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}
