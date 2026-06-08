<svelte:head><title>用户指南 - OpenTether</title></svelte:head>

<article class="prose prose-slate max-w-none">
    <h1>用户指南</h1>

    <h2>快速开始</h2>
    <ol>
        <li>
            <strong>系统初始化</strong> — 首次访问自动跳转 <code>/setup</code> 向导，配置数据库和管理员账号
        </li>
        <li>
            <strong>登录管理后台</strong> — 使用管理员账号登录
            <code>/admin/login</code>
        </li>
        <li>
            <strong>配置 LLM 提供商</strong> — 在「LLM 提供商」页面添加 OpenAI / Azure
            等 API 配置
        </li>
        <li>
            <strong>配置数据源</strong> — 在「数据源管理」页面添加数据库连接
        </li>
        <li>
            <strong>创建 Skills</strong> — 在「Skills 配置」页面创建 AI 技能，设置关键词和权限
        </li>
        <li>
            <strong>添加用户</strong> — 在「用户管理」页面创建员工账号，分配角色和用户组
        </li>
    </ol>

    <div class="not-prose card my-6">
        <h3 class="font-semibold text-slate-800 mb-2">📋 初始化向导字段说明</h3>
        <table class="w-full text-sm">
            <tr class="border-b border-slate-100"
                ><td class="py-2 pr-4 font-medium text-slate-600">数据库类型</td
                ><td class="py-2 text-slate-500"
                    >SQLite（推荐，零配置）/ MySQL / PostgreSQL</td
                ></tr
            >
            <tr class="border-b border-slate-100"
                ><td class="py-2 pr-4 font-medium text-slate-600">数据库名</td
                ><td class="py-2 text-slate-500"
                    >如 <code>opentether</code>，SQLite 会自动保存为
                    <code>data/opentether.db</code></td
                ></tr
            >
            <tr class="border-b border-slate-100"
                ><td class="py-2 pr-4 font-medium text-slate-600"
                    >管理员用户名</td
                ><td class="py-2 text-slate-500">后台登录账号</td></tr
            >
            <tr
                ><td class="py-2 pr-4 font-medium text-slate-600">管理员密码</td
                ><td class="py-2 text-slate-500">建议使用强密码</td></tr
            >
        </table>
    </div>

    <h2>IM 绑定（员工操作）</h2>
    <p>
        管理员配置好 IM 平台后，每个员工需要在系统中绑定自己的 IM 账号，才能通过
        IM 使用 AI 能力。
    </p>

    <h3>绑定步骤</h3>
    <ol>
        <li>登录管理后台</li>
        <li>进入「用户中心」→ 查看 IM 绑定状态</li>
        <li>选择已配置的 IM 平台，输入自己的 IM 账号 ID</li>
        <li>提交绑定，管理员审核后生效</li>
    </ol>

    <div class="not-prose card my-6">
        <h3 class="font-semibold text-slate-800 mb-2">
            各平台 UserID 获取方式
        </h3>
        <table class="w-full text-sm">
            <thead
                ><tr class="border-b border-slate-200"
                    ><th class="text-left py-2 font-medium text-slate-500"
                        >平台</th
                    ><th class="text-left py-2 font-medium text-slate-500"
                        >UserID 格式</th
                    ><th class="text-left py-2 font-medium text-slate-500"
                        >获取方式</th
                    ></tr
                ></thead
            >
            <tbody>
                <tr class="border-b border-slate-100"
                    ><td class="py-2 font-medium">企业微信</td><td
                        class="py-2 text-slate-500 font-mono text-xs"
                        >ZhangSan</td
                    ><td class="py-2 text-slate-500"
                        >企业微信管理后台 → 通讯录</td
                    ></tr
                >
                <tr class="border-b border-slate-100"
                    ><td class="py-2 font-medium">飞书</td><td
                        class="py-2 text-slate-500 font-mono text-xs">ou_xxx</td
                    ><td class="py-2 text-slate-500">飞书管理后台 → 用户详情</td
                    ></tr
                >
                <tr class="border-b border-slate-100"
                    ><td class="py-2 font-medium">钉钉</td><td
                        class="py-2 text-slate-500 font-mono text-xs"
                        >manager123</td
                    ><td class="py-2 text-slate-500">钉钉管理后台 → 员工信息</td
                    ></tr
                >
                <tr
                    ><td class="py-2 font-medium">iLink（公众号）</td><td
                        class="py-2 text-slate-500 font-mono text-xs"
                        >oXXXX_xxx</td
                    ><td class="py-2 text-slate-500"
                        >公众号后台 → 用户 OpenID（首次互动后可见）</td
                    ></tr
                >
            </tbody>
        </table>
    </div>

    <h2>使用 AI 对话</h2>
    <ol>
        <li>登录后进入「用户中心」→「AI 对话」</li>
        <li>在输入框中输入问题，系统自动匹配最合适的 Skill</li>
        <li>支持的功能取决于管理员分配的权限和 Skill</li>
    </ol>

    <h2>权限说明</h2>
    <div class="not-prose grid grid-cols-1 md:grid-cols-3 gap-4 my-4">
        <div class="card">
            <h3 class="font-semibold text-primary-700 text-sm">管理员</h3>
            <p class="text-xs text-slate-500 mt-1">
                全部权限：管理用户、配置系统、查看日志、使用所有 Skill
            </p>
        </div>
        <div class="card">
            <h3 class="font-semibold text-emerald-700 text-sm">普通用户</h3>
            <p class="text-xs text-slate-500 mt-1">
                受限权限：使用被分配的 Skill，访问范围内数据
            </p>
        </div>
        <div class="card">
            <h3 class="font-semibold text-slate-500 text-sm">访客</h3>
            <p class="text-xs text-slate-500 mt-1">
                只读权限：仅可查看被授权的信息
            </p>
        </div>
    </div>

    <h2>常见问题</h2>

    <h3>Q: 提示"超出系统支持范围"怎么办？</h3>
    <p>
        系统只在已注册 Skill 范围内工作。这说明你的问题没有匹配到任何 Skill
        的关键词。请联系管理员添加对应的 Skill 或扩展关键词。
    </p>

    <h3>Q: IM 绑定后收不到消息？</h3>
    <p>检查以下几点：</p>
    <ul>
        <li>管理员是否已启用该 IM 平台配置</li>
        <li>IM 平台的回调 URL 是否指向正确的服务器地址</li>
        <li>绑定状态是否为「活跃」</li>
    </ul>

    <h3>Q: 如何切换向量匹配引擎？</h3>
    <p>
        默认使用内置 TF-IDF（无需任何配置）。如需更强的语义匹配能力，管理员可在 <code
            >config.yaml</code
        > 中配置：
    </p>
    <pre
        class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto">embedding:
  provider: "openai"
  store: "memory"</pre>

    <h3>Q: 忘记管理员密码怎么办？</h3>
    <p>
        首次部署时使用初始化向导设置密码。如需重置，访问 <code>/setup</code> 页面重新初始化（会清空所有数据）。
    </p>
</article>
