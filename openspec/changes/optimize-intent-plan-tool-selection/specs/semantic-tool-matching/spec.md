## ADDED Requirements

### Requirement: Semantic tool matching
The system SHALL provide embedding-based semantic matching for tool selection as an enhancement to keyword-based matching.

#### Scenario: Semantic match for synonymous queries
- **WHEN** user submits "sales revenue" and a tool description contains "销售额"
- **THEN** system uses embedding similarity to match the tool despite different wording

#### Scenario: Semantic match fallback
- **WHEN** embedding service is unavailable
- **THEN** system falls back to keyword-based matching without error

#### Scenario: Semantic match limit
- **WHEN** tool list exceeds 20 items
- **THEN** system only embeds top-k candidates from keyword pre-filter
