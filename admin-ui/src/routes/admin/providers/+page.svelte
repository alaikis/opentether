<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";

    interface ModelConfig {
        name: string;
        enabled: boolean;
        is_vision: boolean;
        is_reasoning: boolean;
        is_function_calling: boolean;
        max_tokens: number;
        temperature: number;
    }

    interface Provider {
        id: string;
        provider_type: string;
        provider_name: string;
        api_base: string;
        api_key: string;
        model: string;
        enabled: boolean;
        priority: number;
        config: string;
        created_at: string;
        updated_at: string;
    }

    let providers: Provider[] = [];
    let loading = true;
    let showModal = false;
    let editingProvider: Provider | null = null;
    let saving = false;
    let testing = false;
    let testResult = "";

    // 表单数据
    let formName = "";
    let formType = "openai";
    let formApiBase = "";
    let formApiKey = "";
    let formEnabled = true;
    let formPriority = 0;
    let formModels: ModelConfig[] = [
        {
            name: "",
            enabled: true,
            is_vision: false,
            is_reasoning: false,
            is_function_calling: false,
            max_tokens: 4096,
            temperature: 0.7,
        },
    ];
    let formEnableStreaming = true;
    let formTimeout = 120;

    const providerTypes = [
        { value: "openai", label: "OpenAI / 兼容 API" },
        { value: "azure", label: "Azure OpenAI" },
        { value: "anthropic", label: "Anthropic Claude" },
        { value: "google", label: "Google Gemini" },
        { value: "deepseek", label: "DeepSeek" },
        { value: "local", label: "本地模型 (Ollama/vLLM)" },
        { value: "other", label: "其他" },
    ];

    onMount(async () => {
        await loadProviders();
    });

    async function loadProviders() {
        loading = true;
        try {
            const data = await api.get<any>("/admin/providers");
            providers = data.providers || data || [];
        } catch (e: any) {
            toast.error(e.message || "加载提供商列表失败");
        } finally {
            loading = false;
        }
    }

    function openAddModal() {
        editingProvider = null;
        formName = "";
        formType = "openai";
        formApiBase = "";
        formApiKey = "";
        formEnabled = true;
        formPriority = providers.length;
        formModels = [
            {
                name: "",
                enabled: true,
                is_vision: false,
                is_reasoning: false,
                is_function_calling: false,
                max_tokens: 4096,
                temperature: 0.7,
            },
        ];
        formEnableStreaming = true;
        formTimeout = 120;
        testResult = "";
        showModal = true;
    }

    function openEditModal(p: Provider) {
        editingProvider = p;
        formName = p.provider_name;
        formType = p.provider_type;
        formApiBase = p.api_base || "";
        formApiKey = "";
        formEnabled = p.enabled;
        formPriority = p.priority || 0;

        try {
            const cfg = JSON.parse(p.config || "{}");
            formModels = cfg.models || [
                {
                    name: p.model || "",
                    enabled: true,
                    is_vision: false,
                    is_reasoning: false,
                    is_function_calling: false,
                    max_tokens: 4096,
                    temperature: 0.7,
                },
            ];
            formEnableStreaming = cfg.enable_streaming !== false;
            formTimeout = cfg.timeout || 120;
        } catch {
            formModels = [
                {
                    name: p.model || "",
                    enabled: true,
                    is_vision: false,
                    is_reasoning: false,
                    is_function_calling: false,
                    max_tokens: 4096,
                    temperature: 0.7,
                },
            ];
            formEnableStreaming = true;
            formTimeout = 120;
        }
        testResult = "";
        showModal = true;
    }

    function addModel() {
        formModels = [
            ...formModels,
            {
                name: "",
                enabled: true,
                is_vision: false,
                is_reasoning: false,
                is_function_calling: false,
                max_tokens: 4096,
                temperature: 0.7,
            },
        ];
    }

    function removeModel(index: number) {
        if (formModels.length <= 1) return;
        formModels = formModels.filter((_, i) => i !== index);
    }

    function parseModels(): ModelConfig[] {
        return formModels.filter((m) => m.name.trim() !== "");
    }

    async function handleSave() {
        if (!formName.trim()) {
            toast.error("请输入提供商名称");
            return;
        }

        const models = parseModels();
        if (models.length === 0) {
            toast.error("至少需要配置一个模型");
            return;
        }

        const config = JSON.stringify({
            models,
            enable_streaming: formEnableStreaming,
            timeout: formTimeout,
        });

        saving = true;
        try {
            const body: any = {
                provider_type: formType,
                provider_name: formName.trim(),
                api_base: formApiBase.trim(),
                api_key: formApiKey.trim(),
                model: models[0].name,
                enabled: formEnabled,
                priority: formPriority,
                config,
            };

            if (editingProvider) {
                await api.put(`/admin/providers/${editingProvider.id}`, body);
                toast.success("提供商已更新");
            } else {
                await api.post("/admin/providers", body);
                toast.success("提供商已添加");
            }
            showModal = false;
            await loadProviders();
        } catch (e: any) {
            toast.error(e.message || "保存失败");
        } finally {
            saving = false;
        }
    }

    async function handleTest() {
        if (!formApiBase.trim()) {
            toast.error("请先填写 API 地址");
            return;
        }
        testing = true;
        testResult = "";
        try {
            const models = parseModels();
            const data = await api.post<any>("/admin/providers/test", {
                provider_type: formType,
                api_base: formApiBase.trim(),
                api_key: formApiKey.trim(),
                model: models[0]?.name || "",
            });
            testResult = data?.message || "✅ 连接测试成功";
        } catch (e: any) {
            testResult = `❌ ${e.message || "连接失败"}`;
        } finally {
            testing = false;
        }
    }

    async function handleToggle(provider: Provider) {
        try {
            await api.put(`/admin/providers/${provider.id}`, {
                ...provider,
                enabled: !provider.enabled,
            });
            await loadProviders();
        } catch (e: any) {
            toast.error(e.message || "操作失败");
        }
    }

    async function handleDelete(provider: Provider) {
        if (!confirm(`确定要删除 "${provider.provider_name}" 吗？`)) return;
        try {
            await api.delete(`/admin/providers/${provider.id}`);
            toast.success("已删除");
            await loadProviders();
        } catch (e: any) {
            toast.error(e.message || "删除失败");
        }
    }

    function getTypeLabel(type: string): string {
        return providerTypes.find((t) => t.value === type)?.label || type;
    }

    function parseProviderModels(p: Provider): ModelConfig[] {
        try {
            const cfg = JSON.parse(p.config || "{}");
            return cfg.models || [];
        } catch {
            return [];
        }
    }
