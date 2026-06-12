<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import {
        RefreshCcw,
        Check,
        X,
        Plus,
        Trash2,
        Eye,
        Brain,
    } from "lucide-svelte";

    // ── Route Examples ────────────────────────────
    interface RouteExample {
        id: string;
        text: string;
        route: string;
        intent: string;
        source: string;
        status: string;
        confidence: number;
        use_count: number;
        updated_at: string;
    }

    // ── Text2SQL Templates ────────────────────────
    interface T2STemplate {
        id: string;
        skill_id: string;
        data_source_id: string;
        type: string;
        key: string;
        content: string;
        confidence: number;
        use_count: number;
        source: string;
        last_used_at: string;
        updated_at: string;
    }

    let activeTab: "routes" | "templates" = "routes";

    let examples: RouteExample[] = [];
    let templates: T2STemplate[] = [];
    let loading = true;
    let status = "pending";
    let newText = "";
    let newRoute = "fast_text2sql";
    let newIntent = "";
    let showTemplateContent = "";
    // 编辑模板弹窗
    let editingTemplate: T2STemplate | null = null;
    let editingTemplateSQL = "";
    const statuses = ["pending", "active", "rejected", ""];
    const routes = ["fast_local", "fast_chat", "fast_text2sql", "agent_loop"];

    onMount(() => {
        activeTab === "routes" ? loadExamples() : loadTemplates();
    });

    function switchTab(tab: "routes" | "templates") {
        activeTab = tab;
        loading = true;
        if (tab === "routes") loadExamples();
        else loadTemplates();
    }

    // ── Routes ────────────────────────────────────
    async function loadExamples() {
        loading = true;
        try {
            const qs = status ? `?status=${status}` : "";
            const data = await api.get<{ examples: RouteExample[] }>(
                `/admin/skills/route-examples${qs}`,
            );
            examples = data.examples || [];
        } catch (e: any) {
            toast.error(e.message || "加载失败");
        } finally {
            loading = false;
        }
    }
    async function createExample() {
        if (!newText.trim()) return toast.error("请输入样本文本");
        try {
            await api.post("/admin/skills/route-examples", {
                text: newText,
                route: newRoute,
                intent: newIntent,
                confidence: 1,
            });
            toast.success("已创建");
            newText = "";
            newIntent = "";
            await loadExamples();
        } catch (e: any) {
            toast.error(e.message || "创建失败");
        }
    }
    async function reviewExample(id: string, action: string) {
        try {
            await api.post(`/admin/skills/route-examples/${id}/review`, {
                action,
            });
            toast.success(action === "approve" ? "已通过" : "已拒绝");
            await loadExamples();
        } catch (e: any) {
            toast.error(e.message || "操作失败");
        }
    }

    // ── Templates ─────────────────────────────────
    async function loadTemplates() {
        loading = true;
        try {
            const data = await api.get<any>("/admin/skills/runtime-memories");
            const list = data?.memories || data || [];
            templates = (Array.isArray(list) ? list : []).filter(
                (t: any) => t.type === "text2sql_template",
            );
        } catch (e: any) {
            toast.error(e.message || "加载模板失败");
        } finally {
            loading = false;
        }
    }
    async function reviewTemplate(id: string, action: string) {
        try {
            await api.post(`/admin/skills/runtime-memories/${id}/review`, {
                action,
                confidence: action === "approve" ? 0.9 : 0.1,
            });
            toast.success(action === "approve" ? "已启用" : "已禁用");
            await loadTemplates();
        } catch (e: any) {
            toast.error(e.message || "操作失败");
        }
    }
    async function deleteTemplate(id: string) {
        if (!confirm("确定删除此模板？")) return;
        try {
            await api.delete(`/admin/skills/runtime-memories/${id}`);
            toast.success("已删除");
            await loadTemplates();
        } catch (e: any) {
            toast.error(e.message || "删除失败");
        }
    }
    function parseTemplateContent(content: string): {
        question?: string;
        sql?: string;
    } {
        try {
            const parsed = JSON.parse(content);
            if (parsed.sql_template) return { sql: parsed.sql_template };
            return parsed;
        } catch {
            return {};
        }
    }

    function openEditTemplate(t: T2STemplate) {
        editingTemplate = t;
        editingTemplateSQL = parseTemplateContent(t.content).sql || "";
    }
    function closeEditTemplate() {
        editingTemplate = null;
        editingTemplateSQL = "";
    }
    async function saveTemplate() {
        if (!editingTemplate) return;
        try {
            const parsed = JSON.parse(editingTemplate.content);
            parsed.sql_template = editingTemplateSQL;
            await api.post(
                `/admin/skills/runtime-memories/${editingTemplate.id}/review`,
                {
                    action: "edit",
                    content: JSON.stringify(parsed),
                },
            );
            toast.success("模板已更新");
            closeEditTemplate();
            await loadTemplates();
        } catch (e: any) {
            toast.error(e.message || "编辑失败");
        }
    }
