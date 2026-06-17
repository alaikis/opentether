<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { Search, RotateCw, Check, X, Play, FileCode, Eye } from "lucide-svelte";

    let all: any[] = [];
    let memories: any[] = [];
    let loading = true;
    let selected: any = null;
    let status = "";
    let keyword = "";
    let renderVars = '{\n  "start_date": "2026-01-01",\n  "end_date": "2026-07-01",\n  "employee_name": "林烽"\n}';
    let rendered = "";
    let testResult = "";
    let dataSourceID = "";
    let execute = false;
    let showDetail = false;

    onMount(load);

    async function load() {
        loading = true;
        try {
            const rows = await api.get<any[]>(`/admin/skills/runtime-memories?status=${status}&limit=500`);
            all = (rows || []).filter((m) => m.type === "text2sql_template");
            filter();
        } finally { loading = false; }
    }

    function filter() {
        memories = all.filter((m) => !keyword || (m.key + m.content).toLowerCase().includes(keyword.toLowerCase()));
    }

    function sqlTemplate(m: any) {
        try { return JSON.parse(m.content).SQLTemplate || m.content; } catch { return m.content; }
    }

    function intent(m: any) {
        try { return JSON.parse(m.content).intent || ""; } catch { return ""; }
    }

    function openDetail(m: any) {
        selected = m;
        rendered = "";
        testResult = "";
        showDetail = true;
    }

    async function renderPreview() {
        if (!selected) return;
        let vars: Record<string, string> = {};
        try { vars = JSON.parse(renderVars); } catch { toast.error("变量 JSON 不合法"); return; }
        const res = await api.post<any>("/admin/sql-templates/test", { sql_template: sqlTemplate(selected), variables: vars, data_source_id: dataSourceID, execute });
        rendered = res.rendered_sql;
        testResult = res.result ? JSON.stringify(res.result, null, 2) : "";
    }

    async function approve(m: any) {
        await api.post(`/admin/skills/runtime-memories/${m.id}/review`, { action: "approve", content: m.content });
        toast.success("模板已发布");
        load();
    }

    async function reject(m: any) {
        await api.post(`/admin/skills/runtime-memories/${m.id}/review`, { action: "reject" });
        toast.success("模板已拒绝");
        load();
    }

    function statusLabel(m: any) {
        if (m.status === "active" || m.source === "admin") return "已发布";
        if (m.status === "pending") return "待审核";
        if (m.status === "rejected" || m.source === "rejected") return "已拒绝";
        return m.status || m.source || "未知";
    }

    function statusClass(m: any) {
        if (m.status === "active" || m.source === "admin") return "bg-green-100 text-green-700";
        if (m.status === "pending") return "bg-amber-100 text-amber-700";
        return "bg-slate-100 text-slate-600";
    }
</script>

<div class="space-y-4">
    <div class="flex items-center justify-between">
        <div>
            <h1 class="text-2xl font-bold text-slate-800">SQL 模板管理</h1>
            <p class="text-sm text-slate-500 mt-1">参数化查询模板，共 {memories.length} 条</p>
        </div>
        <button class="btn-primary flex items-center gap-2" on:click={load}><RotateCw size={14} />刷新</button>
    </div>

    <div class="flex gap-3 flex-wrap">
        <select class="input max-w-[160px]" bind:value={status} on:change={load}>
            <option value="">全部状态</option>
            <option value="pending">待审核</option>
            <option value="approved">已发布</option>
            <option value="rejected">已拒绝</option>
        </select>
        <div class="relative flex-1 max-w-md">
            <Search size={14} class="absolute left-3 top-2.5 text-slate-400" />
            <input class="input pl-8" placeholder="搜索 key / content / intent" bind:value={keyword} on:input={filter} />
        </div>
    </div>

    <div class="grid {showDetail ? 'lg:grid-cols-2' : ''} gap-4">
        <div class="card overflow-x-auto">
            {#if loading}
                <div class="text-slate-400 p-4 text-sm">加载中...</div>
            {:else if memories.length === 0}
                <div class="text-center py-8 text-slate-400"><FileCode size={32} class="mx-auto mb-2" />暂无模板</div>
            {:else}
                <table class="w-full text-sm">
                    <thead><tr class="border-b text-left text-xs text-slate-500"><th class="py-2 px-3 font-medium">Key</th><th class="py-2 px-3 font-medium">状态</th><th class="py-2 px-3 font-medium">意图</th><th class="py-2 px-3 font-medium">置信度</th><th class="py-2 px-3 font-medium">操作</th></tr></thead>
                    <tbody>
                        {#each memories as m}
                            <tr class="border-b hover:bg-slate-50 {selected?.id === m.id ? 'bg-primary-50' : ''}">
                                <td class="py-2 px-3"><span class="font-mono text-xs">{m.key}</span></td>
                                <td class="py-2 px-3"><span class="text-xs px-2 py-0.5 rounded-full {statusClass(m)}">{statusLabel(m)}</span></td>
                                <td class="py-2 px-3 text-xs text-slate-600">{intent(m) || '-'}</td>
                                <td class="py-2 px-3 text-xs text-slate-500">{m.confidence?.toFixed(2) || '-'}</td>
                                <td class="py-2 px-3">
                                    <button class="inline-flex items-center gap-1 text-xs text-primary-600 hover:underline" on:click={() => openDetail(m)}><Eye size={12} />查看</button>
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            {/if}
        </div>

        {#if showDetail && selected}
            <div class="card space-y-4">
                <div class="flex items-center justify-between">
                    <h3 class="font-semibold">{selected.key}</h3>
                    <button class="text-xs text-slate-400" on:click={() => (showDetail = false)}>✕</button>
                </div>
                <div><div class="text-xs text-slate-500">意图: {intent(selected)} | 来源: {selected.source} | 置信度: {selected.confidence?.toFixed(2)}</div></div>
                <textarea class="input font-mono text-xs" rows="8" bind:value={selected.content} />
                <div class="grid grid-cols-2 gap-3">
                    <textarea class="input font-mono text-xs" rows="4" bind:value={renderVars} placeholder="变量 JSON" />
                    <div class="space-y-2">
                        <input class="input text-xs" placeholder="data_source_id" bind:value={dataSourceID} />
                        <label class="inline-flex items-center gap-2 text-xs"><input type="checkbox" bind:checked={execute} />执行查询</label>
                        <button class="btn-primary text-xs w-full" on:click={renderPreview}><Play size={12} />渲染预览</button>
                    </div>
                </div>
                {#if rendered}
                    <pre class="bg-slate-950 text-slate-100 p-3 rounded-lg text-xs overflow-auto max-h-32">{rendered}</pre>
                {/if}
                {#if testResult}
                    <pre class="bg-slate-950 text-slate-100 p-3 rounded-lg text-xs overflow-auto max-h-48">{testResult}</pre>
                {/if}
                <div class="flex gap-2 pt-2">
                    <button class="btn-primary text-xs" on:click={() => approve(selected)}><Check size={12} />发布</button>
                    <button class="px-3 py-1.5 border rounded-lg text-xs text-amber-600" on:click={() => reject(selected)}><X size={12} />拒绝</button>
                </div>
            </div>
        {/if}
    </div>
</div>