## ADDED Requirements

### Requirement: Follow-up query completion
The system SHALL complete short follow-up sales queries using recent conversation context when the inherited slots are unambiguous.

#### Scenario: User asks follow-up amount
- **WHEN** the previous query established a subject or time range and the user asks “卖多少钱？”
- **THEN** the system rewrites the query to include the inherited context and the sales amount metric

### Requirement: Query clarification
The system SHALL ask for missing information when a sales query cannot be safely completed.

#### Scenario: Missing query context
- **WHEN** the user asks “卖多少钱？” with no usable recent context
- **THEN** the system asks which person/group and time range should be queried

### Requirement: Empty response recovery
The system SHALL prefer actionable clarification over generic empty-response fallback for query-like user prompts.

#### Scenario: Model returns empty for query-like prompt
- **WHEN** model response is empty and the prompt looks like a sales query
- **THEN** the system returns a clarification prompt instead of only asking the user to retry