</script>

<svelte:head><title>LLM 提供商 - OpenTether</title></svelte:head>

<div class="space-y-6">
    <div class="card">
        <div class="flex items-center justify-between mb-6">
            <div>
                <h2 class="text-xl font-bold text-slate-800">LLM 提供商</h2>
                <p class="text-sm text-slate-500 mt-1">
                    配置和管理 AI 大语言模型服务。请先至少配置一个可用的提供商。
                </p>
            </div>
            <button
                class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 transition-colors"
                on:click={openAddModal}
            >
                + 添加提供商
            </button>
        </div>

        {#if loading}
            <div class="text-center py-12 text-slate-400">加载中...</div>
        {:else if providers.length === 0}
            <div class="text-center py-12 space-y-3">
                <div class="text-4xl">🤖</div>
                <p class="text-slate-500">还没有配置任何 LLM 提供商</p>
                <p class="text-sm text-slate-400">
                    点击上方"添加提供商"开始配置
                </p>
            </div>
        {:else}
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                {#each providers as p}
                    <div
                        class="p-5 rounded-xl border {p.enabled
                            ? 'border-slate-200'
                            : 'border-slate-100 opacity-60'} hover:shadow-md transition-shadow cursor-pointer"
                        on:click={() => openEditModal(p)}
                    >
                        <div class="flex items-center justify-between mb-3">
                            <div class="flex items-center gap-3">
                                <div
                                    class="w-10 h-10 rounded-lg {p.enabled
                                        ? 'bg-emerald-50'
                                        : 'bg-slate-100'} flex items-center justify-center text-xl"
                                >
                                    {p.provider_type === "openai"
                                        ? "🤖"
                                        : p.provider_type === "anthropic"
                                          ? "🧠"
                                          : p.provider_type === "google"
                                            ? "🌐"
                                            : "🔌"}
                                </div>
                                <div>
                                    <div class="font-semibold">
                                        {p.provider_name}
                                    </div>
                                    <div class="text-xs text-slate-400">
                                        {getTypeLabel(p.provider_type)}
                                    </div>
                                </div>
                            </div>
                            <div class="flex items-center gap-2">
                                <button
                                    class="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-red-50 text-slate-400 hover:text-red-500 transition-colors"
                                    on:click|stopPropagation={() =>
                                        handleDelete(p)}
                                    title="删除">✕</button
                                >
                            </div>
                        </div>
                        <div class="flex flex-wrap gap-1.5">
                            {#each parseProviderModels(p) as m}
                                <span
                                    class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs {p.enabled
                                        ? 'bg-slate-100 text-slate-600'
                                        : 'bg-slate-50 text-slate-400'}"
                                >
                                    {#if m.is_vision}🖼️
                                    {/if}
                                    {#if m.is_reasoning}🧩
                                    {/if}
                                    {m.name}
                                </span>
                            {/each}
                        </div>
                        <div
                            class="flex items-center gap-4 mt-3 pt-3 border-t border-slate-100"
                        >
                            <button
                                class="text-xs {p.enabled
                                    ? 'text-emerald-600'
                                    : 'text-slate-400'} hover:underline"
                                on:click|stopPropagation={() => handleToggle(p)}
                            >
                                <span
                                    class="w-1.5 h-1.5 rounded-full inline-block mr-1 {p.enabled
                                        ? 'bg-emerald-500'
                                        : 'bg-slate-300'}"
                                ></span>
                                {p.enabled ? "已启用" : "已停用"}
                            </button>
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</div>

<!-- 弹窗表单 -->
{#if showModal}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        on:click|self={() => (showModal = false)}
    >
        <div
            class="bg-white rounded-2xl shadow-2xl w-full max-w-2xl max-h-[90vh] overflow-y-auto p-6 m-4"
        >
            <h3 class="text-lg font-bold text-slate-800 mb-4">
                {editingProvider ? "编辑提供商" : "添加提供商"}
            </h3>

            <!-- 基本信息 -->
            <div class="grid grid-cols-2 gap-4 mb-6">
                <div>
                    <label class="block text-sm font-medium text-slate-700 mb-1"
                        >名称 *</label
                    >
                    <input
                        type="text"
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                        placeholder="例如: OpenAI生产环境"
                        bind:value={formName}
                    />
                </div>
                <div>
                    <label class="block text-sm font-medium text-slate-700 mb-1"
                        >类型</label
                    >
                    <select
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm"
                        bind:value={formType}
                    >
                        {#each providerTypes as t}
                            <option value={t.value}>{t.label}</option>
                        {/each}
                    </select>
                </div>
                <div class="col-span-2">
                    <label class="block text-sm font-medium text-slate-700 mb-1"
                        >API 地址</label
                    >
                    <input
                        type="text"
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                        placeholder="https://api.openai.com/v1"
                        bind:value={formApiBase}
                    />
                </div>
                <div class="col-span-2">
                    <label class="block text-sm font-medium text-slate-700 mb-1"
                        >API Key {editingProvider
                            ? "(留空则不修改)"
                            : ""}</label
                    >
                    <input
                        type="password"
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                        placeholder="sk-..."
                        bind:value={formApiKey}
                    />
                </div>
            </div>

            <!-- 模型列表 -->
            <div class="mb-6">
                <div class="flex items-center justify-between mb-3">
                    <label class="text-sm font-medium text-slate-700"
                        >模型列表 *</label
                    >
                    <button
                        class="text-xs text-primary-600 hover:underline"
                        on:click={addModel}>+ 添加模型</button
                    >
                </div>
                <div class="space-y-3">
                    {#each formModels as model, i}
                        <div
                            class="p-3 rounded-lg border border-slate-200 bg-slate-50"
                        >
                            <div class="flex items-center justify-between mb-2">
                                <span class="text-xs font-medium text-slate-500"
                                    >模型 #{i + 1}</span
                                >
                                {#if formModels.length > 1}
                                    <button
                                        class="text-xs text-red-500 hover:underline"
                                        on:click={() => removeModel(i)}
                                        >删除</button
                                    >
                                {/if}
                            </div>
                            <div class="grid grid-cols-3 gap-3 mb-2">
                                <div class="col-span-3">
                                    <input
                                        type="text"
                                        class="w-full px-2 py-1.5 border border-slate-200 rounded text-sm focus:outline-none focus:ring-1 focus:ring-primary-200"
                                        placeholder="模型名称，例如: gpt-4o"
                                        bind:value={model.name}
                                    />
                                </div>
                            </div>
                            <div class="flex flex-wrap gap-2">
                                <label
                                    class="flex items-center gap-1 text-xs cursor-pointer"
                                >
                                    <input
                                        type="checkbox"
                                        bind:checked={model.enabled}
                                        class="rounded"
                                    />
                                    启用
                                </label>
                                <label
                                    class="flex items-center gap-1 text-xs cursor-pointer"
                                    title="支持图像/多模态理解"
                                >
                                    <input
                                        type="checkbox"
                                        bind:checked={model.is_vision}
                                        class="rounded"
                                    />
                                    🖼️ 图像理解
                                </label>
                                <label
                                    class="flex items-center gap-1 text-xs cursor-pointer"
                                    title="支持推理/深度思考"
                                >
                                    <input
                                        type="checkbox"
                                        bind:checked={model.is_reasoning}
                                        class="rounded"
                                    />
                                    🧩 深度推理
                                </label>
                                <label
                                    class="flex items-center gap-1 text-xs cursor-pointer"
                                    title="支持 Function Calling / Tools"
                                >
                                    <input
                                        type="checkbox"
                                        bind:checked={model.is_function_calling}
                                        class="rounded"
                                    />
                                    🔧 函数调用
                                </label>
                                <div class="flex items-center gap-1 text-xs">
                                    <span class="text-slate-400">Token:</span>
                                    <input
                                        type="number"
                                        class="w-16 px-1 py-0.5 border border-slate-200 rounded text-xs"
                                        bind:value={model.max_tokens}
                                        min="256"
                                        step="512"
                                    />
                                </div>
                                <div class="flex items-center gap-1 text-xs">
                                    <span class="text-slate-400">温度:</span>
                                    <input
                                        type="number"
                                        class="w-16 px-1 py-0.5 border border-slate-200 rounded text-xs"
                                        bind:value={model.temperature}
                                        min="0"
                                        max="2"
                                        step="0.1"
                                    />
                                </div>
                            </div>
                        </div>
                    {/each}
                </div>
            </div>

            <!-- 高级选项 -->
            <div class="mb-6">
                <label class="text-sm font-medium text-slate-700 mb-2 block"
                    >高级选项</label
                >
                <div
                    class="grid grid-cols-2 gap-4 p-3 rounded-lg border border-slate-200"
                >
                    <label
                        class="flex items-center gap-2 text-sm cursor-pointer"
                    >
                        <input
                            type="checkbox"
                            bind:checked={formEnableStreaming}
                            class="rounded"
                        />
                        启用流式输出
                    </label>
                    <div class="flex items-center gap-2 text-sm">
                        <span class="text-slate-500">超时(秒):</span>
                        <input
                            type="number"
                            class="w-16 px-2 py-1 border border-slate-200 rounded text-sm"
                            bind:value={formTimeout}
                            min="10"
                            max="600"
                        />
                    </div>
                    <label
                        class="flex items-center gap-2 text-sm cursor-pointer"
                    >
                        <input
                            type="checkbox"
                            bind:checked={formEnabled}
                            class="rounded"
                        />
                        启用此提供商
                    </label>
                </div>
            </div>

            <!-- 测试结果 -->
            {#if testResult}
                <div
                    class="mb-4 p-3 rounded-lg text-sm {testResult.startsWith(
                        '✅',
                    )
                        ? 'bg-emerald-50 text-emerald-700'
                        : 'bg-red-50 text-red-700'}"
                >
                    {testResult}
                </div>
            {/if}

            <!-- 操作按钮 -->
            <div class="flex justify-between items-center">
                <button
                    class="px-4 py-2 text-sm text-slate-600 hover:text-slate-800"
                    on:click={() => (showModal = false)}>取消</button
                >
                <div class="flex gap-2">
                    <button
                        class="px-4 py-2 border border-slate-200 rounded-lg text-sm text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                        on:click={handleTest}
                        disabled={testing}
                    >
                        {testing ? "测试中..." : "测试连接"}
                    </button>
                    <button
                        class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 disabled:opacity-50"
                        on:click={handleSave}
                        disabled={saving}
                    >
                        {saving ? "保存中..." : "保存"}
                    </button>
                </div>
            </div>
        </div>
    </div>
{/if}
