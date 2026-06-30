# OpenTether Enterprise AI Agent

企业级智能体系统 | Enterprise AI Agent System

## 简介

OpenTether 是一个基于 Go + Fiber + GORM 开发的企业级智能体系统，支持多用户、记忆隔离、Skills 配置、IM 集成等功能。

## 特性

- 🤖 **智能对话** - 支持多轮对话，上下文记忆
- 🔐 **企业级权限** - RBAC/ABAC/行级权限控制
- 💬 **多平台 IM** - 企业微信、飞书、钉钉、WhatsApp
- 📊 **智能数据分析** - Text2SQL、自动 Schema 分析、报告生成
- ⚙️ **Skills 配置** - 可配置技能、可扩展执行器
- 📅 **定时任务** - 支持脚本/Python 执行
- 🌐 **多语言** - 支持中文、英文等多语言
- 🔄 **自动更新** - GitHub 版本检测与自动更新
- 📦 **嵌入式部署** - 单二进制部署

## 技术栈

- **后端**: Go, Fiber, GORM
- **数据库**: SQLite (内置), MySQL, PostgreSQL
- **前端**: SvelteKit + shadcn-svelte
- **Embedding**: bge-m3 (本地 ONNX)

## 快速开始

### 方式一：下载预编译二进制

```bash
# 下载对应平台的二进制文件
./wisehoof
```

### 方式二：从源码构建

```bash
# 克隆项目
git clone https://github.com/company/wisehoof.git
cd wisehoof

# 构建
go build -o wisehoof .

# 运行
./wisehoof
```

## 配置

复制并编辑 `config.yaml`:

```yaml
server:
  port: 8080
  mode: "development"

database:
  type: "sqlite"
  name: "data/wisehoof.db"

security:
  jwt:
    secret: "your-secret-key"
    expire: "24h"
```

## API 文档

启动服务后访问:
- API: http://localhost:8080/api/v1
- Admin UI: http://localhost:8080/admin
- Health: http://localhost:8080/health

## 主要 API 端点

### 认证与用户

| 方法 | 端点 | 说明 |
|------|------|------|
| POST | /api/v1/auth/login | 用户登录 |
| POST | /api/v1/auth/refresh | 刷新 Token |
| GET | /api/v1/admin/users | 用户列表 |
| POST | /api/v1/admin/users | 创建用户 |
| PUT | /api/v1/admin/users/:id | 更新用户 |
| DELETE | /api/v1/admin/users/:id | 删除用户 |

### 智能体与对话

| 方法 | 端点 | 说明 |
|------|------|------|
| POST | /api/v1/user/chat | AI 对话 |
| POST | /api/v1/user/chat/stream | 流式对话 |
| GET | /api/v1/user/conversations | 对话列表 |
| GET | /api/v1/admin/agent-task-graphs | 任务图列表 |
| POST | /api/v1/admin/agent-task-graphs | 创建任务图 |
| GET | /api/v1/admin/agent-task-graphs/:id | 任务图详情 |
| GET | /api/v1/prompts/versions | Prompt 版本列表 |

### RAG 检索增强

| 方法 | 端点 | 说明 |
|------|------|------|
| POST | /api/v1/admin/rag/ingest | 文档入库 |
| GET | /api/v1/admin/rag/documents | 文档列表 |
| DELETE | /api/v1/admin/rag/documents/:id | 删除文档 |
| GET | /api/v1/admin/rag/retrieve | 语义检索 |
| GET | /api/v1/admin/rag/search | 关键词检索 |

### 审计与日志

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | /api/v1/admin/audit/logs | 审计日志 |
| POST | /api/v1/admin/audit/logs/export | 导出审计日志 |
| POST | /api/v1/admin/audit/logs/export/s3 | 导出到 S3 |
| GET | /api/v1/admin/audit/compliance/reports | 合规报告 |
| POST | /api/v1/admin/audit/compliance/reports | 生成合规报告 |

### 自动调优

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | /api/v1/admin/tuning/jobs | 调优任务列表 |
| POST | /api/v1/admin/tuning/jobs | 创建调优任务 |
| POST | /api/v1/admin/tuning/jobs/:id/start | 启动调优 |
| GET | /api/v1/admin/tuning/jobs/:id/iterations | 调优迭代历史 |
| POST | /api/v1/admin/tuning/jobs/:id/rollback | 回滚调优 |
| GET | /api/v1/admin/tuning/suggestions | 参数建议 |

### 可观测性

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | /api/v1/admin/metrics/definitions | 指标定义列表 |
| POST | /api/v1/admin/metrics/definitions | 创建指标定义 |
| GET | /api/v1/admin/metrics/:id/query | 查询指标数据 |
| GET | /api/v1/admin/alerts/rules | 告警规则列表 |
| POST | /api/v1/admin/alerts/rules | 创建告警规则 |
| GET | /api/v1/admin/alerts/events | 告警事件列表 |
| POST | /api/v1/admin/alerts/events/:id/ack | 确认告警 |

### MCP 工具生态

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | /api/v1/admin/mcp/configs | MCP 配置列表 |
| POST | /api/v1/admin/mcp/configs | 创建 MCP 配置 |
| POST | /api/v1/admin/mcp/configs/:id/start | 启动 MCP 服务器 |
| POST | /api/v1/admin/mcp/configs/:id/stop | 停止 MCP 服务器 |
| GET | /api/v1/admin/mcp/configs/:id/tools | 列出工具 |
| POST | /api/v1/admin/mcp/configs/:id/call | 调用工具 |
| GET | /api/v1/admin/mcp/servers | 服务器列表 |
| POST | /api/v1/admin/mcp/servers | 注册服务器 |

### 分布式 Hub

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | /api/v1/admin/hub/nodes | 节点列表 |
| POST | /api/v1/admin/hub/nodes | 注册节点 |
| DELETE | /api/v1/admin/hub/nodes/:id | 注销节点 |
| GET | /api/v1/admin/hub/tasks | 任务列表 |
| POST | /api/v1/admin/hub/tasks | 提交任务 |
| GET | /api/v1/admin/hub/tasks/:id/results | 任务结果 |
| POST | /api/v1/admin/hub/tasks/:id/cancel | 取消任务 |

## 开发

### 前端开发

```bash
cd admin-ui
npm install
npm run dev
```

### 后端开发

```bash
# 运行
go run main.go

# 测试
go test ./...
```

## 部署

### Docker

```bash
docker-compose up -d
```

### Kubernetes

```bash
kubectl apply -f k8s/
```

## 目录结构

```
wisehoof/
├── main.go              # 入口文件
├── config.yaml          # 配置文件
├── internal/
│   ├── config/          # 配置加载
│   ├── database/        # 数据库
│   ├── handler/         # HTTP 处理器
│   ├── middleware/      # 中间件
│   ├── models/          # 数据模型
│   ├── router/          # 路由
│   ├── service/         # 业务逻辑
│   ├── agent/           # AI Agent 核心
│   ├── skills/          # Skills 系统
│   ├── scheduler/       # 调度系统
│   └── im/              # IM 集成
├── admin-ui/            # 前端项目
│   └── build/           # 构建产物 (嵌入)
├── data/                # 数据目录
├── logs/                # 日志目录
└── models/              # 模型文件
    └── embedding/       # Embedding 模型
```

## License

MIT License