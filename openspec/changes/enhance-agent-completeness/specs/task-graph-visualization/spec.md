## ADDED Requirements

### Requirement: Task graph stores dependency relationships
The system SHALL maintain an in-memory directed acyclic graph (DAG) representing task dependencies, where each node represents a subtask and edges represent execution order constraints.

#### Scenario: Task created with dependencies
- **WHEN** a new task is created with explicit depends_on field
- **THEN** the system SHALL add edges from dependent tasks to the new task in the graph

#### Scenario: Task execution respects dependencies
- **WHEN** the scheduler attempts to execute a task
- **THEN** the system SHALL delay execution until all predecessor tasks have completed

### Requirement: Real-time task status streaming
The system SHALL emit task status change events through Server-Sent Events (SSE) for real-time UI updates.

#### Scenario: Task status changes
- **WHEN** a task transitions from pending to running
- **THEN** the system SHALL emit an event with the new status to all connected clients

#### Scenario: Client subscribes to task updates
- **WHEN** a client connects to /api/v1/tasks/{id}/stream
- **THEN** the system SHALL send all subsequent status changes for that task

### Requirement: Task graph can be queried
The system SHALL provide APIs to retrieve the current task graph state and execution history.

#### Scenario: Query task graph
- **WHEN** user calls GET /api/v1/tasks/graph
- **THEN** the system SHALL return JSON representation of the DAG with node states

#### Scenario: Query task execution history
- **WHEN** user calls GET /api/v1/tasks/{id}/history
- **THEN** the system SHALL return execution timeline including start time, end time, and retry count