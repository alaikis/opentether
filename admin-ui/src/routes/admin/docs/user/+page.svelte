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

    <h2>IM 绑定（员工自助操作）</h2>
    <p>
        管理员配置好 IM 平台后，员工可通过 <strong>扫码</strong> 或
        <strong>手动输入</strong> 两种方式绑定自己的 IM 账号。
    </p>

    <h3>方式一：扫码绑定（推荐）</h3>
    <ol>
        <li>登录管理后台，进入「我的 IM」页面</li>
        <li>查看可用平台列表：<code>GET /api/v1/user/im/platforms</code></li>
        <li>选择平台，点击「扫码绑定」</li>
        <li>系统生成绑定二维码（含临时 token，有效期 30 分钟）</li>
        <li>用对应 IM 应用扫描二维码</li>
        <li>扫码后自动完成绑定，无需手动输入任何 ID</li>
    </ol>

    <div class="not-prose card my-6 bg-indigo-50 border-indigo-200">
        <h3 class="font-semibold text-indigo-800 mb-2">
            ✨ 扫码绑定 vs 手动绑定
        </h3>
        <table class="w-full text-sm">
            <thead
                ><tr class="border-b border-indigo-200">
                    <th class="text-left py-2 font-medium text-indigo-600"
                        >扫码绑定</th
                    >
                    <th class="text-left py-2 font-medium text-indigo-600"
                        >手动绑定</th
                    >
                </tr></thead
            >
            <tbody>
                <tr>
                    <td class="py-2 text-slate-600 pr-4">无需知道平台 UserID</td
                    >
                    <td class="py-2 text-slate-600"
                        >需要从平台后台获取 UserID</td
                    >
                </tr>
                <tr>
                    <td class="py-2 text-slate-600 pr-4">自动建立信任关系</td>
                    <td class="py-2 text-slate-600">需管理员审核</td>
                </tr>
                <tr>
                    <td class="py-2 text-slate-600 pr-4"
                        >支持平台：微信生态、企业微信、飞书、钉钉</td
                    >
                    <td class="py-2 text-slate-600">所有平台</td>
                </tr>
            </tbody>
        </table>
    </div>

    <h3>方式二：手动绑定</h3>
    <ol>
        <li>登录管理后台</li>
        <li>进入「我的 IM」→ 查看 IM 绑定状态</li>
        <li>选择已配置的 IM 平台，输入自己的 IM 账号 ID</li>
        <li>提交绑定</li>
    </ol>

    <h3>各平台 UserID 获取方式</h3>
    <div class="not-prose card my-6">
        <table class="w-full text-sm">
            <thead
                ><tr class="border-b border-slate-200">
                    <th class="text-left py-2 font-medium text-slate-500"
                        >平台</th
                    >
                    <th class="text-left py-2 font-medium text-slate-500"
                        >扫码绑定</th
                    >
                    <th class="text-left py-2 font-medium text-slate-500"
                        >UserID 格式</th
                    >
                    <th class="text-left py-2 font-medium text-slate-500"
                        >手动获取方式</th
                    >
                </tr></thead
            >
            <tbody>
                <tr class="border-b border-slate-100"
                    ><td class="py-2 font-medium">企业微信</td><td
                        class="py-2 text-green-600 text-xs">✅ 支持</td
                    ><td class="py-2 text-slate-500 font-mono text-xs"
                        >ZhangSan</td
                    ><td class="py-2 text-slate-500"
                        >企业微信管理后台 → 通讯录</td
                    ></tr
                >
                <tr class="border-b border-slate-100"
                    ><td class="py-2 font-medium">飞书</td><td
                        class="py-2 text-green-600 text-xs">✅ 支持</td
                    ><td class="py-2 text-slate-500 font-mono text-xs"
                        >ou_xxx</td
                    ><td class="py-2 text-slate-500">飞书管理后台 → 用户详情</td
                    ></tr
                >
                <tr class="border-b border-slate-100"
                    ><td class="py-2 font-medium">钉钉</td><td
                        class="py-2 text-green-600 text-xs">✅ 支持</td
                    ><td class="py-2 text-slate-500 font-mono text-xs"
                        >manager123</td
                    ><td class="py-2 text-slate-500">钉钉管理后台 → 员工信息</td
                    ></tr
                >
                <tr class="border-b border-slate-100"
                    ><td class="py-2 font-medium">iLink AI（公众号）</td><td
                        class="py-2 text-green-600 text-xs">✅ 支持</td
                    ><td class="py-2 text-slate-500 font-mono text-xs"
                        >oXXXX_xxx</td
                    ><td class="py-2 text-slate-500"
                        >公众号后台 → 用户 OpenID</td
                    ></tr
                >
                <tr
                    ><td class="py-2 font-medium">WhatsApp</td><td
                        class="py-2 text-amber-600 text-xs">验证码</td
                    ><td class="py-2 text-slate-500 font-mono text-xs"
                        >+86138xxxx</td
                    ><td class="py-2 text-slate-500">WhatsApp 手机号</td></tr
                >
            </tbody>
        </table>
    </div>

    <h2>API 密钥管理</h2>
    <p>API 密钥用于让外部系统（如 OA、ERP）代表您调用 OpenTether 的功能。</p>

    <h3>员工自助管理</h3>
    <ol>
        <li>进入「我的 API 密钥」页面</li>
        <li>点击「创建密钥」→ 输入名称、选择权限范围、设置有效期</li>
        <li>系统返回完整密钥（<strong>仅此一次可见，请立即保存</strong>）</li>
        <li>
            密钥格式：<code
                >ot_sk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx</code
            >
        </li>
    </ol>

    <div class="not-prose card my-6 bg-amber-50 border-amber-200">
        <h3 class="font-semibold text-amber-800 mb-2">⚠️ 安全提醒</h3>
        <ul class="text-sm text-amber-700 space-y-1 list-disc list-inside">
            <li>创建后立即保存密钥，关闭页面后无法再次查看</li>
            <li>如密钥泄露，可随时「重新生成」使旧密钥失效</li>
            <li>建议为不同系统创建不同的密钥，便于审计和撤销</li>
            <li>定期检查密钥使用情况，删除不再需要的密钥</li>
        </ul>
    </div>

    <h3>外部系统集成</h3>
    <p>OA/ERP 系统管理员可将 API Key 配置到系统中，实现：</p>
    <ul>
        <li>自动为新员工绑定 IM 账号</li>
        <li>同步员工信息到 OpenTether</li>
        <li>触发定时任务和报表生成</li>
    </ul>

    <h2>使用 AI 对话</h2>
    <p>OpenTether 内置智能体引擎（Agentic Loop），能自动多步推理、调用工具。</p>

    <h3>对话方式</h3>
    <div class="not-prose grid grid-cols-1 md:grid-cols-2 gap-4 my-4">
        <div class="card">
            <h3 class="font-semibold text-primary-700 text-sm">
                REST API 同步对话
            </h3>
            <p class="text-xs text-slate-500 mt-1">
                适合简单查询、一次性请求。提交问题后等待完整回复。
            </p>
            <p class="text-xs text-slate-400 mt-2 font-mono">
                POST /api/v1/user/chat
            </p>
        </div>
        <div class="card">
            <h3 class="font-semibold text-emerald-700 text-sm">SSE 流式对话</h3>
            <p class="text-xs text-slate-500 mt-1">
                适合长文本生成，逐字流式输出，体验更流畅。
            </p>
            <p class="text-xs text-slate-400 mt-2 font-mono">
                POST /api/v1/user/chat/stream
            </p>
        </div>
    </div>

    <h3>多轮对话</h3>
    <p>
        每次对话返回 <code>conversation_id</code
        >，下次请求时传入即可延续上下文：
    </p>
    <pre
        class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto">// 第一轮
