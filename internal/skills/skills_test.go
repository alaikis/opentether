package skills

import (
	"testing"
)

func TestSkillRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	// Test registering a skill
	skill := &TestSkill{name: "test_skill", description: "A test skill"}
	err := registry.Register(skill)
	if err != nil {
		t.Fatalf("Failed to register skill: %v", err)
	}

	// Test duplicate registration
	err = registry.Register(skill)
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}
}

func TestSkillRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	skill := &TestSkill{name: "test_skill", description: "A test skill"}
	registry.Register(skill)

	// Test getting existing skill
	retrieved, err := registry.Get("test_skill")
	if err != nil {
		t.Fatalf("Failed to get skill: %v", err)
	}
	if retrieved.Name() != "test_skill" {
		t.Errorf("Expected skill name 'test_skill', got '%s'", retrieved.Name())
	}

	// Test getting non-existent skill
	_, err = registry.Get("non_existent")
	if err == nil {
		t.Error("Expected error for non-existent skill")
	}
}

func TestSkillRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// Register multiple skills
	registry.Register(&TestSkill{name: "skill1", description: "Skill 1"})
	registry.Register(&TestSkill{name: "skill2", description: "Skill 2"})

	skills := registry.List()
	if len(skills) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(skills))
	}
}

func TestSkillRegistry_Enabled(t *testing.T) {
	registry := NewRegistry()

	// Register enabled skill
	enabledSkill := &TestSkill{name: "enabled", description: "Enabled", enabled: true}
	registry.Register(enabledSkill)

	// Register disabled skill
	disabledSkill := &TestSkill{name: "disabled", description: "Disabled", enabled: false}
	registry.Register(disabledSkill)

	// List enabled should only return enabled
	enabled := registry.Enabled()
	if len(enabled) != 1 {
		t.Errorf("Expected 1 enabled skill, got %d", len(enabled))
	}

	// List all should return both
	all := registry.List()
	if len(all) != 2 {
		t.Errorf("Expected 2 skills total, got %d", len(all))
	}
}

func TestSkillExecutionContext(t *testing.T) {
	ctx := &ExecutionContext{
		UserID:    "user123",
		UserName:  "Test User",
		Input:     "Hello",
		SkillName: "chat",
		Params:    map[string]interface{}{},
	}

	if ctx.UserID != "user123" {
		t.Errorf("Expected userID 'user123', got '%s'", ctx.UserID)
	}
	if ctx.Input != "Hello" {
		t.Errorf("Expected input 'Hello', got '%s'", ctx.Input)
	}
}

func TestBuiltinSkills(t *testing.T) {
	// Test that builtin skills are registered
	registry := NewRegistry()
	RegisterBuiltinSkills(registry)

	// Should have at least the basic skills
	skills := registry.Enabled()
	if len(skills) == 0 {
		t.Error("Expected at least one built-in skill")
	}

	// Check for key skills
	skillNames := make(map[string]bool)
	for _, s := range skills {
		skillNames[s.Name()] = true
	}

	expectedSkills := []string{"chat", "text2sql", "file_process", "report", "api_caller"}
	for _, name := range expectedSkills {
		if !skillNames[name] {
			t.Logf("Warning: skill '%s' not found", name)
		}
	}
}

// TestSkill is a mock skill for testing
type TestSkill struct {
	name        string
	description string
	enabled     bool
	result      string
	err         error
}

func (s *TestSkill) Name() string        { return s.name }
func (s *TestSkill) Description() string { return s.description }
func (s *TestSkill) Type() string        { return "test" }
func (s *TestSkill) Enabled() bool       { return s.enabled }
func (s *TestSkill) Execute(ctx *ExecutionContext) (*Result, error) {
	return &Result{Output: s.result}, s.err
}
func (s *TestSkill) Schema() string { return "{}" }
