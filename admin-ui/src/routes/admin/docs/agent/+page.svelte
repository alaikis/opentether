<svelte:head><title>智能体使用指南 - OpenTether</title></svelte:head>

<article class="prose prose-slate max-w-none">
    <h1>智能体使用指南</h1>
    <p>
        OpenTether 智能体是一个能够自动推理、多步执行、调用工具的 AI
        Agent。下面介绍如何使用它完成各种任务。
    </p>

    <h2>快速体验</h2>
    <p>
        登录管理后台后，在左侧菜单找到「用户中心」→「AI 对话」，即可开始对话。
    </p>

    <div class="not-prose card my-4 bg-indigo-50 border-indigo-200">
        <h3 class="font-semibold text-indigo-800 mb-2">💡 两种对话模式</h3>
        <table class="w-full text-sm">
            <tr class="border-b border-indigo-100">
                <td class="py-2 font-medium text-indigo-700 w-32">REST 同步</td>
                <td class="py-2 text-slate-600"
                    >提交问题，等待完整回复。适合简单查询、一次性的数据请求。</td
                >
            </tr>
            <tr>
                <td class="py-2 font-medium text-indigo-700">SSE 流式</td>
                <td class="py-2 text-slate-600"
                    >像打字机一样逐字输出。适合长文本生成、报表分析。</td
                >
            </tr>
        </table>
    </div>

    <h2>智能体工作方式</h2>
    <p>
        智能体采用 <strong>ReAct（推理+行动）</strong> 模式，不是简单地一问一答，而是：
    </p>

    <div class="not-prose my-4">
        <pre
            class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto">用户: "查询华东区销售冠军并生成报表"
    │
    ├─ 第1步 推理 → 判断需要查数据库
    │   调用: text2sql("华东区销售额排名")
    │   结果: [数据] 张三 | 500万 | 华东 ...
    │
    ├─ 第2步 推理 → 还需要产品维度
    │   调用: text2sql("华东区各产品线销售")
    │   结果: [数据] 产品A | 200万 ...
    │
    └─ 第3步 推理 → 信息足够，生成最终回答
        回复: "华东区销售冠军是张三，销售额500万。各产品线中产品A表现最好..."</pre>
    </div>

    <h2>对话场景示例</h2>

    <h3>📊 数据查询与分析</h3>
    <div class="not-prose card my-4">
        <p class="text-sm font-medium text-slate-700 mb-2">你可以这样问：</p>
        <ul class="text-sm text-slate-600 space-y-1 list-disc list-inside">
            <li>"查询本月销售额超过 10 万的员工"</li>
            <li>"对比华东和华南两个区域的业绩"</li>
            <li>"分析最近三个月的销售趋势"</li>
            <li>"找出库存低于安全线的产品"</li>
            <li>"统计各部门的考勤情况"</li>
        </ul>
    </div>

    <h3>👥 员工信息查询</h3>
    <div class="not-prose card my-4">
        <p class="text-sm font-medium text-slate-700 mb-2">你可以这样问：</p>
        <ul class="text-sm text-slate-600 space-y-1 list-disc list-inside">
            <li>"查询技术部的所有员工"</li>
            <li>"张三是哪个部门的？"</li>
            <li>"列出所有销售经理"</li>
            <li>"入职超过 3 年的员工有哪些"</li>
        </ul>
    </div>

    <h3>📄 报表生成</h3>
    <div class="not-prose card my-4">
        <p class="text-sm font-medium text-slate-700 mb-2">你可以这样问：</p>
        <ul class="text-sm text-slate-600 space-y-1 list-disc list-inside">
            <li>"生成本月销售报表"</li>
            <li>"导出一份员工绩效汇总 PDF"</li>
            <li>"把刚才的查询结果生成报表"</li>
        </ul>
    </div>

    <h3>💬 通用对话</h3>
    <div class="not-prose card my-4">
        <p class="text-sm font-medium text-slate-700 mb-2">你也可以：</p>
        <ul class="text-sm text-slate-600 space-y-1 list-disc list-inside">
            <li>"帮我分析一下这个数据"</li>
            <li>"用通俗的语言解释刚才的查询结果"</li>
            <li>"给出改善销售业绩的建议"</li>
        </ul>
    </div>

    <h2>参数补全（多轮询问）</h2>
    <p>
        当智能体发现你提供的信息不足以完成任务时，会主动向你提问，而不是猜测或跳过：
    </p>

    <div class="not-prose my-4">
        <pre
            class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto">你: "帮我查一下销售额"

AI: "请问您需要查询哪个时间段的数据？是本月还是上个月？"

你: "本月"

AI: "好的，请问是查看所有区域的还是特定区域的？"

你: "华东区"

AI → 执行查询 → 返回华东区本月销售额</pre>
    </div>

    <h2>写入操作确认（安全屏障）</h2>
    <p>所有插入、更新、删除操作都需要你明确确认后才执行：</p>

    <div class="not-prose card my-4 bg-amber-50 border-amber-200">
        <h3 class="font-semibold text-amber-800 mb-2">⚠️ 操作确认示例</h3>
        <pre
            class="bg-slate-900 text-slate-100 p-3 rounded-lg text-xs overflow-x-auto">你: "把张三级别的改为高级工程师"

AI: ## ⚠️ 员工信息更新确认
    将更新员工"张三"的职位信息

    | 类型   | 目标   | 详情                        |
    |--------|--------|-----------------------------|
    | update | users  | 将张三(工号xxx)的职位更新为高级工程师 |

    以上操作需要您的确认。请回复"确认执行"以继续。</pre>
        <p class="text-sm text-amber-700 mt-2">
            <strong>只有在你明确回复"确认执行"后，操作才会真正执行。</strong>
            操作完成后会自动存入审计日志，管理员可追溯。
        </p>
    </div>
    <p>智能体支持上下文记忆，你可以在一个对话中连续提问：</p>

    <div class="not-prose my-4">
        <pre
            class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto">第1轮: "查询技术部员工"
    → 返回 12 名员工列表