POST /api/v1/user/chat  {'{"message": "查询销售数据", "conversation_id": ""}'}
    → 返回 {'{"conversation_id": "uuid-xxx", ...}'}

// 第二轮（带上下文）
POST /api/v1/user/chat  {'{"message": "按地区排名", "conversation_id": "uuid-xxx"}'}
    → AI 会结合上一轮结果回答</pre>

    <h3>智能体自动工具调用</h3>
    <p>当您提问涉及数据查询时，AI 会自动识别并调用对应工具：</p>
    <ul>
        <li><strong>text2sql</strong> — 自动将问题转为 SQL 查询数据库</li>
        <li><strong>employee_query</strong> — 查询员工信息和组织架构</li>
        <li><strong>generate_pdf</strong> — 生成 PDF 报表</li>
        <li><strong>generate_report</strong> — 生成数据报表</li>
    </ul>
    <p>一次对话中，AI 可以连续调用多个工具完成任务，无需手动切换。</p>

    <h2>在 IM 中使用 AI</h2>
    <p>绑定 IM 账号后，可以直接在微信/飞书/钉钉中与 AI 对话：</p>
    <ol>
        <li>确保已绑定 IM 账号（参考上方「IM 绑定」章节）</li>
        <li>在 IM 应用中直接发送消息给机器人</li>
        <li>AI 自动识别您的身份，在权限范围内响应</li>
        <li>支持文本对话、数据查询、报表生成</li>
    </ol>

    <h3>支持的 IM 平台</h3>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">平台</th><th class="text-left p-3"
                    >绑定方式</th
                ><th class="text-left p-3">使用方式</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-medium">企业微信</td><td
                    class="p-3 text-slate-500">扫码绑定</td
                ><td class="p-3 text-slate-500">企业微信内直接对话</td></tr
            >
            <tr
                ><td class="p-3 font-medium">飞书</td><td
                    class="p-3 text-slate-500">扫码绑定</td
                ><td class="p-3 text-slate-500">飞书内直接对话</td></tr
            >
            <tr
                ><td class="p-3 font-medium">钉钉</td><td
                    class="p-3 text-slate-500">扫码绑定</td
                ><td class="p-3 text-slate-500">钉钉内直接对话</td></tr
            >
            <tr
                ><td class="p-3 font-medium">iLink AI</td><td
                    class="p-3 text-slate-500">微信扫码关注公众号</td
                ><td class="p-3 text-slate-500">微信内对话（同步回复）</td></tr
            >
            <tr
                ><td class="p-3 font-medium">WhatsApp</td><td
                    class="p-3 text-slate-500">验证码绑定</td
                ><td class="p-3 text-slate-500">WhatsApp 对话</td></tr
            >
        </tbody>
    </table>

    <h2>权限说明</h2>
    <div class="not-prose grid grid-cols-1 md:grid-cols-3 gap-4 my-4">
        <div class="card">
            <h3 class="font-semibold text-primary-700 text-sm">管理员</h3>
            <p class="text-xs text-slate-500 mt-1">
                全部权限：管理用户、配置系统、查看日志、管理 API 密钥、使用所有
                Skill
            </p>
        </div>
        <div class="card">
            <h3 class="font-semibold text-emerald-700 text-sm">普通用户</h3>
            <p class="text-xs text-slate-500 mt-1">
                受限权限：使用被分配的 Skill，访问范围内数据，管理自己的 IM
                绑定和 API 密钥
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
        系统只在已注册 Skill 范围内工作。说明你的问题没有匹配到任何 Skill
        的关键词。请联系管理员添加对应的 Skill 或扩展关键词。
    </p>

    <h3>Q: IM 绑定后收不到消息？</h3>
    <p>检查以下几点：</p>
    <ul>
        <li>管理员是否已启用该 IM 平台配置</li>
        <li>IM 平台的回调 URL 是否指向正确的服务器地址</li>
        <li>绑定状态是否为「活跃」</li>
        <li>扫码绑定是否在 30 分钟内完成</li>
    </ul>

    <h3>Q: 如何生成 API 密钥给 OA 系统用？</h3>
    <ol>
        <li>进入「我的 API 密钥」页面</li>
        <li>创建密钥 → 选择权限范围 → 保存返回的密钥</li>
        <li>
            将密钥配置到 OA/ERP 系统的 HTTP 请求 Header：<code
                >X-API-Key: ot_sk_xxx...</code
            >
        </li>
    </ol>

    <h3>Q: 流式对话和非流式对话有什么区别？</h3>
    <p>
        <strong>非流式（REST）</strong
        >：提交后等待完整回复再显示，适合短回答。<br />
        <strong>流式（SSE）</strong
        >：像打字机一样逐字输出，适合长文本生成，体验更好。
    </p>

    <h3>Q: 忘记管理员密码怎么办？</h3>
    <p>
        首次部署时使用初始化向导设置密码。如需重置，访问 <code>/setup</code> 页面重新初始化（会清空所有数据）。
    </p>

    <h3>Q: 独立 Agent 是什么？</h3>
    <p>
        独立 Agent
        是可以部署在其他机器上的智能体客户端，与主系统配对后接收远程任务。适用于：
    </p>
    <ul>
        <li>需要 GPU 的任务（AI 推理）</li>
        <li>隔离环境的任务（内网数据库查询）</li>
        <li>分布式任务执行</li>
    </ul>
    <p>配对方式：Agent 启动生成配对码 → 管理员输入配对码 → 自动建立连接。</p>

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

    <h3>Q: 如何从 Markdown 文件快速创建 Skill？</h3>
    <p>
        管理员可使用「从 MD 创建 Skill」功能，上传格式化的 Markdown
        文件即可自动解析为 Skill：
    </p>
    <ul>
        <li>管理后台 → Skills → 上传 MD 文件</li>
        <li>
            或直接调用 API：<code>POST /api/v1/admin/skills/from-markdown</code>
        </li>
    </ul>
</article>
