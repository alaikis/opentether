## Context

The current Agent Loop works for short and medium tasks. Very long jobs require splitting execution into nodes and saving checkpoints. This change introduces the foundation for Task Graph execution.

## Goals / Non-Goals

**Goals:**

- Store task graph, nodes, and outputs.
- Execute graphs asynchronously.
- Save node status, summaries, errors, timestamps, and outputs.
- Expose create/status APIs.

**Non-Goals:**

- Full visual graph editor.
- Advanced DAG planner.
- Distributed worker queue.

## Design

Models:

- `agent_task_graphs`
- `agent_task_nodes`
- `agent_task_outputs`

Execution:

- Create graph from goal.
- Persist default plan nodes.
- Run asynchronously.
- Each node calls AgentService with a constrained query.
- Store output summary and raw output JSON.
- Graph status becomes completed/failed.

Future work can replace the default plan generator with an LLM planner that emits a DAG.
