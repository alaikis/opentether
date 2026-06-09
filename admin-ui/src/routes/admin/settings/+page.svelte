<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import {
        Settings,
        Server,
        Database,
        Shield,
        Brain,
        Cog,
        RefreshCw,
        Mail,
        Save,
        AlertCircle,
        CheckCircle,
    } from "lucide-svelte";

    let config: any = {};
    let loading = true;
    let saving = false;
    let message = "";
    let error = "";
    let activeTab = "server";

    const tabs = [
        { id: "server", label: "服务", icon: Server },
        { id: "database", label: "数据库", icon: Database },
        { id: "security", label: "安全", icon: Shield },
        { id: "embedding", label: "向量引擎", icon: Brain },
        { id: "executor", label: "执行器", icon: Cog },
        { id: "smtp", label: "邮件", icon: Mail },
        { id: "update", label: "更新", icon: RefreshCw },
    ];

    onMount(async () => {
        try {
            config = await api.get("/admin/system/config");
        } catch (e: any) {
            error = e.message;
        } finally {
            loading = false;
        }
    });

    async function saveConfig() {
        saving = true;
        message = "";
        error = "";
        try {
            const result = await api.put("/admin/system/config", config);
            message = result.message || "配置已保存";
        } catch (e: any) {
            error = e.message;
        } finally {
            saving = false;
        }
    }

    async function testSMTP() {
        try {
            await api.post("/admin/system/smtp/test");
            message = "测试邮件已发送";
        } catch (e: any) {
            error = e.message;
        }
    }

    function toggleArrayItem(arr: string[], item: string) {
        const idx = arr.indexOf(item);
        if (idx >= 0) arr.splice(idx, 1);
        else arr.push(item);
        config = config;
    }

    function handleCorsKeydown(e: KeyboardEvent) {
        if (e.key === "Enter") {
            const target = e.target as HTMLInputElement;
            const val = target.value.trim();
            if (val && !config.security.cors.allowed_origins.includes(val)) {
                config.security.cors.allowed_origins.push(val);
            }
            target.value = "";
        }
    }

    function removeOrigin(origin: string) {
        config.security.cors.allowed_origins =
            config.security.cors.allowed_origins.filter(
                (o: string) => o !== origin,
            );
    }
</script>

<svelte:head><title>系统设置 - OpenTether</title></svelte:head>

