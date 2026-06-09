<svelte:head><title>开发者文档 - OpenTether</title></svelte:head>

<article class="prose prose-slate max-w-none">
    <h1>开发者文档</h1>

    <h2>系统架构</h2>
    <div class="not-prose grid grid-cols-1 md:grid-cols-2 gap-4 my-4">
        <div class="card">
            <h3 class="font-semibold text-sm text-primary-700">Go 后端</h3>
            <p class="text-xs text-slate-500 mt-1">
                Fiber HTTP · GORM ORM · JWT 认证 · 嵌入式前端 · Agentic Loop
            </p>
            <p class="text-xs text-slate-400 mt-2">
                main.go · internal/router,handler,service,agent,middleware
            </p>
        </div>
        <div class="card">
            <h3 class="font-semibold text-sm text-emerald-700">
                SvelteKit 前端
            </h3>
            <p class="text-xs text-slate-500 mt-1">
                Svelte · TailwindCSS · Static Adapter
            </p>
            <p class="text-xs text-slate-400 mt-2">
                admin-ui/src/routes/login,(app)
            </p>
        </div>
        <div class="card">
            <h3 class="font-semibold text-sm text-amber-700">向量引擎</h3>
            <p class="text-xs text-slate-500 mt-1">
                TF-IDF（默认）· Embedder 接口 · VectorStore 接口
            </p>
            <p class="text-xs text-slate-400 mt-2">
                internal/embedding,vectorstore
            </p>
        </div>
        <div class="card">
            <h3 class="font-semibold text-sm text-purple-700">IM 接入</h3>
            <p class="text-xs text-slate-500 mt-1">
                企业微信 · 飞书 · 钉钉 · iLink AI · WhatsApp
            </p>
            <p class="text-xs text-slate-400 mt-2">
                internal/im/ · 支持扫码绑定 + 外部系统集成
            </p>
        </div>
        <div class="card">
            <h3 class="font-semibold text-sm text-rose-700">智能体引擎</h3>
            <p class="text-xs text-slate-500 mt-1">
                ReAct Loop · 意图识别 · 工具调用 · 多步推理
            </p>
            <p class="text-xs text-slate-400 mt-2">
                internal/agent/loop.go,engine.go,hub.go
            </p>
        </div>
        <div class="card">
            <h3 class="font-semibold text-sm text-teal-700">独立 Agent</h3>
            <p class="text-xs text-slate-500 mt-1">
                配对码注册 · Pull 模式任务分发 · 心跳监控
            </p>
            <p class="text-xs text-slate-400 mt-2">
                internal/agent/hub.go · cmd/agent/
            </p>
        </div>
    </div>

    <h2>API 认证</h2>
    <p>系统支持两种认证方式：</p>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">方式</th>
                <th class="text-left p-3">场景</th>
                <th class="text-left p-3">Header</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr>
                <td class="p-3 font-medium">JWT Bearer</td>
                <td class="p-3 text-slate-500">管理后台、员工使用</td>
                <td class="p-3 font-mono text-xs"
                    >Authorization: Bearer &lt;token&gt;</td
                >
            </tr>
            <tr>
                <td class="p-3 font-medium">API Key</td>
                <td class="p-3 text-slate-500">外部系统（OA/ERP）集成</td>
                <td class="p-3 font-mono text-xs">X-API-Key: ot_sk_xxxx...</td>
            </tr>
        </tbody>
    </table>

    <h2>API 参考（完整）</h2>
    <p>
        所有 API 以 <code>/api/v1</code> 为前缀。标注 🔐 需 JWT，👑 需 admin 角色，🔑
        需 API Key。
    </p>

    <h3>认证接口（公开）</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">方法</th><th class="text-left p-3"
                    >路径</th
                ><th class="text-left p-3">说明</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-primary-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/auth/login</td
                ><td class="p-3 text-slate-500">登录获取 JWT</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-primary-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/auth/refresh</td
                ><td class="p-3 text-slate-500">刷新 Token</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-primary-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/setup/status</td
                ><td class="p-3 text-slate-500">初始化状态</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-primary-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/setup</td
                ><td class="p-3 text-slate-500">系统初始化</td></tr
            >
        </tbody>
    </table>

    <h3>用户管理 👑</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">方法</th><th class="text-left p-3"
                    >路径</th
                ><th class="text-left p-3">说明</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/users</td
                ><td class="p-3 text-slate-500">用户列表</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/users</td
                ><td class="p-3 text-slate-500">创建用户</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-amber-600">PUT</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/users/:id</td
                ><td class="p-3 text-slate-500">更新用户</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-red-600">DELETE</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/users/:id</td
                ><td class="p-3 text-slate-500">删除用户</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/users/batch</td
                ><td class="p-3 text-slate-500">批量创建</td></tr
            >
        </tbody>
    </table>

    <h3>用户组 👑</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">方法</th><th class="text-left p-3"
                    >路径</th
                ><th class="text-left p-3">说明</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/groups</td
                ><td class="p-3 text-slate-500">用户组列表</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/groups</td
                ><td class="p-3 text-slate-500">创建用户组</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-amber-600">PUT</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/groups/:id</td
                ><td class="p-3 text-slate-500">更新用户组</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-red-600">DELETE</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/groups/:id</td
                ><td class="p-3 text-slate-500">删除用户组</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/groups/:id/members</td
                ><td class="p-3 text-slate-500">添加成员</td></tr
            >
        </tbody>
    </table>

    <h3>LLM 提供商 👑</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">方法</th><th class="text-left p-3"
                    >路径</th
                ><th class="text-left p-3">说明</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/providers</td
                ><td class="p-3 text-slate-500">列表</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/providers</td
                ><td class="p-3 text-slate-500">创建</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-amber-600">PUT</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/providers/:id</td
                ><td class="p-3 text-slate-500">更新</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-red-600">DELETE</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/providers/:id</td
                ><td class="p-3 text-slate-500">删除</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/providers/:id/test</td
                ><td class="p-3 text-slate-500">测试连接</td></tr
            >
        </tbody>
    </table>

    <h3>数据源 👑</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">方法</th><th class="text-left p-3"
                    >路径</th
                ><th class="text-left p-3">说明</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/datasources</td
                ><td class="p-3 text-slate-500">列表</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/datasources</td
                ><td class="p-3 text-slate-500">创建</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-amber-600">PUT</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/datasources/:id</td
                ><td class="p-3 text-slate-500">更新</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-red-600">DELETE</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/datasources/:id</td
                ><td class="p-3 text-slate-500">删除</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/datasources/:id/test</td
                ><td class="p-3 text-slate-500">测试连接</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/datasources/:id/analyze</td
                ><td class="p-3 text-slate-500">AI 分析表结构</td></tr
            >
        </tbody>
    </table>

    <h3>Skills 👑</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">方法</th><th class="text-left p-3"
                    >路径</th
                ><th class="text-left p-3">说明</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/skills</td
                ><td class="p-3 text-slate-500">列表</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/skills</td
                ><td class="p-3 text-slate-500">创建</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-amber-600">PUT</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/skills/:id</td
                ><td class="p-3 text-slate-500">更新</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-red-600">DELETE</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/skills/:id</td
                ><td class="p-3 text-slate-500">删除</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/skills/:id/test</td
                ><td class="p-3 text-slate-500">测试</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/skills/:id/sync</td
                ><td class="p-3 text-slate-500">同步向量</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/skills/from-markdown</td
                ><td class="p-3 text-slate-500">从 MD 创建</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/skills/preview</td
                ><td class="p-3 text-slate-500">预览 MD 解析</td></tr
            >
        </tbody>
    </table>

    <h3>MCP 协议集成</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">方法</th><th class="text-left p-3"
                    >路径</th
                ><th class="text-left p-3">说明</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/mcp/configs</td
                ><td class="p-3 text-slate-500">列表</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/mcp/configs</td
                ><td class="p-3 text-slate-500">创建</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/mcp/configs/:id/start</td
                ><td class="p-3 text-slate-500">启动</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/mcp/configs/:id/stop</td
                ><td class="p-3 text-slate-500">停止</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/mcp/configs/:id/status</td
                ><td class="p-3 text-slate-500">状态</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/mcp/configs/:id/tools</td
                ><td class="p-3 text-slate-500">工具列表</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/mcp/configs/:id/call</td
                ><td class="p-3 text-slate-500">调用工具</td></tr
            >
        </tbody>
    </table>

    <h3>报表 / PDF</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">方法</th><th class="text-left p-3"
                    >路径</th
                ><th class="text-left p-3">说明</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/reports/pdf</td
                ><td class="p-3 text-slate-500">生成 PDF 报表</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/reports/employee-pdf</td
                ><td class="p-3 text-slate-500">员工报表</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/reports/query-pdf</td
                ><td class="p-3 text-slate-500">查询结果 PDF</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/docs/md2pdf</td
                ><td class="p-3 text-slate-500">Markdown→PDF</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/docs/md2pdf/template</td
                ><td class="p-3 text-slate-500">模板转换</td></tr
            >
        </tbody>
    </table>

    <h3>定时任务</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">方法</th><th class="text-left p-3"
                    >路径</th
                ><th class="text-left p-3">说明</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/tasks</td
                ><td class="p-3 text-slate-500">列表</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/tasks</td
                ><td class="p-3 text-slate-500">创建</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-amber-600">PUT</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/tasks/:id</td
                ><td class="p-3 text-slate-500">更新</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-red-600">DELETE</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/tasks/:id</td
                ><td class="p-3 text-slate-500">删除</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/tasks/:id/run</td
                ><td class="p-3 text-slate-500">执行</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/tasks/:id/logs</td
                ><td class="p-3 text-slate-500">日志</td></tr
            >
        </tbody>
    </table>

    <h3>IM 平台管理 👑</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">方法</th><th class="text-left p-3"
                    >路径</th
                ><th class="text-left p-3">说明</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/im/configs</td
                ><td class="p-3 text-slate-500">IM 配置列表</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/im/configs</td
                ><td class="p-3 text-slate-500">创建 IM 配置</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-amber-600">PUT</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/im/configs/:id</td
                ><td class="p-3 text-slate-500">更新</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-red-600">DELETE</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/im/configs/:id</td
                ><td class="p-3 text-slate-500">删除</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/im/configs/:id/test</td
                ><td class="p-3 text-slate-500">测试连接</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/im/pairings</td
                ><td class="p-3 text-slate-500">全员绑定列表</td></tr
            >
        </tbody>
    </table>

    <h3>API 密钥管理</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">权限</th><th class="text-left p-3"
                    >方法</th
                ><th class="text-left p-3">路径</th><th class="text-left p-3"
                    >说明</th
                >
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 text-xs"
                    ><span
                        class="px-1.5 py-0.5 bg-red-50 text-red-600 rounded text-xs"
                        >👑</span
                    ></td
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/api-keys</td
                ><td class="p-3 text-slate-500">创建（返回 raw_key 仅一次）</td
                ></tr
            >
            <tr
                ><td class="p-3 text-xs"
                    ><span
                        class="px-1.5 py-0.5 bg-red-50 text-red-600 rounded text-xs"
                        >👑</span
                    ></td
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/api-keys</td
                ><td class="p-3 text-slate-500">列表（可按 user_id 过滤）</td
                ></tr
            >
            <tr
                ><td class="p-3 text-xs"
                    ><span
                        class="px-1.5 py-0.5 bg-red-50 text-red-600 rounded text-xs"
                        >👑</span
                    ></td
                ><td class="p-3 font-mono text-red-600">DELETE</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/api-keys/:id</td
                ><td class="p-3 text-slate-500">删除</td></tr
            >
            <tr
                ><td class="p-3 text-xs"
                    ><span
                        class="px-1.5 py-0.5 bg-red-50 text-red-600 rounded text-xs"
                        >👑</span
                    ></td
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/api-keys/:id/regenerate</td
                ><td class="p-3 text-slate-500">重新生成</td></tr
            >
            <tr
                ><td class="p-3 text-xs"
                    ><span
                        class="px-1.5 py-0.5 bg-blue-50 text-blue-600 rounded text-xs"
                        >🔐</span
                    ></td
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/user/api-keys</td
                ><td class="p-3 text-slate-500">我的密钥</td></tr
            >
            <tr
                ><td class="p-3 text-xs"
                    ><span
                        class="px-1.5 py-0.5 bg-blue-50 text-blue-600 rounded text-xs"
                        >🔐</span
                    ></td
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/user/api-keys</td
                ><td class="p-3 text-slate-500">自建密钥</td></tr
            >
            <tr
                ><td class="p-3 text-xs"
                    ><span
                        class="px-1.5 py-0.5 bg-blue-50 text-blue-600 rounded text-xs"
                        >🔐</span
                    ></td
                ><td class="p-3 font-mono text-red-600">DELETE</td><td
                    class="p-3 font-mono text-xs">/api/v1/user/api-keys/:id</td
                ><td class="p-3 text-slate-500">删除</td></tr
            >
            <tr
                ><td class="p-3 text-xs"
                    ><span
                        class="px-1.5 py-0.5 bg-blue-50 text-blue-600 rounded text-xs"
                        >🔐</span
                    ></td
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/user/api-keys/:id/regenerate</td
                ><td class="p-3 text-slate-500">重新生成</td></tr
            >
        </tbody>
    </table>

    <h3>IM 自助绑定（扫码流程）🔐</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">方法</th><th class="text-left p-3"
                    >路径</th
                ><th class="text-left p-3">说明</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/user/im/platforms</td
                ><td class="p-3 text-slate-500">可绑定平台列表</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/user/im/request-bind</td
                ><td class="p-3 text-slate-500"
                    >发起绑定（返回 token + 二维码）</td
                ></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/external/im/confirm-bind</td
                ><td class="p-3 text-slate-500">确认绑定（IM 回调用）</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/user/im/bindings</td
                ><td class="p-3 text-slate-500">我的绑定列表</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/user/im/bindings</td
                ><td class="p-3 text-slate-500">手动绑定</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-red-600">DELETE</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/user/im/bindings/:id</td
                ><td class="p-3 text-slate-500">解绑</td></tr
            >
        </tbody>
    </table>

    <h3>外部系统集成 🔑</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">方法</th><th class="text-left p-3"
                    >路径</th
                ><th class="text-left p-3">说明</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/external/bind-im</td
                ><td class="p-3 text-slate-500">OA/ERP 代员工绑定 IM</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/external/users</td
                ><td class="p-3 text-slate-500">查询用户列表</td></tr
            >
        </tbody>
    </table>

    <h3>用户接口 🔐</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">方法</th><th class="text-left p-3"
                    >路径</th
                ><th class="text-left p-3">说明</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/user/chat</td
                ><td class="p-3 text-slate-500">AI 对话（Agentic Loop）</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/user/chat/stream</td
                ><td class="p-3 text-slate-500">SSE 流式对话</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/user/conversations</td
                ><td class="p-3 text-slate-500">对话历史</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/user/conversations/:id</td
                ><td class="p-3 text-slate-500">对话详情</td></tr
            >
        </tbody>
    </table>

    <h3>日志 & 系统 👑</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">方法</th><th class="text-left p-3"
                    >路径</th
                ><th class="text-left p-3">说明</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/logs/audit</td
                ><td class="p-3 text-slate-500">审计日志</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/logs/request</td
                ><td class="p-3 text-slate-500">请求日志</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/admin/logs/export</td
                ><td class="p-3 text-slate-500">导出日志</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-blue-600">GET</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/system/config</td
                ><td class="p-3 text-slate-500">系统配置</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-amber-600">PUT</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/system/config</td
                ><td class="p-3 text-slate-500">更新配置</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-green-600">POST</td><td
                    class="p-3 font-mono text-xs"
                    >/api/v1/admin/system/smtp/test</td
                ><td class="p-3 text-slate-500">测试 SMTP</td></tr
            >
        </tbody>
    </table>

    <h3>IM 回调（公开）</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">路径</th><th class="text-left p-3"
                    >平台</th
                ><th class="text-left p-3">回复方式</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-xs">/api/v1/im/callback/wecom</td
                ><td class="p-3 text-slate-500">企业微信</td><td
                    class="p-3 text-slate-500">异步推送</td
                ></tr
            >
            <tr
                ><td class="p-3 font-mono text-xs"
                    >/api/v1/im/callback/feishu</td
                ><td class="p-3 text-slate-500">飞书</td><td
                    class="p-3 text-slate-500">异步推送</td
                ></tr
            >
            <tr
                ><td class="p-3 font-mono text-xs"
                    >/api/v1/im/callback/dingtalk</td
                ><td class="p-3 text-slate-500">钉钉</td><td
                    class="p-3 text-slate-500">异步推送</td
                ></tr
            >
            <tr
                ><td class="p-3 font-mono text-xs">/api/v1/im/callback/ilink</td
                ><td class="p-3 text-slate-500">iLink AI</td><td
                    class="p-3 text-slate-500"
                    ><span
                        class="px-1.5 py-0.5 bg-amber-50 text-amber-700 rounded-full text-xs"
                        >同步回复</span
                    ></td
                ></tr
            >
            <tr
                ><td class="p-3 font-mono text-xs"
                    >/api/v1/im/callback/personal-wechat</td
                ><td class="p-3 text-slate-500">个人微信</td><td
                    class="p-3 text-slate-500">异步推送</td
                ></tr
            >
            <tr
                ><td class="p-3 font-mono text-xs"
                    >/api/v1/im/callback/whatsapp</td
                ><td class="p-3 text-slate-500">WhatsApp</td><td
                    class="p-3 text-slate-500">异步推送</td
                ></tr
            >
        </tbody>
    </table>

    <h2>智能体引擎 (Agentic Loop)</h2>
    <p>系统核心是一个 <strong>ReAct 模式</strong> 的智能体循环：</p>

    <div class="not-prose my-4">
        <pre
            class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto">用户: "查询销售冠军并生成报表"
    ↓
