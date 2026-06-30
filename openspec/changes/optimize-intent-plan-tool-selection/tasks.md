## 1. Core agent improvements

- [x] 1.1 Add rule-based fallback to perceiveAndPlan
- [x] 1.2 Implement strict matching in filterToolsByPlan
- [x] 1.3 Inject user long-term memory into recognizeIntent
- [x] 1.4 Add tool selection feedback recording to selectRelevantTools

## 2. Semantic matching module

- [x] 2.1 Create semantic_match.go with embedding-based tool matching
- [x] 2.2 Integrate semantic matching as optional enhancement in selectRelevantTools

## 3. Dynamic intent routing

- [x] 3.1 Create skill_intent_rules database table and model
- [x] 3.2 Add admin CRUD endpoints for skill_intent_rules
- [x] 3.3 Modify planExecution to query database skillMap with hardcoded fallback

## 4. Testing and verification

- [x] 4.1 Add unit tests for perceiveAndPlan fallback
- [x] 4.2 Add unit tests for strict filterToolsByPlan
- [x] 4.3 Add unit tests for semantic matching
- [x] 4.4 Run full test suite to verify no regressions
