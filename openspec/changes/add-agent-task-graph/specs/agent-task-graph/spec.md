## ADDED Requirements

### Requirement: Task graph persistence
The system SHALL persist long-running agent tasks as task graphs with nodes and outputs.

#### Scenario: Create graph
- **WHEN** a user creates a long task
- **THEN** the system stores an agent task graph and pending nodes

### Requirement: Background execution
The system SHALL execute task graphs asynchronously.

#### Scenario: Graph runs
- **WHEN** a graph is created
- **THEN** the system starts execution without blocking the request

### Requirement: Status inspection
The system SHALL expose task graph status, nodes, and outputs.

#### Scenario: User checks graph status
- **WHEN** a user requests a graph by ID
- **THEN** the system returns graph status, node checkpoints, and outputs
