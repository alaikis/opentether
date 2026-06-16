## Context

The system already has conversation memory and Agent Loop support for `clarify`, but simple sales queries may go through fast paths or Text2SQL before structured clarification happens. We need a lightweight deterministic layer for common business query slots.

## Goals / Non-Goals

**Goals:**

- Recognize short follow-up sales questions.
- Inherit missing metric/time/subject from recent conversation memory when available.
- Ask a clarification question when the prompt is too ambiguous.
- Keep behavior deterministic and local before invoking an LLM.

**Non-Goals:**

- Build a full natural-language dialogue state machine.
- Add a visual slot-filling UI.
- Replace Agent Loop clarification for complex tool operations.

## Decisions

Add a local query clarification module in the agent package. It extracts simple sales slots from the current message and recent messages. If a short follow-up has enough inherited context, it rewrites the message into an explicit query. If required slots remain missing, it returns a clarification ChatResponse.

## Risks / Trade-offs

- Conservative matching may miss some phrasing but avoids incorrect automatic assumptions.
- Inheritance is limited to recent same-conversation messages to avoid stale context.
