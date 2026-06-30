## MODIFIED Requirements

### Requirement: Tool output completion detection
The system SHALL detect complete tool results using structured field inspection and semantic heuristics, not hardcoded string markers.

#### Scenario: Structured SQL tool output
- **WHEN** tool output contains JSON with columns, rows, and row_count fields
- **THEN** system marks result as complete and triggers early-stop

#### Scenario: Unstructured complete output
- **WHEN** tool output is long (>2000 chars) and contains result indicators but no hardcoded markers
- **THEN** system uses heuristic detection to determine completeness

#### Scenario: Truncated output
- **WHEN** tool output contains truncation markers ("被截断", "超出", "不完整")
- **THEN** system does NOT trigger early-stop and continues loop
