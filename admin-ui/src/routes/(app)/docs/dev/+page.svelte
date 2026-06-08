<svelte:head><title>开发者文档 - OpenTether</title></svelte:head>

<article class="prose prose-slate max-w-none">
    <h1>开发者文档</h1>

    <h2>系统架构</h2>
    <div class="not-prose grid grid-cols-1 md:grid-cols-2 gap-4 my-4">
        <div class="card">
            <h3 class="font-semibold text-sm text-primary-700">Go 后端</h3>
            <p class="text-xs text-slate-500 mt-1">
                Fiber HTTP · GORM ORM · JWT 认证 · 嵌入式前端
            </p>
            <p class="text-xs text-slate-400 mt-2">
                cmd/main.go · internal/router,handler,service,agent
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
                企业微信 · 飞书 · 钉钉 · iLink AI
            </p>
            <p class="text-xs text-slate-400 mt-2">
                internal/im/wechat,feishu,dingtalk,ilink.go
            </p>
        </div>
    </div>

    <h2>API 参考</h2>
    <p>
        所有 API 以 <code>/api/v1</code> 为前缀，需在请求头中携带 JWT Token（除公开接口外）。
    </p>

    <h3>认证接口（公开）</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50"
                ><th class="text-left p-3">方法</th><th class="text-left p-3"
                    >路径</th
                ><th class="text-left p-3">说明</th></tr
            ></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-primary-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/auth/login</td
                ><td class="p-3 text-slate-500">登录获取 Token</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-primary-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/auth/refresh</td
                ><td class="p-3 text-slate-500">刷新 Token</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-primary-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/setup/status</td
                ><td class="p-3 text-slate-500">检查初始化状态</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-primary-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/setup</td
                ><td class="p-3 text-slate-500">系统初始化</td></tr
            >
        </tbody>
    </table>

    <h3>管理员接口（需 JWT + admin 角色）</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50"
                ><th class="text-left p-3">路径前缀</th><th
                    class="text-left p-3">说明</th
                ></tr
            ></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-xs">/api/v1/admin/users</td><td
                    class="p-3 text-slate-500">用户管理 CRUD</td
                ></tr
            >
            <tr
                ><td class="p-3 font-mono text-xs">/api/v1/admin/groups</td><td
                    class="p-3 text-slate-500">用户组管理</td
                ></tr
            >
            <tr
                ><td class="p-3 font-mono text-xs">/api/v1/admin/providers</td
                ><td class="p-3 text-slate-500">LLM 提供商配置</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-xs">/api/v1/admin/datasources</td
                ><td class="p-3 text-slate-500">数据源管理</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-xs">/api/v1/admin/skills</td><td
                    class="p-3 text-slate-500">Skill 管理 + 向量同步</td
                ></tr
            >
            <tr
                ><td class="p-3 font-mono text-xs">/api/v1/admin/im/configs</td
                ><td class="p-3 text-slate-500">IM 平台配置</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-xs">/api/v1/admin/im/pairings</td
                ><td class="p-3 text-slate-500">查看全员 IM 绑定</td></tr
            >
        </tbody>
    </table>

    <h3>用户接口（需 JWT）</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50"
                ><th class="text-left p-3">方法</th><th class="text-left p-3"
                    >路径</th
                ><th class="text-left p-3">说明</th></tr
            ></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-primary-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/user/chat</td
                ><td class="p-3 text-slate-500">发送消息给 AI Agent</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-primary-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/user/chat/stream</td
                ><td class="p-3 text-slate-500">SSE 流式对话</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-primary-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/user/conversations</td
                ><td class="p-3 text-slate-500">获取对话历史</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-primary-600">GET</td><td
                    class="p-3 font-mono text-xs">/api/v1/user/im/bindings</td
                ><td class="p-3 text-slate-500">查看我的 IM 绑定</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-primary-600">POST</td><td
                    class="p-3 font-mono text-xs">/api/v1/user/im/bindings</td
                ><td class="p-3 text-slate-500">绑定 IM 账号</td></tr
            >
        </tbody>
    </table>

    <h3>IM 回调（公开，由平台调用）</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50"
                ><th class="text-left p-3">路径</th><th class="text-left p-3"
                    >平台</th
                ></tr
            ></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-mono text-xs">/api/v1/im/callback/wecom</td
                ><td class="p-3 text-slate-500">企业微信</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-xs"
                    >/api/v1/im/callback/feishu</td
                ><td class="p-3 text-slate-500">飞书</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-xs"
                    >/api/v1/im/callback/dingtalk</td
                ><td class="p-3 text-slate-500">钉钉</td></tr
            >
            <tr
                ><td class="p-3 font-mono text-xs">/api/v1/im/callback/ilink</td
                ><td class="p-3 text-slate-500">iLink AI（同步回复）</td></tr
            >
        </tbody>
    </table>

    <h2>自定义 Skill 开发</h2>
    <p>Skill 是系统的核心能力单元。每个 Skill 需定义：</p>
    <ol>
        <li>
            <strong>名称和类型</strong> — chat, text2sql, file_process, report, api_caller
            等
        </li>
        <li>
            <strong>关键词</strong> — JSON 数组，用于意图匹配。例：<code
                >["查询","统计","销售额","排名"]</code
            >
        </li>
        <li><strong>描述</strong> — 用于向量化，帮助语义匹配</li>
        <li><strong>权限</strong> — 可分配给用户组，控制谁可以使用</li>
        <li>
            <strong>数据范围</strong> — all/self/department，控制可见数据边界
        </li>
    </ol>

    <p>
        创建后调用 <code>POST /api/v1/admin/skills/:id/sync</code> 同步向量索引，使其参与语义匹配。
    </p>

    <h2>构建部署</h2>

    <h3>生产构建（嵌入模式）</h3>
    <pre
        class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto">build.bat all</pre>
    <p class="text-sm text-slate-500">
        产物：<code>output/opentether-windows-amd64.exe</code
        >（单文件，含全部前端）
    </p>

    <h3>开发模式</h3>
    <p class="text-sm text-slate-500">
        <code>build.bat dev</code> + <code>cd admin-ui && npm run dev</code>
    </p>

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
</article>
