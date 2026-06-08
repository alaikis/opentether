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

| 模块 | 端点 | 说明 |
|------|------|------|
| 认证 | POST /api/v1/auth/login | 用户登录 |
| 用户 | GET/POST /api/v1/admin/users | 用户管理 |
| 用户组 | GET/POST /api/v1/admin/groups | 用户组管理 |
| Provider | GET/POST /api/v1/admin/providers | LLM Provider |
| 数据源 | GET/POST /api/v1/admin/datasources | 数据源管理 |
| Skills | GET/POST /api/v1/admin/skills | Skills 配置 |
| 任务 | GET/POST /api/v1/admin/tasks | 定时任务 |
| IM | GET/POST /api/v1/admin/im/configs | IM 配置 |
| 对话 | POST /api/v1/user/chat | AI 对话 |

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