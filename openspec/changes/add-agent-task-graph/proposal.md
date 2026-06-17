## Why

Long and multi-step agent tasks should not be executed inside one oversized Agent Loop prompt. They need checkpointed task graphs with node-level execution, output references, resumability, and status APIs.

## What Changes

- Add task graph, task node, and task output models.
- Add background graph execution service.
- Add APIs to create and inspect long-running task graphs.
- Filter `<environment_details>` before storing task goals.

## Capabilities

### New Capabilities
- `agent-task-graph`: Long-task execution using graph nodes, checkpoints, outputs, and status APIs.

## Impact

- Database schema additions.
- Service layer and admin/user APIs for long-running task graphs.
- AgentService integration for node execution.
