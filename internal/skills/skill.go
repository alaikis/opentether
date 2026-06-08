package skills

import (
	"errors"
	"fmt"
	"sync"
)

// Skill is the interface that all skills must implement
type Skill interface {
	// Name returns the unique name of the skill
	Name() string

	// Description returns a human-readable description
	Description() string

	// Type returns the type of skill (chat, text2sql, file_process, report, api_caller, etc.)
	Type() string

	// Enabled returns whether the skill is enabled
	Enabled() bool

	// Execute runs the skill with the given context
	Execute(ctx *ExecutionContext) (*Result, error)

	// Schema returns the JSON schema for skill parameters
	Schema() string
}

// ExecutionContext contains the context for skill execution
type ExecutionContext struct {
	UserID      string
	UserName    string
	Department  string
	Input       string
	SkillName   string
	Params      map[string]interface{}
	Context     map[string]interface{} // Additional context like conversation history
	DataSourceID string                // For text2sql skill
}

// Result represents the result of skill execution
type Result struct {
	Output      string                 `json:"output"`       // Text output
	Data        map[string]interface{} `json:"data"`         // Structured data
	SkillUsed   string                 `json:"skill_used"`   // Name of skill used
	TokenCount  int                    `json:"token_count"`  // Token usage
	Error       string                 `json:"error,omitempty"` // Error message if failed
}

// Registry manages skill registration and discovery
type Registry struct {
	skills map[string]Skill
	mu     sync.RWMutex
}

// NewRegistry creates a new skill registry
func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]Skill),
	}
}

// Register adds a skill to the registry
func (r *Registry) Register(skill Skill) error {
	if skill == nil {
		return errors.New("skill cannot be nil")
	}

	name := skill.Name()
	if name == "" {
		return errors.New("skill name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[name]; exists {
		return fmt.Errorf("skill already registered: %s", name)
	}

	r.skills[name] = skill
	return nil
}

// Get retrieves a skill by name
func (r *Registry) Get(name string) (Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skill, exists := r.skills[name]
	if !exists {
		return nil, fmt.Errorf("skill not found: %s", name)
	}

	return skill, nil
}

// List returns all registered skills
func (r *Registry) List() []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]Skill, 0, len(r.skills))
	for _, skill := range r.skills {
		skills = append(skills, skill)
	}

	return skills
}

// Enabled returns all enabled skills
func (r *Registry) Enabled() []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]Skill, 0)
	for _, skill := range r.skills {
		if skill.Enabled() {
			skills = append(skills, skill)
		}
	}

	return skills
}

// Unregister removes a skill from the registry
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[name]; !exists {
		return fmt.Errorf("skill not found: %s", name)
	}

	delete(r.skills, name)
	return nil
}

// Global registry for built-in skills
var globalRegistry = NewRegistry()

// DefaultRegistry returns the global skill registry
func DefaultRegistry() *Registry {
	return globalRegistry
}

// RegisterGlobal registers a skill in the global registry
func RegisterGlobal(skill Skill) error {
	return globalRegistry.Register(skill)
}

// GetGlobal retrieves a skill from the global registry
func GetGlobal(name string) (Skill, error) {
	return globalRegistry.Get(name)
}

// ListGlobal lists all skills in the global registry
func ListGlobal() []Skill {
	return globalRegistry.List()
}

// EnabledGlobal returns all enabled skills in the global registry
func EnabledGlobal() []Skill {
	return globalRegistry.Enabled()
}
