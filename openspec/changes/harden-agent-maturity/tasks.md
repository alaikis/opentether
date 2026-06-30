## 1. Multi-task orchestration integration

- [x] 1.1 Wire ExecuteMultiTaskPlan into services.go ChatStream and Chat entry points
- [x] 1.2 Implement detectTaskTree to support tree-structured task detection
- [x] 1.3 Implement extractContextWords and extractMetricFromQuery for nested sub-task generation
- [x] 1.4 Add max parallelism limit (5) and max recursion depth (3) configuration

## 2. Agent core robustness

- [x] 2.1 Replace isCompleteToolResult with structured field + heuristic detection
- [x] 2.2 Enhance summarizeCompleteParallelResults to handle mixed success/failure
- [x] 2.3 Add LLM decision quality monitoring in retryLoopDecision
- [x] 2.4 Implement fallback strategy when LLM quality threshold exceeded

## 3. Self-learning validation

- [x] 3.1 Add SelfLearning.Backtest method for historical pattern validation
- [x] 3.2 Add runtime quality metrics collection (retry count, parse errors)
- [x] 3.3 Expose validation endpoints for admin monitoring

## 4. Testing and verification

- [x] 4.1 Add unit tests for multi-task orchestration integration
- [x] 4.2 Add unit tests for new isCompleteToolResult logic
- [x] 4.3 Add unit tests for self-learning backtest
- [x] 4.4 Run existing test suite to verify no regressions