LLM 推理 → Decision: {'{"action":"tool_call", "tool_name":"text2sql"}'}
    ↓  执行 text2sql → 返回数据
LLM 推理 → Decision: {'{"action":"tool_call", "tool_name":"generate_pdf"}'}
    ↓  执行 generate_pdf → 报表已生成
LLM 推理 → Decision: {'{"action":"final_answer", "final_answer":"..."}'}
    ↓
返回最终答案</pre>
    </div>

    <p>关键参数：</p>
    <ul>
        <li><strong>最大迭代</strong>: 10 步（防止无限循环）</li>
        <li><strong>超时</strong>: 5 分钟</li>
        <li>
            <strong>工具</strong>: chat, text2sql, employee_query, generate_pdf,
            generate_report
        </li>
        <li>
            <strong>决策格式</strong>: JSON
            <code>{"{"}"action":"tool_call","tool_name":"..."{"}"}</code>
        </li>
    </ul>

    <p>配置 <code>internal/agent/loop.go</code> 中的常量即可调整。</p>

    <h2>独立 Agent 部署</h2>
    <p>
        系统支持将智能体引擎独立打包为 Agent 客户端，与 Master
        配对后接受远程任务分发。
    </p>

    <h3>配对流程</h3>
    <ol>
        <li>Agent 客户端启动 → 自动生成配对码 <code>ot_p_xxxx</code></li>
        <li>管理员在 Master UI 输入配对码</li>
        <li>Agent 通过 HTTP 向 Master 注册</li>
        <li>Agent 进入轮询模式：<code>GET /api/v1/agent/tasks/poll</code></li>
        <li>执行完成后 <code>POST /api/v1/agent/tasks/:id/result</code></li>
        <li>每 15s 发送心跳：<code>POST /api/v1/agent/heartbeat</code></li>
    </ol>

    <div class="not-prose card my-4">
        <h3 class="font-semibold text-slate-800 mb-2">通信方式选择</h3>
        <p class="text-sm text-slate-500">
            默认使用 <strong>HTTP Pull</strong>（零依赖）。企业场景 &lt;50 Agent
            节点足够。
            <code>config.yaml</code> 中已预留 Redis/Kafka 升级路径：
        </p>
        <pre
            class="bg-slate-900 text-slate-100 p-3 rounded-lg text-xs mt-2 overflow-x-auto">executor:
  mode: "independent"
  independent:
    queue:
      type: "redis"     # http | redis | kafka
      address: "localhost:6379"</pre>
    </div>

    <h2>自定义 Skill 开发</h2>
    <p>Skill 是系统的核心能力单元。每个 Skill 需定义：</p>
    <ol>
        <li>
            <strong>名称和类型</strong> — chat, text2sql, file_process, report, api_caller
            等
        </li>
        <li><strong>关键词</strong> — JSON 数组，用于意图匹配</li>
        <li><strong>描述</strong> — 用于向量化，帮助语义匹配</li>
        <li><strong>权限</strong> — 可分配给用户组，控制谁可以使用</li>
        <li>
            <strong>数据范围</strong> — all/self/department，控制可见数据边界
        </li>
    </ol>
    <p>
        创建后调用 <code>POST /api/v1/admin/skills/:id/sync</code> 同步向量索引。
    </p>

    <h3>从 Markdown 创建 Skill</h3>
    <p>
        支持从 MD 文件快速创建 Skill，自动解析标题、描述、关键词和 Prompt 模板：
    </p>
    <pre
        class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto"># 销售数据分析
