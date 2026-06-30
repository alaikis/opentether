## Context

当前项目采用分散式的观察和学习机制：
- `TemplateWatcher` 负责模板文件监控
- `SelfLearning` 处理失败模式学习
- `ExperienceManager` 管理经验复用
- `Memory/LettaMemory` 处理用户记忆

这些组件各自独立运行，缺乏统一的协调机制，导致：
1. 系统级监控能力不足
2. 反馈响应延迟（失败 3 次才触发）
3. 无法跨组件关联分析问题
4. Prompt 版本管理缺乏自动化

## Goals / Non-Goals

**Goals:**
- 构建统一的 SystemObserver 监控中心，覆盖 Skill/DS/LLM 运行时指标
- 实现实时失败分类和即时反馈机制（首次失败即触发）
- 建立 FeedbackLoop 框架，整合所有反馈源
- 实现 Prompt 版本动态演进（A/B 测试 + 动态阈值）
- 增强 Soul 进化机制，支持隐式反馈和多维度 Persona

**Non-Goals:**
- 不修改现有 Skill 核心逻辑
- 不改变现有的权限和认证机制
- 不引入新的外部依赖

## Decisions

### 1. Observer 架构选择: Pull + Push 混合模式

**选择**: 定期 Pull 统计 + 事件驱动 Push 告警

**理由**:
- 轻量级指标（调用次数、延迟）使用定期 Pull 聚合
- 异常事件（失败、超时）使用 Push 实时处理
- 平衡了性能和实时性

### 2. FeedbackLoop 消息模型: Channel + Async Processing

**选择**: Go Channel 作为收集通道，后台 goroutine 异步处理

**理由**:
- 非阻塞收集，不影响主请求流程
- 天然支持背压（backpressure）
- 与 Go 的并发模型一致

### 3. Prompt 版本演进: 保守升级策略

**选择**: 达到阈值后生成新版本，但不自动切换

**理由**:
- 避免自动切换导致线上问题
- 保留人工审核环节
- 支持 A/B 测试验证

### 4. Soul 进化触发条件: 累积式 + 阈值触发

**选择**: 查询次数 ≥5 触发基础进化，≥20 触发深度进化

**理由**:
- 避免数据不足时的过度拟合
- 区分"了解用户"和"深度理解用户"两个阶段

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Observer 引入额外性能开销 | 使用无锁数据结构 + 批量写入 |
| FeedbackLoop 消息积压 | 设置 Channel 缓冲区上限 + 降级策略 |
| Prompt A/B 测试复杂度 | 仅在明确场景启用，控制流量比例 |
| Soul 进化方向偏差 | 保留手动调整接口 |

## Migration Plan

1. **Phase 1**: 新增 Observer 模块（向后兼容）
2. **Phase 2**: 增强 SelfLearning（渐进式）
3. **Phase 3**: 集成 FeedbackLoop（可选启用）
4. **Phase 4**: 启用 Prompt 演进（灰度）
5. **Phase 5**: 启用增强 Soul（灰量）

## Open Questions

- [ ] 是否需要将 Observer 数据暴露给外部监控系统（Prometheus）?
- [ ] FeedbackLoop 是否需要支持消息持久化?
- [ ] Prompt A/B 测试的流量分配策略?