<svelte:head><title>文档中心 - OpenTether</title></svelte:head>

<div class="max-w-4xl">
    <h1 class="text-2xl font-bold text-slate-800 mb-2">文档中心</h1>
    <p class="text-slate-500 mb-10">
        OpenTether 企业级智能体管理平台——让 AI 能力安全可控地服务企业
    </p>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-5 mb-10">
        <a href="/admin/docs/dev" class="card group">
            <div
                class="w-11 h-11 rounded-xl bg-indigo-50 text-indigo-600 flex items-center justify-center text-xl mb-3"
            >
                💻
            </div>
            <h3 class="font-semibold text-slate-800">开发者文档</h3>
            <p class="text-sm text-slate-500">
                API 参考 · 架构设计 · Skill 开发 · 部署指南
            </p>
        </a>
        <a href="/admin/docs/user" class="card group">
            <div
                class="w-11 h-11 rounded-xl bg-emerald-50 text-emerald-600 flex items-center justify-center text-xl mb-3"
            >
                📖
            </div>
            <h3 class="font-semibold text-slate-800">用户指南</h3>
            <p class="text-sm text-slate-500">
                快速上手 · 功能说明 · IM 绑定 · 常见问题
            </p>
        </a>
    </div>

    <hr class="border-slate-200 my-8" />

    <h2 class="text-xl font-bold text-slate-800 mb-4">平台架构概览</h2>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-5 mb-6">
        <div class="card">
            <h3 class="font-semibold text-primary-700 text-sm mb-2">
                🏢 公司层
            </h3>
            <p class="text-sm text-slate-500">
                管理员统一配置 Skills、数据源、LLM 提供商、IM
                平台。设置用户组与权限边界。
            </p>
        </div>
        <div class="card">
            <h3 class="font-semibold text-emerald-700 text-sm mb-2">
                👤 员工层
            </h3>
            <p class="text-sm text-slate-500">
                员工绑定个人 IM 账号，在权限范围内使用 AI
                能力。所有操作受公司边界管控。
            </p>
        </div>
        <div class="card">
            <h3 class="font-semibold text-amber-700 text-sm mb-2">🤖 AI 层</h3>
            <p class="text-sm text-slate-500">
                向量匹配 → 意图识别 → 权限检查 → Skill 执行。只在注册 Skill
                范围内工作，杜绝幻听。
            </p>
        </div>
    </div>

    <h2 class="text-xl font-bold text-slate-800 mb-4 mt-8">支持的接入方式</h2>

    <div class="overflow-x-auto">
        <table
            class="w-full text-sm border border-slate-200 rounded-lg overflow-hidden"
        >
            <thead>
                <tr class="bg-slate-50">
                    <th class="text-left py-3 px-4 font-medium text-slate-600"
                        >平台</th
                    >
                    <th class="text-left py-3 px-4 font-medium text-slate-600"
                        >标识</th
                    >
                    <th class="text-left py-3 px-4 font-medium text-slate-600"
                        >绑定方式</th
                    >
                    <th class="text-left py-3 px-4 font-medium text-slate-600"
                        >回复方式</th
                    >
                </tr>
            </thead>
            <tbody class="divide-y divide-slate-100">
                <tr
                    ><td class="py-3 px-4 font-medium">企业微信</td><td
                        class="py-3 px-4 text-slate-500">wecom</td
                    ><td class="py-3 px-4 text-slate-500">企业微信 UserID</td
                    ><td class="py-3 px-4 text-slate-500">异步推送</td></tr
                >
                <tr
                    ><td class="py-3 px-4 font-medium">飞书</td><td
                        class="py-3 px-4 text-slate-500">feishu</td
                    ><td class="py-3 px-4 text-slate-500">飞书 OpenID</td><td
                        class="py-3 px-4 text-slate-500">异步推送</td
                    ></tr
                >
                <tr
                    ><td class="py-3 px-4 font-medium">钉钉</td><td
                        class="py-3 px-4 text-slate-500">dingtalk</td
                    ><td class="py-3 px-4 text-slate-500">钉钉 UserID</td><td
                        class="py-3 px-4 text-slate-500">异步推送</td
                    ></tr
                >
                <tr
                    ><td class="py-3 px-4 font-medium">iLink AI（公众号）</td
                    ><td class="py-3 px-4 text-slate-500">ilink</td><td
                        class="py-3 px-4 text-slate-500">微信 OpenID</td
                    ><td class="py-3 px-4 text-slate-500"
                        ><span
                            class="px-2 py-0.5 bg-amber-50 text-amber-700 rounded-full text-xs"
                            >同步回复</span
                        ></td
                    ></tr
                >
            </tbody>
        </table>
    </div>

    <h2 class="text-xl font-bold text-slate-800 mb-4 mt-8">向量匹配引擎</h2>
    <p class="text-sm text-slate-500 mb-3">
        默认使用内置 TF-IDF 向量化 + 内存存储（零依赖启动）。可通过配置切换为
        OpenAI Embedding 或 Milvus 向量数据库。
    </p>

    <pre
        class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto">embedding:
  provider: "tfidf"    # tfidf(默认) | openai | local
  store: "memory"      # memory(默认) | milvus | qdrant</pre>
</div>
