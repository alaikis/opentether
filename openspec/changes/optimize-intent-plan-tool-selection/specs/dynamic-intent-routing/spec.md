## ADDED Requirements

### Requirement: Dynamic intent routing rules
The system SHALL support database-configured intent-to-skill routing rules as a fallback and enhancement to hardcoded skillMap.

#### Scenario: Database rule matches
- **WHEN** a query matches a rule in skill_intent_rules table
- **THEN** system routes to the configured skill_type

#### Scenario: No database rule
- **WHEN** no matching rule exists in skill_intent_rules
- **THEN** system falls back to hardcoded skillMap

#### Scenario: Rule priority
- **WHEN** multiple rules match with different priorities
- **THEN** system selects the rule with highest priority

## MODIFIED Requirements

### Requirement: Intent recognition with user memory
The system SHALL inject user long-term memory (historical skill usage preferences) into intent recognition.

#### Scenario: User has history
- **WHEN** user has previously used skill_text2sql multiple times
- **THEN** intent recognition boosts text2sql-related candidates for that user

#### Scenario: New user
- **WHEN** user has no history
- **THEN** intent recognition works without memory injection
