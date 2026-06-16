## Why

Business users often ask follow-up questions such as “卖多少钱？” after asking about orders or a specific employee. The agent should inherit recent query context when safe and ask a concise clarification question when required slots are missing, instead of returning an empty-response fallback.

## What Changes

- Add deterministic query slot extraction for metric, time range, subject, and trend intent.
- Add follow-up query rewriting that inherits missing slots from recent conversation memory.
- Add clarification responses for underspecified sales-data questions.
- Replace empty-model fallback with an actionable clarification message for query-like requests.
- Add tests covering “卖多少钱？” context completion and missing-slot clarification.

## Capabilities

### New Capabilities
- `query-clarification-slots`: Multi-turn sales-query slot filling and clarification.

### Modified Capabilities

## Impact

- Agent fast-path preprocessing.
- Conversation memory usage for query continuation.
- Empty model response fallback for query-like prompts.
- Tests for slot extraction and rewrite behavior.
