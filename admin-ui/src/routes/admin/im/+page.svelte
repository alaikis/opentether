<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { Plus, Pencil, Trash2, CheckCircle, XCircle } from "lucide-svelte";

    interface IMConfig {
        id: string;
        platform: string;
        name: string;
        app_id: string;
        webhook_url: string;
        callback_url: string;
        enabled: boolean;
        created_at: string;
    }

    let configs: IMConfig[] = [];
    let loading = true;
    let error = "";

    let showModal = false;
    let editingConfig: IMConfig | null = null;
    let saving = false;
    let testing = false;
    let testResult = "";

    let formPlatform = "wecom";
    let formName = "";
    let formAppID = "";
    let formAppSecret = "";
    let formToken = "";
    let formWebhookURL = "";
    let formCallbackURL = "";
    let formEnabled = true;

    const platformOptions = [
        {
            value: "wecom",
            label: "企业微信",
            icon: "💬",
            color: "bg-emerald-50 text-emerald-600",
        },
        {
            value: "personal_wechat",
            label: "个人微信",
            icon: "💚",
            color: "bg-green-50 text-green-600",
        },
        {
            value: "feishu",
            label: "飞书",
            icon: "🐦",
            color: "bg-blue-50 text-blue-600",
        },
        {
            value: "dingtalk",
            label: "钉钉",
            icon: "📌",
            color: "bg-sky-50 text-sky-600",
        },
        {
            value: "whatsapp_personal",
            label: "个人 WhatsApp",
            icon: "📱",
            color: "bg-teal-50 text-teal-600",
        },
        {
            value: "whatsapp_business",
            label: "企业 WhatsApp",
            icon: "🏢",
            color: "bg-cyan-50 text-cyan-600",
        },
        {
            value: "ilink",
            label: "iLink",
            icon: "🔗",
            color: "bg-purple-50 text-purple-600",
        },
    ];

    onMount(async () => {
        await loadConfigs();
    });

    async function loadConfigs() {
        loading = true;
        error = "";
        try {
            const data = await api.get<IMConfig[]>("/admin/im/configs");
            configs = Array.isArray(data) ? data : [];
        } catch (e: any) {
            error = e.message || "加载 IM 配置失败";
            configs = [];
        } finally {
            loading = false;
        }
    }

    function openAddModal() {
        editingConfig = null;
        formPlatform = "wecom";
        formName = "";
        formAppID = "";
        formAppSecret = "";
        formToken = "";
        formWebhookURL = "";
        formCallbackURL = "";
        formEnabled = true;
        testResult = "";
        showModal = true;
    }

    function openEditModal(c: IMConfig) {
        editingConfig = c;
        formPlatform = c.platform;
        formName = c.name;
        formAppID = c.app_id || "";
        formAppSecret = "";
        formToken = "";
        formWebhookURL = c.webhook_url || "";
        formCallbackURL = c.callback_url || "";
        formEnabled = c.enabled;
        testResult = "";
        showModal = true;
    }

    function closeModal() {
        showModal = false;
        editingConfig = null;
    }

    async function handleSave() {
        if (!formName) {
            toast.error("请填写配置名称");
            return;
        }
        saving = true;
        try {
            const body: Record<string, any> = {
                platform: formPlatform,
                name: formName,
                app_id: formAppID,
                webhook_url: formWebhookURL,
                callback_url: formCallbackURL,
                enabled: formEnabled,
            };
            if (formAppSecret) body.app_secret = formAppSecret;
            if (formToken) body.token = formToken;

            if (editingConfig) {
                await api.put(`/admin/im/configs/${editingConfig.id}`, body);
                toast.success("配置已更新");
            } else {
                await api.post("/admin/im/configs", body);
                toast.success("配置已创建");
            }
            closeModal();
            await loadConfigs();
        } catch (e: any) {
            toast.error(e.message || "保存失败");
        } finally {
            saving = false;
        }
    }

    async function handleTest(c: IMConfig) {
        testing = true;
        try {
            const data = await api.post<any>(`/admin/im/configs/${c.id}/test`);
            toast.success(data.message || "测试成功");
        } catch (e: any) {
            toast.error(e.message || "测试失败");
        } finally {
            testing = false;
        }
    }

    async function handleToggle(c: IMConfig) {
        try {
            await api.put(`/admin/im/configs/${c.id}`, { enabled: !c.enabled });
            toast.success(c.enabled ? "已停用" : "已启用");
            await loadConfigs();
        } catch (e: any) {
            toast.error(e.message || "操作失败");
        }
    }

    async function handleDelete(c: IMConfig) {
        if (!confirm(`确定删除 IM 配置 "${c.name}" 吗？`)) return;
        try {
            await api.delete(`/admin/im/configs/${c.id}`);
            toast.success("配置已删除");
            await loadConfigs();
        } catch (e: any) {
            toast.error(e.message || "删除失败");
        }
    }

    function getPlatformInfo(platform: string) {
        return (
            platformOptions.find((p) => p.value === platform) ||
            platformOptions[0]
        );
    }
