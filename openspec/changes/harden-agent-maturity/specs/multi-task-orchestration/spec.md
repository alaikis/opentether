## ADDED Requirements

### Requirement: Multi-task orchestration in chat flow
The system SHALL detect multi-part user queries and execute sub-tasks in parallel, then aggregate results into a unified response.

#### Scenario: User asks multi-part question
- **WHEN** user submits "查询北京和上海的销售额"
- **THEN** system splits into two sub-tasks, executes both in parallel, and returns combined results

#### Scenario: User asks single-part question
- **WHEN** user submits "查询北京销售额"
- **THEN** system executes via existing single-task flow without multi-task overhead

#### Scenario: Sub-task execution failure
- **WHEN** one sub-task fails during parallel execution
- **THEN** other sub-tasks continue, and the final response includes both successful results and error notice

## ADDED Requirements

### Requirement: Tree task detection
The system SHALL detect tree-structured tasks (parent-child dependencies) and execute them respecting dependency order.

#### Scenario: Tree task with dependencies
- **WHEN** user submits query that maps to tasks with parent-child relationships
- **THEN** system executes parent first, then children after parent completes

#### Scenario: No tree detected
- **WHEN** query cannot be structured as tree tasks
- **THEN** system falls back to flat parallel execution or single-task execution
