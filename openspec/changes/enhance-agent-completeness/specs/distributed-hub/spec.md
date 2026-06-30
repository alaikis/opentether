## ADDED Requirements

### Requirement: Nodes auto-discover each other
Agent instances SHALL use gossip protocol to automatically discover other nodes in the cluster.

#### Scenario: Node starts and discovers peers
- **WHEN** a new node starts with cluster enabled
- **THEN** the node SHALL discover existing peers within 10 seconds via UDP broadcast

#### Scenario: New node joins cluster
- **WHEN** an existing node receives a discovery packet from a new node
- **THEN** the existing node SHALL add the new node to its peer list and share cluster state

### Requirement: Tasks can be distributed across nodes
The system SHALL support distributing tasks to different agent nodes for parallel execution.

#### Scenario: Submit task to cluster
- **WHEN** user submits a task with distribute=true
- **THEN** the system SHALL select the least loaded node and dispatch the task

#### Scenario: Node becomes unavailable during task execution
- **WHEN** a running node stops responding (heartbeat timeout)
- **THEN** the system SHALL detect the failure within 30 seconds and reschedule pending tasks

### Requirement: Node health monitoring
The system SHALL continuously monitor node health through heartbeat messages.

#### Scenario: Heartbeat timeout
- **WHEN** a node misses 3 consecutive heartbeat intervals
- **THEN** the system SHALL mark the node as unhealthy and redistribute its tasks

#### Scenario: Node recovers
- **WHEN** an unhealthy node resumes heartbeat
- **THEN** the system SHALL mark the node as healthy and resume sending new tasks

### Requirement: Distributed cache consistency
The system SHALL maintain eventual consistency for shared cached data across nodes.

#### Scenario: Update shared cache
- **WHEN** any node updates a cache entry tagged as distributed
- **THEN** the change SHALL propagate to all nodes within 5 seconds