</script>

<svelte:head><title>快路径 - OpenTether</title></svelte:head>

<div class="space-y-5">
    <!-- Tab switcher -->
    <div class="flex items-center gap-1 p-1 bg-slate-100 rounded-xl w-fit">
        <button
            class="px-4 py-2 rounded-lg text-sm font-medium transition-colors {activeTab ===
            'routes'
                ? 'bg-white text-slate-800 shadow-sm'
                : 'text-slate-500 hover:text-slate-700'}"
            on:click={() => switchTab("routes")}
        >
            路由样本
        </button>
        <button
            class="px-4 py-2 rounded-lg text-sm font-medium transition-colors {activeTab ===
            'templates'
                ? 'bg-white text-slate-800 shadow-sm'
                : 'text-slate-500 hover:text-slate-700'}"
            on:click={() => switchTab("templates")}
        >
            Text2SQL 模板
        </button>
    </div>

    {#if activeTab === "routes"}
        <!-- ──── 路由样本 ──── -->
        <div class="card">
            <div class="flex items-center justify-between mb-5">
                <div>
                    <h2 class="text-xl font-bold text-slate-800">
                        FastPath 路由样本
                    </h2>
                    <p class="text-sm text-slate-500 mt-1">
                        管理 TF-IDF FastPathClassifier 的样本和审核队列
                    </p>
                </div>
                <button
                    class="px-3 py-2 rounded-lg bg-slate-100 text-sm flex items-center gap-1"
                    on:click={loadExamples}
                    ><RefreshCcw size={15} /> 刷新</button
                >
            </div>
            <div
                class="grid grid-cols-1 md:grid-cols-[1fr_180px_160px_auto] gap-2 mb-5"
            >
                <input
                    class="px-3 py-2 border rounded-lg text-sm"
                    placeholder="样本文本"
                    bind:value={newText}
                />
                <select
                    class="px-3 py-2 border rounded-lg text-sm"
                    bind:value={newRoute}
                >
                    {#each routes as r}<option value={r}>{r}</option>{/each}
                </select>
                <input
                    class="px-3 py-2 border rounded-lg text-sm"
                    placeholder="intent"
                    bind:value={newIntent}
                />
                <button
                    class="px-3 py-2 bg-primary-600 text-white rounded-lg text-sm"
                    on:click={createExample}><Plus size={15} /> 新增</button
                >
            </div>
            <div class="flex items-center gap-2 mb-4">
                <span class="text-sm text-slate-500">状态</span>
                <select
                    bind:value={status}
                    on:change={loadExamples}
                    class="border rounded-lg px-3 py-1.5 text-sm"
                >
                    {#each statuses as s}<option value={s}>{s || "全部"}</option
                        >{/each}
                </select>
            </div>
            {#if loading}<div class="text-center py-10 text-slate-400">
                    加载中...
                </div>
            {:else if examples.length === 0}<div
                    class="text-center py-10 text-slate-400"
                >
                    暂无样本
                </div>
            {:else}
                <table class="w-full text-sm">
                    <thead
                        ><tr class="text-left border-b text-slate-500"
                            ><th class="py-2 px-3">文本</th><th
                                class="py-2 px-3">Route</th
                            ><th class="py-2 px-3">状态</th><th
                                class="py-2 px-3">置信度</th
                            ><th class="py-2 px-3 text-right">操作</th></tr
                        ></thead
                    >
                    <tbody>
                        {#each examples as ex}
                            <tr class="border-b hover:bg-slate-50">
                                <td
                                    class="py-2 px-3 max-w-lg truncate"
                                    title={ex.text}>{ex.text}</td
                                >
                                <td class="py-2 px-3 font-mono text-xs"
                                    >{ex.route}
                                    <div class="text-slate-400">
                                        {ex.intent}
                                    </div></td
                                >
                                <td class="py-2 px-3"
                                    ><span
                                        class="text-xs px-2 py-1 rounded bg-slate-100"
                                        >{ex.source}/{ex.status}</span
                                    ></td
                                >
                                <td class="py-2 px-3"
                                    >{Math.round(
                                        (ex.confidence || 0) * 100,
                                    )}%</td
                                >
                                <td class="py-2 px-3 text-right">
                                    {#if ex.status !== "active"}<button
                                            class="p-1.5 text-green-600"
                                            title="通过"
                                            on:click={() =>
                                                reviewExample(ex.id, "approve")}
                                            ><Check size={15} /></button
                                        >{/if}
                                    {#if ex.status !== "rejected"}<button
                                            class="p-1.5 text-red-500"
                                            title="拒绝"
                                            on:click={() =>
                                                reviewExample(ex.id, "reject")}
                                            ><X size={15} /></button
                                        >{/if}
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            {/if}
        </div>
    {:else}
        <!-- ──── Text2SQL 模板 ──── -->
        <div class="card">
            <div class="flex items-center justify-between mb-5">
                <div>
                    <h2 class="text-xl font-bold text-slate-800">
                        Text2SQL 模板
                    </h2>
                    <p class="text-sm text-slate-500 mt-1">
                        自动生成的 SQL 模板，可通过审核提升为高置信快速通道
                    </p>
                </div>
                <button
                    class="px-3 py-2 rounded-lg bg-slate-100 text-sm flex items-center gap-1"
                    on:click={loadTemplates}
                    ><RefreshCcw size={15} /> 刷新</button
                >
            </div>
            {#if loading}<div class="text-center py-10 text-slate-400">
                    加载中...
                </div>
            {:else if templates.length === 0}<div
                    class="text-center py-10 text-slate-400"
                >
                    暂无模板。通过对话成功执行 text2sql 查询后，模板会自动生成。
                </div>
            {:else}
                <table class="w-full text-sm">
                    <thead
                        ><tr class="text-left border-b text-slate-500"
                            ><th class="py-2 px-3">模板Key</th><th
                                class="py-2 px-3">来源/置信度</th
                            ><th class="py-2 px-3">使用</th><th
                                class="py-2 px-3">SQL模板</th
                            ><th class="py-2 px-3 text-right">操作</th></tr
                        ></thead
                    >
                    <tbody>
                        {#each templates as t}
                            <tr class="border-b hover:bg-slate-50">
                                <td
                                    class="py-2 px-3 font-mono text-xs max-w-[200px] truncate"
                                    title={t.key}>{t.key}</td
                                >
                                <td class="py-2 px-3">
                                    <span
                                        class="text-xs px-2 py-1 rounded {t.source ===
                                        'admin'
                                            ? 'bg-emerald-50 text-emerald-700'
                                            : 'bg-slate-100 text-slate-600'}"
                                        >{t.source}</span
                                    >
                                    <span class="ml-1 text-xs text-slate-400"
                                        >{(t.confidence * 100).toFixed(
                                            0,
                                        )}%</span
                                    >
                                </td>
                                <td class="py-2 px-3 text-xs text-slate-500"
                                    >{t.use_count} 次</td
                                >
                                <td class="py-2 px-3 max-w-[400px]">
                                    <code
                                        class="text-[11px] text-slate-500 bg-slate-50 px-1.5 py-0.5 rounded block truncate"
                                        >{parseTemplateContent(t.content).sql ||
                                            t.content.slice(0, 80)}</code
                                    >
                                </td>
                                <td class="py-2 px-3 text-right">
                                    <button
                                        class="p-1.5 text-blue-500"
                                        title="编辑SQL"
                                        on:click={() => openEditTemplate(t)}
                                        ><Eye size={15} /></button
                                    >
                                    {#if t.confidence < 0.9 || t.source !== "admin"}
                                        <button
                                            class="p-1.5 text-green-600"
                                            title="提升为高置信"
                                            on:click={() =>
                                                reviewTemplate(t.id, "approve")}
                                            ><Check size={15} /></button
                                        >
                                    {/if}
                                    <button
                                        class="p-1.5 text-red-500"
                                        title="禁用"
                                        on:click={() =>
                                            reviewTemplate(t.id, "reject")}
                                        ><X size={15} /></button
                                    >
                                    <button
                                        class="p-1.5 text-slate-400 hover:text-red-500"
                                        title="删除"
                                        on:click={() => deleteTemplate(t.id)}
                                        ><Trash2 size={15} /></button
                                    >
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            {/if}
        </div>

        <!-- 编辑模板弹窗 -->
        {#if editingTemplate}
            <div
                class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
                on:click|self={closeEditTemplate}
            >
                <div
                    class="bg-white rounded-2xl shadow-2xl w-full max-w-2xl max-h-[80vh] overflow-y-auto p-6 m-4"
                >
                    <h3 class="text-lg font-bold text-slate-800 mb-4">
                        编辑 SQL 模板
                    </h3>
                    <div class="mb-3">
                        <label
                            class="block text-xs font-medium text-slate-500 mb-1"
                            >模板 Key</label
                        >
                        <code
                            class="text-sm text-slate-700 bg-slate-50 px-2 py-1 rounded"
                            >{editingTemplate.key}</code
                        >
                    </div>
                    <div class="mb-4">
                        <label
                            class="block text-xs font-medium text-slate-500 mb-1"
                            >SQL 模板</label
                        >
                        <textarea
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm font-mono min-h-[200px] focus:outline-none focus:ring-2 focus:ring-primary-200"
                            bind:value={editingTemplateSQL}
                            placeholder="SELECT ..."
                        ></textarea>
                    </div>
                    <div class="flex justify-end gap-2">
                        <button
                            class="px-4 py-2 text-sm text-slate-600 hover:text-slate-800"
                            on:click={closeEditTemplate}>取消</button
                        >
                        <button
                            class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700"
                            on:click={saveTemplate}>保存</button
                        >
                    </div>
                </div>
            </div>
        {/if}
    {/if}
</div>