</script>

<svelte:head><title>IM 配置 - OpenTether</title></svelte:head>

<div class="card">
    <div class="flex items-center justify-between mb-6">
        <div>
            <h2 class="text-xl font-bold text-slate-800">IM 配置</h2>
            <p class="text-sm text-slate-500 mt-1">企业微信、飞书、钉钉集成</p>
        </div>
        <button
            class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 transition-colors flex items-center gap-1.5"
            on:click={openAddModal}
        >
            <Plus size={16} />
            添加配置
        </button>
    </div>

    {#if error}
        <div
            class="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm"
        >
            {error}
            <button class="ml-2 underline" on:click={loadConfigs}>重试</button>
        </div>
    {/if}

    {#if loading}
        <div class="text-center py-12 text-slate-400">加载中...</div>
    {:else if configs.length === 0}
        <div class="text-center py-12 space-y-3">
            <div class="text-4xl">💬</div>
            <p class="text-slate-500">暂无 IM 配置</p>
            <p class="text-sm text-slate-400">点击"添加配置"开始集成</p>
        </div>
    {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {#each configs as c}
                {@const info = getPlatformInfo(c.platform)}
                <div
                    class="p-5 rounded-xl border border-slate-200 hover:border-primary-200 hover:shadow-sm transition-all"
                >
                    <div class="flex items-start justify-between mb-3">
                        <div class="flex items-center gap-2.5">
                            <div
                                class="w-10 h-10 rounded-lg {info.color} flex items-center justify-center text-xl"
                            >
                                {info.icon}
                            </div>
                            <div>
                                <h3 class="font-semibold text-sm">{c.name}</h3>
                                <p class="text-xs text-slate-400">
                                    {info.label}
                                </p>
                            </div>
                        </div>
                        <div class="flex items-center gap-1">
                            {#if c.enabled}
                                <CheckCircle
                                    size={16}
                                    class="text-emerald-500"
                                />
                            {:else}
                                <XCircle size={16} class="text-slate-300" />
                            {/if}
                        </div>
                    </div>

                    <div class="text-xs text-slate-500 space-y-1 mb-3">
                        {#if c.app_id}
                            <div>App ID: {c.app_id}</div>
                        {/if}
                        {#if c.callback_url}
                            <div class="truncate">回调: {c.callback_url}</div>
                        {/if}
                    </div>

                    <div
                        class="flex items-center justify-between gap-1 pt-3 border-t border-slate-100"
                    >
                        <div class="flex gap-1">
                            <button
                                class="px-2 py-1 text-xs rounded-md {c.enabled
                                    ? 'bg-slate-100 text-slate-600 hover:bg-slate-200'
                                    : 'bg-emerald-50 text-emerald-600 hover:bg-emerald-100'} transition-colors"
                                on:click={() => handleToggle(c)}
                            >
                                {c.enabled ? "停用" : "启用"}
                            </button>
                            <button
                                class="px-2 py-1 text-xs rounded-md bg-slate-100 text-slate-600 hover:bg-slate-200 transition-colors"
                                on:click={() => handleTest(c)}
                                disabled={testing}
                            >
                                测试
                            </button>
                        </div>
                        <div class="flex gap-1">
                            <button
                                class="p-1 text-slate-400 hover:text-primary-600 hover:bg-primary-50 rounded transition-colors"
                                title="编辑"
                                on:click={() => openEditModal(c)}
                            >
                                <Pencil size={13} />
                            </button>
                            <button
                                class="p-1 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors"
                                title="删除"
                                on:click={() => handleDelete(c)}
                            >
                                <Trash2 size={13} />
                            </button>
                        </div>
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>

<!-- Modal -->
{#if showModal}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        on:click|self={closeModal}
        on:keydown={(e) => e.key === "Escape" && closeModal()}
    >
        <div
            class="bg-white rounded-2xl shadow-xl border border-slate-100 w-full max-w-lg max-h-[85vh] overflow-y-auto p-6"
        >
            <h3 class="text-lg font-bold text-slate-800 mb-4">
                {editingConfig ? "编辑配置" : "添加配置"}
            </h3>

            <div class="space-y-4">
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                        >
                            平台
                        </label>
                        <select
                            bind:value={formPlatform}
                            disabled={!!editingConfig}
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200 disabled:bg-slate-50"
                        >
                            {#each platformOptions as opt}
                                <option value={opt.value}
                                    >{opt.icon} {opt.label}</option
                                >
                            {/each}
                        </select>
                    </div>
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                        >
                            名称 <span class="text-red-500">*</span>
                        </label>
                        <input
                            type="text"
                            bind:value={formName}
                            placeholder="配置名称"
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                        />
                    </div>
                </div>

                <div>
                    <label
                        class="block text-sm font-medium text-slate-700 mb-1"
                    >
                        App ID
                    </label>
                    <input
                        type="text"
                        bind:value={formAppID}
                        placeholder="应用的 App ID / Corp ID"
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                    />
                </div>

                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                        >
                            App Secret {editingConfig ? "(留空不修改)" : ""}
                        </label>
                        <input
                            type="password"
                            bind:value={formAppSecret}
                            placeholder="应用密钥"
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                        >
                            Token
                        </label>
                        <input
                            type="password"
                            bind:value={formToken}
                            placeholder="验证 Token"
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                        />
                    </div>
                </div>

                <div>
                    <label
                        class="block text-sm font-medium text-slate-700 mb-1"
                    >
                        Webhook URL
                    </label>
                    <input
                        type="text"
                        bind:value={formWebhookURL}
                        placeholder="https://..."
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                    />
                </div>

                <div>
                    <label
                        class="block text-sm font-medium text-slate-700 mb-1"
                    >
                        回调 URL
                    </label>
                    <input
                        type="text"
                        bind:value={formCallbackURL}
                        placeholder="服务器回调地址"
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                    />
                </div>

                <div>
                    <label
                        class="flex items-center gap-2 text-sm text-slate-700 cursor-pointer"
                    >
                        <input
                            type="checkbox"
                            bind:checked={formEnabled}
                            class="rounded"
                        />
                        启用此配置
                    </label>
                </div>
            </div>

            <div
                class="flex justify-end gap-2 mt-6 pt-4 border-t border-slate-100"
            >
                <button
                    class="px-4 py-2 bg-slate-100 text-slate-700 rounded-lg text-sm font-medium hover:bg-slate-200 transition-colors"
                    on:click={closeModal}
                    disabled={saving}
                >
                    取消
                </button>
                <button
                    class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 transition-colors disabled:opacity-50"
                    on:click={handleSave}
                    disabled={saving}
                >
                    {#if saving}保存中...{:else}保存{/if}
                </button>
            </div>
        </div>
    </div>
{/if}