---
skill_type: report
category: sales
keywords: 销售,业绩,排名,数据分析
---

自动分析销售数据并生成排名报表。

```prompt
你是一个销售数据分析专家。请分析以下数据并给出洞察...
```</pre>
    <p>
        上传：<code>POST /api/v1/admin/skills/from-markdown</code
        >（multipart/form-data）
    </p>

    <h2>外部系统集成（OA / ERP）</h2>
    <p>通过 API Key 认证，外部系统可以代用户执行操作：</p>
    <ol>
        <li>管理员创建 API Key → <code>POST /api/v1/admin/api-keys</code></li>
        <li>将返回的 <code>raw_key</code> 配置到 OA/ERP 系统</li>
        <li>OA/ERP 请求时携带 <code>X-API-Key</code> header</li>
        <li>中间件自动验证并注入用户上下文</li>
    </ol>

    <h3>集成示例：代员工绑定 IM</h3>
    <pre
        class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto">POST /api/v1/external/bind-im
X-API-Key: ot_sk_abc123...
{'{\n    "global_user_id": "OA-USER-001",\n    "user_name": "张三",\n    "im_config_id": "uuid-of-wecom-config",\n    "im_user_id": "zhangsan",\n    "im_user_name": "张三"\n}'}</pre>

    <h2>向量引擎扩展</h2>
    <p>默认使用 TF-IDF + 内存存储。可通过实现接口扩展：</p>

    <div class="not-prose space-y-3 my-4">
        <div class="card">
            <h3 class="font-semibold text-sm mb-2">自定义 Embedder</h3>
            <p class="text-xs text-slate-500">
                实现 <code>embedding.Embedder</code> 接口的三个方法：<code
                    >Embed</code
                >
                / <code>Dims</code> / <code>Name</code>，在 <code>init()</code>
                中调用 <code>embedding.Register(name, factory)</code> 注册。
            </p>
        </div>
        <div class="card">
            <h3 class="font-semibold text-sm mb-2">自定义 VectorStore</h3>
            <p class="text-xs text-slate-500">
                实现 <code>vectorstore.Store</code> 接口：<code>Index</code> /
                <code>Search</code>
                / <code>Remove</code> / <code>Count</code> /
                <code>Clear</code>，在 <code>init()</code> 中调用
                <code>vectorstore.RegisterStore(name, factory)</code> 注册。
            </p>
        </div>
    </div>

    <p>配置 <code>config.yaml</code> 即可切换：</p>
    <pre
        class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto">embedding:
  provider: "my-model"
  store: "milvus"</pre>

    <h2>构建部署</h2>

    <h3>生产构建（嵌入模式）</h3>
    <pre
        class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto">build.bat all</pre>
    <p class="text-sm text-slate-500">
        产物：<code>output/opentether-windows-amd64.exe</code
        >（单文件，含全部前端）
    </p>

    <h3>独立 Agent 构建</h3>
    <pre
        class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto">cd cmd/agent
go build -o opentether-agent .</pre>
    <p class="text-sm text-slate-500">独立 Agent 无需嵌入前端，体积更小。</p>

    <h3>开发模式</h3>
    <p class="text-sm text-slate-500">
        <code>build.bat dev</code> + <code>cd admin-ui && npm run dev</code>
    </p>
</article>