<div class="max-w-4xl">
    <div class="flex items-center justify-between mb-6">
        <div>
            <h1 class="text-2xl font-bold text-slate-800">系统设置</h1>
            <p class="text-sm text-slate-500 mt-1">
                更新配置后部分修改需重启服务生效
            </p>
        </div>
        <button
            class="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 flex items-center gap-2 text-sm disabled:opacity-50"
            on:click={saveConfig}
            disabled={saving || loading}
        >
            <Save class="w-4 h-4" />
            {saving ? "保存中..." : "保存配置"}
        </button>
    </div>

    {#if message}
        <div
            class="mb-4 p-3 bg-green-50 border border-green-200 text-green-700 rounded-lg flex items-center gap-2 text-sm"
        >
            <CheckCircle class="w-4 h-4" />
            {message}
        </div>
    {/if}
    {#if error}
        <div
            class="mb-4 p-3 bg-red-50 border border-red-200 text-red-700 rounded-lg flex items-center gap-2 text-sm"
        >
            <AlertCircle class="w-4 h-4" />
            {error}
        </div>
    {/if}

    {#if loading}
        <div class="text-center py-12 text-slate-400">加载中...</div>
    {:else}
        <!-- Tab Navigation -->
        <div class="flex gap-1 mb-6 border-b border-slate-200 overflow-x-auto">
            {#each tabs as tab}
                <button
                    class="px-4 py-2.5 text-sm font-medium whitespace-nowrap border-b-2 transition-colors {activeTab ===
                    tab.id
                        ? 'border-primary-600 text-primary-600'
                        : 'border-transparent text-slate-500 hover:text-slate-700'}"
                    on:click={() => (activeTab = tab.id)}
                >
                    <span class="flex items-center gap-1.5">
                        <svelte:component this={tab.icon} class="w-4 h-4" />
                        {tab.label}
                    </span>
                </button>
            {/each}
        </div>

        <!-- Server Tab -->
        {#if activeTab === "server"}
            <div class="card space-y-4">
                <h3 class="font-semibold text-slate-800">服务端配置</h3>
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >监听端口</label
                        >
                        <input
                            type="number"
                            bind:value={config.server.port}
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >运行模式</label
                        >
                        <select
                            bind:value={config.server.mode}
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        >
                            <option value="development">development</option>
                            <option value="production">production</option>
                        </select>
                    </div>
                </div>
            </div>
        {/if}

        <!-- Database Tab -->
        {#if activeTab === "database"}
            <div class="card space-y-4">
                <h3 class="font-semibold text-slate-800">数据库配置</h3>
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >类型</label
                        >
                        <select
                            bind:value={config.database.type}
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        >
                            <option value="sqlite">SQLite</option>
                            <option value="mysql">MySQL</option>
                            <option value="postgres">PostgreSQL</option>
                        </select>
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >数据库名/文件</label
                        >
                        <input
                            type="text"
                            bind:value={config.database.name}
                            placeholder="data/opentether.db"
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >主机</label
                        >
                        <input
                            type="text"
                            bind:value={config.database.host}
                            placeholder="localhost"
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >端口</label
                        >
                        <input
                            type="number"
                            bind:value={config.database.port}
                            placeholder="3306"
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >用户名</label
                        >
                        <input
                            type="text"
                            bind:value={config.database.user}
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >密码</label
                        >
                        <input
                            type="password"
                            bind:value={config.database.password}
                            placeholder="留空不修改"
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                </div>
            </div>
        {/if}

        <!-- Security Tab -->
        {#if activeTab === "security"}
            <div class="space-y-4">
                <div class="card space-y-4">
                    <h3 class="font-semibold text-slate-800">JWT 配置</h3>
                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <label
                                class="block text-xs font-medium text-slate-600 mb-1"
                                >密钥</label
                            >
                            <input
                                type="password"
                                bind:value={config.security.jwt.secret}
                                placeholder="留空不修改"
                                class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                            />
                        </div>
                        <div>
                            <label
                                class="block text-xs font-medium text-slate-600 mb-1"
                                >过期时间</label
                            >
                            <input
                                type="text"
                                bind:value={config.security.jwt.expire}
                                placeholder="24h"
                                class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                            />
                        </div>
                        <div>
                            <label
                                class="block text-xs font-medium text-slate-600 mb-1"
                                >刷新过期时间</label
                            >
                            <input
                                type="text"
                                bind:value={config.security.jwt.refresh_expire}
                                placeholder="7d"
                                class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                            />
                        </div>
                    </div>
                </div>

                <div class="card space-y-4">
                    <h3 class="font-semibold text-slate-800">速率限制</h3>
                    <div class="flex items-center gap-4">
                        <label class="flex items-center gap-2 text-sm">
                            <input
                                type="checkbox"
                                bind:checked={
                                    config.security.rate_limit.enabled
                                }
                                class="rounded"
                            />
                            启用
                        </label>
                        <div class="flex-1">
                            <label
                                class="block text-xs font-medium text-slate-600 mb-1"
                                >每分钟请求数</label
                            >
                            <input
                                type="number"
                                bind:value={
                                    config.security.rate_limit
                                        .requests_per_minute
                                }
                                class="w-32 border border-slate-300 rounded-lg px-3 py-2 text-sm"
                            />
                        </div>
                    </div>
                </div>

                <div class="card space-y-4">
                    <h3 class="font-semibold text-slate-800">HTTPS</h3>
                    <div class="space-y-3">
                        <label class="flex items-center gap-2 text-sm">
                            <input
                                type="checkbox"
                                bind:checked={config.security.https.enabled}
                                class="rounded"
                            />
                            启用 HTTPS
                        </label>
                        <div class="grid grid-cols-2 gap-4">
                            <div>
                                <label
                                    class="block text-xs font-medium text-slate-600 mb-1"
                                    >证书文件</label
                                >
                                <input
                                    type="text"
                                    bind:value={config.security.https.cert_file}
                                    placeholder="./certs/server.crt"
                                    class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                                />
                            </div>
                            <div>
                                <label
                                    class="block text-xs font-medium text-slate-600 mb-1"
                                    >密钥文件</label
                                >
                                <input
                                    type="text"
                                    bind:value={config.security.https.key_file}
                                    placeholder="./certs/server.key"
                                    class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                                />
                            </div>
                        </div>
                    </div>
                </div>

                <div class="card space-y-4">
                    <h3 class="font-semibold text-slate-800">CORS 跨域</h3>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >允许的来源</label
                        >
                        <div class="flex flex-wrap gap-2 mb-2">
                            {#each config.security.cors.allowed_origins as origin}
                                <span
                                    class="px-2 py-1 bg-slate-100 rounded text-xs flex items-center gap-1"
                                >
                                    {origin}
                                    <button
                                        class="text-red-500 hover:text-red-700"
                                        on:click={() => removeOrigin(origin)}
                                        >×</button
                                    >
                                </span>
                            {/each}
                        </div>
                        <input
                            type="text"
                            placeholder="输入后回车添加"
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                            on:keydown={handleCorsKeydown}
                        />
                    </div>
                </div>
            </div>
        {/if}

        <!-- Embedding Tab -->
        {#if activeTab === "embedding"}
            <div class="card space-y-4">
                <h3 class="font-semibold text-slate-800">向量引擎配置</h3>
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >Embedding 提供者</label
                        >
                        <select
                            bind:value={config.embedding.provider}
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        >
                            <option value="tfidf">tfidf (内置，零依赖)</option>
                            <option value="openai">openai</option>
                            <option value="local">local</option>
                        </select>
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >模型</label
                        >
                        <input
                            type="text"
                            bind:value={config.embedding.model}
                            placeholder="bge-m3"
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >向量维度</label
                        >
                        <input
                            type="number"
                            bind:value={config.embedding.dimension}
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >向量存储</label
                        >
                        <select
                            bind:value={config.embedding.store}
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        >
                            <option value="memory">memory (内置)</option>
                            <option value="milvus">milvus</option>
                            <option value="qdrant">qdrant</option>
                        </select>
                    </div>
                </div>
            </div>
        {/if}

        <!-- Executor Tab -->
        {#if activeTab === "executor"}
            <div class="space-y-4">
                <div class="card space-y-4">
                    <h3 class="font-semibold text-slate-800">执行器配置</h3>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >执行模式</label
                        >
                        <select
                            bind:value={config.executor.mode}
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        >
                            <option value="embedded">embedded (内嵌)</option>
                            <option value="independent"
                                >independent (独立 Agent)</option
                            >
                        </select>
                    </div>
                </div>

                <div class="card space-y-4">
                    <h3 class="font-semibold text-slate-800">内嵌执行器</h3>
                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <label
                                class="block text-xs font-medium text-slate-600 mb-1"
                                >最大并发数</label
                            >
                            <input
                                type="number"
                                bind:value={
                                    config.executor.embedded.max_concurrent
                                }
                                class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                            />
                        </div>
                        <div>
                            <label
                                class="block text-xs font-medium text-slate-600 mb-1"
                                >超时时间</label
                            >
                            <input
                                type="text"
                                bind:value={config.executor.embedded.timeout}
                                placeholder="1h"
                                class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                            />
                        </div>
                    </div>
                </div>

                <div class="card space-y-4">
                    <h3 class="font-semibold text-slate-800">
                        独立 Agent 队列
                    </h3>
                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <label
                                class="block text-xs font-medium text-slate-600 mb-1"
                                >队列类型</label
                            >
                            <select
                                bind:value={
                                    config.executor.independent.queue.type
                                }
                                class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                            >
                                <option value="http">http (Pull 模式)</option>
                                <option value="redis">redis</option>
                                <option value="kafka">kafka</option>
                            </select>
                        </div>
                        <div>
                            <label
                                class="block text-xs font-medium text-slate-600 mb-1"
                                >队列地址</label
                            >
                            <input
                                type="text"
                                bind:value={
                                    config.executor.independent.queue.address
                                }
                                placeholder="localhost:6379"
                                class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                            />
                        </div>
                    </div>
                </div>
            </div>
        {/if}

        <!-- SMTP Tab -->
        {#if activeTab === "smtp"}
            <div class="card space-y-4">
                <div class="flex items-center justify-between">
                    <h3 class="font-semibold text-slate-800">
                        邮件配置 (SMTP)
                    </h3>
                    <button
                        class="px-3 py-1.5 text-xs border border-slate-300 rounded-lg hover:bg-slate-50"
                        on:click={testSMTP}>测试发送</button
                    >
                </div>
                <label class="flex items-center gap-2 text-sm">
                    <input
                        type="checkbox"
                        bind:checked={config.smtp.enabled}
                        class="rounded"
                    />
                    启用邮件服务
                </label>
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >SMTP 主机</label
                        >
                        <input
                            type="text"
                            bind:value={config.smtp.host}
                            placeholder="smtp.example.com"
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >端口</label
                        >
                        <input
                            type="number"
                            bind:value={config.smtp.port}
                            placeholder="587"
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >加密方式</label
                        >
                        <select
                            bind:value={config.smtp.encryption}
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        >
                            <option value="tls">TLS</option>
                            <option value="ssl">SSL</option>
                            <option value="none">无</option>
                        </select>
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >用户名</label
                        >
                        <input
                            type="text"
                            bind:value={config.smtp.username}
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >密码</label
                        >
                        <input
                            type="password"
                            bind:value={config.smtp.password}
                            placeholder="留空不修改"
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >发件人邮箱</label
                        >
                        <input
                            type="text"
                            bind:value={config.smtp.from_email}
                            placeholder="noreply@example.com"
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >发件人名称</label
                        >
                        <input
                            type="text"
                            bind:value={config.smtp.from_name}
                            placeholder="OpenTether"
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >默认收件人</label
                        >
                        <input
                            type="text"
                            bind:value={config.smtp.to_email}
                            placeholder="admin@example.com"
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                </div>
            </div>
        {/if}

        <!-- Update Tab -->
        {#if activeTab === "update"}
            <div class="card space-y-4">
                <h3 class="font-semibold text-slate-800">自动更新配置</h3>
                <label class="flex items-center gap-2 text-sm">
                    <input
                        type="checkbox"
                        bind:checked={config.update.enabled}
                        class="rounded"
                    />
                    启用自动更新检查
                </label>
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >检查间隔</label
                        >
                        <input
                            type="text"
                            bind:value={config.update.check_interval}
                            placeholder="24h"
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-xs font-medium text-slate-600 mb-1"
                            >GitHub 仓库</label
                        >
                        <input
                            type="text"
                            bind:value={config.update.github_repo}
                            placeholder="alaikis/opentether"
                            class="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                    </div>
                </div>
                <div class="flex gap-6">
                    <label class="flex items-center gap-2 text-sm">
                        <input
                            type="checkbox"
                            bind:checked={config.update.auto_backup}
                            class="rounded"
                        />
                        更新前自动备份
                    </label>
                    <label class="flex items-center gap-2 text-sm">
                        <input
                            type="checkbox"
                            bind:checked={config.update.require_approval}
                            class="rounded"
                        />
                        需要管理员审批
                    </label>
                </div>
            </div>
        {/if}
    {/if}
</div>
