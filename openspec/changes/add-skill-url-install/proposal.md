## Why

当前系统支持通过 Markdown 文件创建 Skill（`POST /skills/from-markdown`）和 AI 生成 Skill（`POST /skills/generate`），但缺少直接从 URL 安装 Skill 的能力。用户需要手动下载 Skill 配置文件再上传，体验不够流畅。

## What Changes

1. **URL 安装 Skill** - 新增 `POST /api/v1/admin/skills/install-from-url` 端点
2. **远程 Skill 解析** - 支持从 URL 获取 Skill 定义（Markdown/JSON 格式）
3. **自动创建** - 解析后自动创建 Skill 记录

## Capabilities

### New Capabilities

- `skill-url-install`: 从 URL 直接安装 Skill

### Modified Capabilities

- 无

## Impact

- 新增 `internal/handler/skill_install_handler.go`
- 修改 `internal/router/api.go` 注册路由
- 复用现有 `fetchTextFromURL` 和 `UploadMarkdownAndCreateSkill` 逻辑
