## 1. System Observer 模块

- [x] 1.1 创建 `internal/agent/observer.go` 文件，定义 Observer 结构体
- [x] 1.2 实现 Skill 使用统计收集功能
- [x] 1.3 实现 DataSource 健康度监控功能
- [x] 1.4 实现 LLM 质量指标收集功能
- [x] 1.5 实现用户行为画像功能
- [x] 1.6 添加 HTTP handler 暴露监控端点

## 2. Self-Learning 实时增强

- [x] 2.1 修改 `self_learning.go`，首次失败即生成低置信度提示
- [x] 2.2 添加失败类型分类（语法/语义/超时/权限）
- [x] 2.3 实现 LLM 质量下降预警机制
- [x] 2.4 添加动态置信度累积逻辑

## 3. 统一 FeedbackLoop 框架

- [x] 3.1 创建 `internal/agent/feedback_loop.go` 文件
- [x] 3.2 实现 Observation 收集 Channel
- [x] 3.3 实现后台 Insight 处理逻辑
- [x] 3.4 实现 UpgradeAction 分发机制
- [x] 3.5 集成到 AgentEngine

## 4. Prompt 版本自动化

- [x] 4.1 创建 `internal/agent/prompt_evolution.go` 文件
- [x] 4.2 实现动态阈值计算逻辑
- [x] 4.3 实现 Prompt 版本管理
- [x] 4.4 添加 A/B 测试框架
- [x] 4.5 实现自动版本选择

## 5. Soul 进化增强

- [x] 5.1 修改 `memory.go` 中的 LettaMemory 结构
- [x] 5.2 添加隐式反馈收集（会话长度、追问率）
- [x] 5.3 实现多维度 Persona（专业级别、回复风格）
- [x] 5.4 实现累积式 + 阈值触发进化逻辑
- [x] 5.5 添加手动 Override 支持