第2轮: "其中哪些是高级工程师？"
    → AI 记住上文，自动在 12 人中筛选

第3轮: "把他们的平均工龄算出来"
    → AI 继续在上文基础上计算</pre>
    </div>

    <h2>多步并行查询（高效率）</h2>
    <p>当你需要多个维度的数据时，智能体会自动并行执行互不依赖的查询：</p>

    <div class="not-prose my-4">
        <pre
            class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto">"做一个完整销售分析"
    │
    ├─ ─并行─ 查询各地区销售额 ───── 2s ─┐
    ├─ ─并行─ 查询产品排名    ───── 2s ─┤
    └─ ─并行─ 查询月度趋势    ───── 2s ─┘  ← 同时执行，总耗时 2s
    │
    └─ 自动组装为完整分析报告</pre>
    </div>

    <h2>经验复用（省 Token）</h2>
    <p>
        智能体会记住你常用的操作模式。当类似问题再次出现时，直接复用之前的执行步骤，大幅节省时间和
        Token：
    </p>

    <div class="not-prose card my-4 bg-amber-50 border-amber-200">
        <p class="text-sm text-amber-800">
            <strong>第一次</strong>："查询本月销售排名" → 完整推理 3 步，消耗
            ~5000 tokens
        </p>
        <p class="text-sm text-amber-800 mt-1">
            <strong>第二次</strong>："查询本月销售排名" →
            命中经验，直接执行，消耗 ~200 tokens
        </p>
        <p class="text-sm text-amber-700 mt-2">
            ✨ Token 节省: <strong>~96%</strong>
        </p>
    </div>

    <h2>API 调用方式</h2>

    <h3>REST API（同步）</h3>
    <pre
        class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto">POST /api/v1/user/chat
Authorization: Bearer &lt;your-jwt-token&gt;
Content-Type: application/json

{'{"message": "查询销售数据", "conversation_id": ""}'}</pre>

    <h3>SSE 流式（逐字输出）</h3>
    <pre
        class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto">POST /api/v1/user/chat/stream
Authorization: Bearer &lt;your-jwt-token&gt;
Content-Type: application/json

{'{"message": "生成本月销售分析报告", "conversation_id": ""}'}

// 响应: text/event-stream
data: 本月
data: 销售
data: 分析报告
data: ...
data: [DONE]</pre>

    <h3>在 IM 中使用</h3>
    <p>绑定 IM 账号后，直接在微信/飞书/钉钉中发送消息即可，无需调用 API。</p>

    <h2>可用工具列表</h2>
    <table class="w-full text-sm border border-slate-200 rounded-lg my-3">
        <thead
            ><tr class="bg-slate-50">
                <th class="text-left p-3">工具</th><th class="text-left p-3"
                    >功能</th
                ><th class="text-left p-3">使用条件</th>
            </tr></thead
        >
        <tbody class="divide-y divide-slate-100">
            <tr
                ><td class="p-3 font-medium font-mono text-xs">chat</td><td
                    class="p-3 text-slate-500">通用对话，回答各类问题</td
                ><td class="p-3 text-slate-500">所有用户</td></tr
            >
            <tr
                ><td class="p-3 font-medium font-mono text-xs">text2sql</td><td
                    class="p-3 text-slate-500">自然语言转 SQL，查询数据库</td
                ><td class="p-3 text-slate-500">需配置数据源</td></tr
            >
            <tr
                ><td class="p-3 font-medium font-mono text-xs"
                    >employee_query</td
                ><td class="p-3 text-slate-500">查询员工信息、组织架构</td><td
                    class="p-3 text-slate-500">所有用户（数据范围受限）</td
                ></tr
            >
            <tr
                ><td class="p-3 font-medium font-mono text-xs">generate_pdf</td
                ><td class="p-3 text-slate-500">生成 PDF 报表</td><td
                    class="p-3 text-slate-500">需注册 skill_pdf</td
                ></tr
            >
            <tr
                ><td class="p-3 font-medium font-mono text-xs"
                    >generate_report</td
                ><td class="p-3 text-slate-500">生成数据报表</td><td
                    class="p-3 text-slate-500">需注册 skill_report</td
                ></tr
            >
        </tbody>
    </table>

    <h2>常见问题</h2>

    <h3>Q: 智能体为什么没有调用工具？</h3>
    <p>可能原因：</p>
    <ul>
        <li>问题太简单，不需要工具就能回答</li>
        <li>LLM 提供商未配置或不可用</li>
        <li>数据源未配置（针对 text2sql）</li>
    </ul>

    <h3>Q: 如何让智能体更准确地理解我的需求？</h3>
    <ul>
        <li>问题描述尽量具体，包含关键信息</li>
        <li>使用业务术语（如"销售额""库存"），智能体已通过 Skill 关键词匹配</li>
        <li>复杂任务可以分步提问，逐步细化</li>
    </ul>

    <h3>Q: 一次对话能调用几次工具？</h3>
    <p>
        最多 10 步，单次对话总超时 5 分钟。在实际使用中，大多数任务 2-4
        步即可完成。
    </p>

    <h3>Q: 如何在外部系统中调用？</h3>
    <ol>
        <li>在「系统设置」中为外部系统创建 API Key</li>
        <li>请求时携带 <code>X-API-Key: ot_sk_xxxx...</code></li>
        <li>调用 <code>POST /api/v1/external/bind-im</code> 等集成接口</li>
    </ol>
</article>
