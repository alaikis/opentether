## MODIFIED Requirements

### Requirement: perceiveAndPlan rule-based fallback
The system SHALL provide a rule-based fallback for task planning when LLM-based perceiveAndPlan fails.

#### Scenario: LLM planning succeeds
- **WHEN** LLM returns valid plan JSON
- **THEN** system uses LLM plan as-is

#### Scenario: LLM planning fails
- **WHEN** LLM returns error, empty response, or invalid JSON
- **THEN** system generates plan using keyword-to-tool mapping rules

#### Scenario: Rule fallback tool selection
- **WHEN** rule fallback is triggered
- **THEN** system selects tools based on keyword presence in tool names and descriptions

### Requirement: filterToolsByPlan strict matching
The system SHALL strictly filter tools by plan-declared tool names, removing loose wildcard matching.

#### Scenario: Plan declares specific tools
- **WHEN** plan specifies ["tool_a", "tool_b"]
- **THEN** only tool_a and tool_b remain in available tools

#### Scenario: Plan declares wildcard
- **WHEN** plan specifies ["__all__"]
- **THEN** all tools remain available

#### Scenario: Plan declares non-existent tool
- **WHEN** plan specifies ["tool_nonexistent"]
- **THEN** system logs warning and returns all tools (graceful degradation)

### Requirement: Tool selection feedback recording
The system SHALL record tool selection results (query → selected tools → scores) for future optimization.

#### Scenario: Tool selection completes
- **WHEN** selectRelevantTools returns a result
- **THEN** system records the selection to SkillRuntimeMemory for analysis

#### Scenario: Feedback table full
- **WHEN** feedback records exceed configured limit
- **THEN** system prunes oldest records
