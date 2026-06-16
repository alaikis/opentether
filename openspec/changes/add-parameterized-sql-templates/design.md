## Context

The previous template path loaded approved template memories but did not render or execute the template. It fell back to normal Text2SQL generation. This made templates mostly metadata and allowed dates from memory or LLM inference to drift.

## Goals / Non-Goals

**Goals:**

- Render approved SQL templates with runtime variables.
- Resolve relative date ranges from the current request time.
- Reject templates with unresolved variables.
- Execute rendered SQL through read-only validation, limits, audit, and data-boundary safeguards.
- Mark outputs as `freshness=live_query` with `generated_at`.

**Non-Goals:**

- Build a full template designer UI.
- Cache historical snapshots in this change.
- Support arbitrary template expressions beyond explicit placeholders.

## Decisions

Only approved templates from admin runtime memories are eligible. Variables are quoted as SQL literals. Unresolved variables cause FastPath to skip and fall back to the normal Agent Loop.

## Risks / Trade-offs

- Template matching remains basic and uses optional `intent`; later work should add embedding or LLM planner matching.
- Only explicit date placeholders are supported initially.
