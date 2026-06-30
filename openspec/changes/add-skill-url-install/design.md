## Context

当前 Skill 创建方式：
1. `POST /skills/from-markdown` — 上传 Markdown 文件
2. `POST /skills/generate` — AI 生成

缺少直接从 URL 安装 Skill 的能力。

## Goals / Non-Goals

**Goals:**
1. 支持从 URL 获取 Skill 定义并安装
2. 支持 Markdown 和 JSON 格式
3. 自动解析并创建 Skill

**Non-Goals:**
- 不修改现有 Skill 核心逻辑
- 不引入新的存储依赖

## Decisions

### 1. URL 获取策略

**决定**: 复用现有 `fetchTextFromURL` 函数

**理由**: 
- 已在 `service.go` 中实现
- 支持 HTTP GET，超时 8 秒

### 2. 格式支持

**决定**: 支持 Markdown 格式（复用现有解析器）

**理由**: 
- 现有 `UploadMarkdownAndCreateSkill` 已支持 Markdown
- JSON 格式可后续扩展
