## Why

当前项目的 Observe 机制和自我升级机制存在明显不足：缺乏系统级的监控能力（Skill 使用统计、数据源健康度、LLM 质量指标），Self-Learning 响应迟缓（仅在失败 3 次后才触发），反馈循环分散在不同模块中，Prompt 版本管理缺乏自动化。这些问题导致系统无法及时发现和响应运行时异常，影响用户体验和系统稳定性。

## What Changes

1. **新增 SystemObserver 模块** - 统一的系统级监控中心
   - Skill 使用统计（调用次数、成功率、平均延迟）
   - 数据源健康度监控（连接延迟、查询超时率）
   - LLM 质量指标（响应延迟、错误率、token 消耗）
   - 用户行为画像（活跃时段、查询类型分布）

2. **增强 SelfLearning 机制** - 实时失败分类和即时反馈
   - 首次失败即生成低置信度提示，后续累积提升
   - 失败类型分类（语法错误、语义错误、超时、权限问题）
   - LLM 质量下降预警

3. **统一 FeedbackLoop 框架** - 整合分散的反馈机制
   - 统一的 Observation 收集通道
   - 内置的 Insight 处理pipeline
   - 可扩展的 UpgradeAction 机制

4. **Prompt 版本自动化** - 动态阈值和 A/B 测试支持
   - 基于历史趋势的动态阈值调整
   - Prompt 版本 A/B 测试框架
   - 自动选择最优版本

5. **增强 Soul 进化** - 隐式反馈和多维度 Persona
   - 隐式反馈收集（会话长度、追问率）
   - 多维度 Persona 进化（专业程度、回复风格）

## Capabilities

### New Capabilities
- `system-observer`: 系统级监控组件，收集 Skill/DS/LLM 运行时指标
- `feedback-loop`: 统一反馈循环框架，整合所有反馈源
- `dynamic-prompt-evolution`: Prompt 版本动态演进机制
- `enhanced-soul-evolution`: 增强的用户画像进化系统

### Modified Capabilities
- `self-learning`: 增强为实时失败分类和即时反馈机制

## Impact

- **新增文件**: `internal/agent/observer.go`, `internal/agent/feedback_loop.go`, `internal/agent/prompt_evolution.go`
- **修改文件**: `internal/agent/self_learning.go`, `internal/agent/memory.go`
- **依赖**: 无新外部依赖
- **API**: 新增 `/api/v1/admin/observer/*` 监控端点