<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { Plus, Pencil, Trash2, Search, Link, Zap } from "lucide-svelte";

    interface DataSource {
        id: string;
        name: string;
        source_type: string;
        host: string;
        port: number;
        user: string;
        database: string;
        schema_info: string;
        table_relations: string;
        enabled: boolean;
        created_at: string;
    }
    interface Relation {
        from_table: string;
        from_column: string;
        to_table: string;
        to_column: string;
    }
    interface SchemaTable {
        name: string;
        columns: { name: string; type?: string; key_type?: string }[];
    }

    let items: DataSource[] = [];
    let loading = true;
    let error = "";
    let searchQuery = "";
    let showModal = false;
    let editing: DataSource | null = null;
    let saving = false;
    let testing = false;
    let testResult = "";
    let fName = "";
    let fType = "mysql";
    let fHost = "";
    let fPort = 3306;
    let fUser = "";
    let fPassword = "";
    let fDatabase = "";
    let fEnabled = true;

    let showRelationsModal = false;
    let relationsDS: DataSource | null = null;
    let relationsJson = "";
    let relationsParsed: Relation[] = [];
    let savingRelations = false;
    let showSchemaModal = false;
    let schemaDS: DataSource | null = null;
    const types = [
        "mysql",
        "postgres",
        "sqlite",
        "mongodb",
        "api",
        "csv",
        "other",
    ];

    let filteredItems: DataSource[] = [];
    $: {
        let q = searchQuery.toLowerCase();
        filteredItems = q
            ? items.filter(
                  (ds) =>
                      ds.name.toLowerCase().includes(q) ||
                      ds.source_type.toLowerCase().includes(q) ||
                      (ds.host || "").toLowerCase().includes(q),
              )
            : items;
    }
    onMount(() => {
        loadData();
    });
    async function loadData() {
        loading = true;
        error = "";
        try {
            let d = await api.get<any>("/admin/datasources");
            items = d.datasources || d || [];
        } catch (e: any) {
            error = e.message || "加载失败";
        } finally {
            loading = false;
        }
    }
    function openAdd() {
        editing = null;
        fName = "";
        fType = "mysql";
        fHost = "";
        fPort = 3306;
        fUser = "";
        fPassword = "";
        fDatabase = "";
        fEnabled = true;
        showModal = true;
    }
    function openEdit(d: DataSource) {
        editing = d;
        fName = d.name;
        fType = d.source_type;
        fHost = d.host || "";
        fPort = d.port || 3306;
        fUser = d.user || "";
        fPassword = "";
        fDatabase = d.database || "";
        fEnabled = d.enabled;
        showModal = true;
    }
    async function handleSave() {
        if (!fName) {
            toast.error("请填写名称");
            return;
        }
        saving = true;
        try {
            let b: any = {
                name: fName,
                source_type: fType,
                host: fHost,
                port: fPort,
                user: fUser,
                password: fPassword,
                database: fDatabase,
                enabled: fEnabled,
            };
            if (editing) {
                await api.put(`/admin/datasources/${editing.id}`, b);
                toast.success("已更新");
            } else {
                await api.post("/admin/datasources", b);
                toast.success("已创建");
            }
            showModal = false;
            await loadData();
        } catch (e: any) {
            toast.error(e.message || "保存失败");
        } finally {
            saving = false;
        }
    }
    async function handleDelete(d: DataSource) {
        if (!confirm(`确定删除"${d.name}"?`)) return;
        try {
            await api.delete(`/admin/datasources/${d.id}`);
            toast.success("已删除");
            await loadData();
        } catch (e: any) {
            toast.error(e.message || "删除失败");
        }
    }
    async function handleToggle(d: DataSource) {
        try {
            await api.put(`/admin/datasources/${d.id}`, {
                enabled: !d.enabled,
            });
            toast.success(d.enabled ? "已停用" : "已启用");
            await loadData();
        } catch (e: any) {
            toast.error(e.message || "操作失败");
        }
    }
    async function handleTest() {
        testing = true;
        testResult = "";
        try {
            // 新建数据源：用表单凭据测试；已有数据源：用 ID 测试
            if (editing) {
                let data = await api.post<any>(
                    `/admin/datasources/${editing.id}/test`,
                );
                testResult =
                    data.message || (data.success ? "连接成功" : "连接失败");
            } else {
                let body = {
                    source_type: fType,
                    host: fHost,
                    port: fPort,
                    user: fUser,
                    password: fPassword,
                    database: fDatabase,
                };
                let data = await api.post<any>("/admin/datasources/test", body);
                testResult =
                    data.message || (data.success ? "连接成功" : "连接失败");
            }
        } catch (e: any) {
            testResult = e.message || "测试失败";
        } finally {
            testing = false;
        }
    }
    async function handleAnalyze(d: DataSource) {
        try {
            toast.info("正在分析...");
            await api.post(`/admin/datasources/${d.id}/analyze`);
            toast.success("分析完成");
            await loadData();
        } catch (e: any) {
            toast.error(e.message || "分析失败");
        }
    }

    function openRelations(d: DataSource) {
        relationsDS = d;
        relationsJson = d.table_relations || "[]";
        try {
            const parsed = JSON.parse(relationsJson);
            relationsParsed = Array.isArray(parsed) ? parsed : [];
        } catch {
            relationsParsed = [];
        }
        showRelationsModal = true;
    }
    function viewSchema(ds: DataSource) {
        schemaDS = ds;
        showSchemaModal = true;
    }

    function schemaTables(ds: DataSource | null): SchemaTable[] {
        if (!ds?.schema_info) return [];
        const raw = ds.schema_info.trim();
        try {
            if (raw.startsWith("[")) {
                const parsed = JSON.parse(raw);
                return Array.isArray(parsed) ? parsed : [];
            }
        } catch {
            // fallback to text parser
        }
        const tables: SchemaTable[] = [];
        let current: SchemaTable | null = null;
        for (const line of raw.split("\n")) {
            const t = line.trim();
            if (t.startsWith("表:")) {
                current = { name: t.replace(/^表:\s*/, ""), columns: [] };
                tables.push(current);
            } else if (current && t.startsWith("-")) {
                const [name, type] = t.replace(/^[-\s]+/, "").split(":");
                if (name)
                    current.columns.push({
                        name: name.trim(),
                        type: type?.trim(),
                    });
            }
        }
        return tables;
    }

    function schemaRelations(ds: DataSource | null): Relation[] {
        if (!ds?.table_relations) return [];
        try {
            const parsed = JSON.parse(ds.table_relations);
            return Array.isArray(parsed) ? parsed : [];
        } catch {
            return [];
        }
    }
    function addRelation() {
        relationsParsed = [
            ...relationsParsed,
            { from_table: "", from_column: "", to_table: "", to_column: "" },
        ];
    }
    function removeRelation(idx: number) {
        relationsParsed = relationsParsed.filter((_, i) => i !== idx);
    }
    async function saveRelations() {
        if (!relationsDS) return;
        savingRelations = true;
        try {
            relationsJson = JSON.stringify(relationsParsed);
            await api.put(`/admin/datasources/${relationsDS.id}/relations`, {
                table_relations: relationsJson,
            });
            toast.success("表关系已保存");
            showRelationsModal = false;
            await loadData();
        } catch (e: any) {
            toast.error(e.message || "保存失败");
        } finally {
            savingRelations = false;
        }
    }

    let analyzingDS = "";
    async function aiAnalyzeRelationsDS(d: DataSource) {
        if (!d.schema_info) return toast.error("请先分析数据源表结构");
        analyzingDS = d.id;
        try {
            const tbls = schemaTables(d);
            if (tbls.length === 0) return toast.error("未找到表结构");
            const res = await fetch(
                "/api/v1/admin/skills/ai-generate-relations/stream",
                {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                        Authorization: `Bearer ${localStorage.getItem("token") || ""}`,
                    },
                    body: JSON.stringify({
                        data_source_id: d.id,
                        tables: tbls.map((t) => ({
                            name: t.name,
                            columns: t.columns,
                        })),
                    }),
                },
            );
            if (!res.ok) throw new Error(`请求失败 (${res.status})`);
            const reader = res.body?.getReader();
            if (!reader) throw new Error("无法读取流");
            const decoder = new TextDecoder();
            let buffer = "";
            const newRels: Relation[] = [];
            while (true) {
                const { done, value } = await reader.read();
                if (done) break;
                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split("\n");
                buffer = lines.pop() || "";
                for (const line of lines) {
                    if (!line.startsWith("data: ")) continue;
                    const raw = line.slice(6);
                    if (raw === "[DONE]") break;
                    try {
                        const evt = JSON.parse(raw);
                        if (evt.type === "relation" && evt.data) {
                            newRels.push(evt.data);
                            toast.info(
                                `新关系: ${evt.data.from_table} -> ${evt.data.to_table}`,
                                { duration: 3000 },
                            );
                        } else if (evt.type === "done") {
                            if (newRels.length > 0) {
                                relationsDS = d;
                                relationsParsed = newRels;
                                showRelationsModal = true;
                                toast.success(
                                    `AI发现 ${newRels.length} 条关系，请确认保存`,
                                );
                            } else {
                                toast.info("AI 未发现新关系");
                            }
                        } else if (evt.type === "error") {
                            toast.error(evt.message || "AI 分析失败");
                        }
                    } catch {}
                }
            }
        } catch (e: any) {
            toast.error(e.message || "AI 分析失败");
        } finally {
            analyzingDS = "";
        }
    }

    async function aiAnalyzeInRelationsModal() {
        if (!relationsDS) return;
        const d = relationsDS;
        if (!d.schema_info) return toast.error("请先分析数据源表结构");
        analyzingDS = d.id;
        try {
            const tbls = schemaTables(d);
            if (tbls.length === 0) return toast.error("未找到表结构");
            const res = await fetch(
                "/api/v1/admin/skills/ai-generate-relations/stream",
                {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                        Authorization: `Bearer ${localStorage.getItem("token") || ""}`,
                    },
                    body: JSON.stringify({
                        data_source_id: d.id,
                        tables: tbls.map((t) => ({
                            name: t.name,
                            columns: t.columns,
                        })),
                    }),
                },
            );
            if (!res.ok) throw new Error(`请求失败 (${res.status})`);
            const reader = res.body?.getReader();
            if (!reader) throw new Error("无法读取流");
            const decoder = new TextDecoder();
            let buffer = "";
            // 保留已有关系，在此基础上追加
            const existingKeys = new Set(
                relationsParsed.map(
                    (r) =>
                        `${r.from_table}.${r.from_column}-${r.to_table}.${r.to_column}`,
                ),
            );
            let addedCount = 0;
            while (true) {
                const { done, value } = await reader.read();
                if (done) break;
                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split("\n");
                buffer = lines.pop() || "";
                for (const line of lines) {
                    if (!line.startsWith("data: ")) continue;
                    const raw = line.slice(6);
                    if (raw === "[DONE]") break;
                    try {
                        const evt = JSON.parse(raw);
                        if (evt.type === "relation" && evt.data) {
                            const rel = evt.data as Relation;
                            const key = `${rel.from_table}.${rel.from_column}-${rel.to_table}.${rel.to_column}`;
                            if (!existingKeys.has(key)) {
                                existingKeys.add(key);
                                relationsParsed = [...relationsParsed, rel];
                                relationsParsed = relationsParsed;
                                addedCount++;
                            }
                        } else if (evt.type === "done") {
                            toast.success(`AI 新增 ${addedCount} 条关系`);
                        } else if (evt.type === "error") {
                            toast.error(evt.message || "AI 分析失败");
                        }
                    } catch {}
                }
            }
        } catch (e: any) {
            toast.error(e.message || "AI 分析失败");
        } finally {
            analyzingDS = "";
        }
    }
</script>

<svelte:head><title>数据源 - OpenTether</title></svelte:head>

<div class="card">
    <div class="flex items-center justify-between mb-6">
        <div>
            <h2 class="text-xl font-bold text-slate-800">数据源管理</h2>
            <p class="text-sm text-slate-500 mt-1">
                配置数据库、API 等数据源连接
            </p>
        </div>
        <button
            class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 flex items-center gap-1.5"
            on:click={openAdd}><Plus size={16} /> 添加数据源</button
        >
    </div>
    {#if error}<div
            class="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm"
        >
            {error}<button class="ml-2 underline" on:click={loadData}
                >重试</button
            >
        </div>{/if}
    <div class="mb-4 relative">
        <Search
            size={16}
            class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
        /><input
            type="text"
            placeholder="搜索..."
            bind:value={searchQuery}
            class="w-full pl-9 pr-4 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
        />
    </div>
    {#if loading}<div class="text-center py-12 text-slate-400">加载中...</div>
    {:else if filteredItems.length === 0}<div class="text-center py-12">
            <p class="text-slate-500">
                {searchQuery ? "未找到" : "暂无数据源"}
            </p>
        </div>
    {:else}
        <div class="overflow-x-auto">
            <table class="w-full text-sm">
                <thead
                    ><tr class="border-b border-slate-200"
                        ><th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >名称</th
                        ><th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >类型</th
                        ><th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >主机</th
                        ><th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >数据库</th
                        ><th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >状态</th
                        ><th
                            class="text-right py-3 px-4 font-medium text-slate-500"
                            >操作</th
                        ></tr
                    ></thead
                >
                <tbody>
                    {#each filteredItems as d}
                        <tr class="border-b border-slate-100 hover:bg-slate-50">
                            <td class="py-3 px-4 font-medium">{d.name}</td>
                            <td class="py-3 px-4"
                                ><span
                                    class="px-2 py-0.5 bg-slate-100 text-slate-600 rounded-full text-xs"
                                    >{d.source_type}</span
                                ></td
                            >
                            <td
                                class="py-3 px-4 text-slate-500 font-mono text-xs"
                                >{d.host}:{d.port || ""}</td
                            >
                            <td class="py-3 px-4 text-slate-500"
                                >{d.database || "-"}</td
                            >
                            <td class="py-3 px-4"
                                ><button
                                    class="text-xs hover:underline"
                                    on:click={() => handleToggle(d)}
                                    >{d.enabled ? "已启用" : "已停用"}</button
                                ></td
                            >
                            <td class="py-3 px-4 text-right">
                                <div
                                    class="flex items-center justify-end gap-1"
                                >
                                    <button
                                        class="p-1.5 text-slate-400 hover:text-purple-600 hover:bg-purple-50 rounded-md"
                                        title="查看 Schema"
                                        on:click={() => viewSchema(d)}
                                        >📋</button
                                    >
                                    <button
                                        class="p-1.5 text-slate-400 hover:text-primary-600 hover:bg-primary-50 rounded-md"
                                        title="分析表结构"
                                        on:click={() => handleAnalyze(d)}
                                        ><Zap size={14} /></button
                                    >
                                    <button
                                        class="p-1.5 text-slate-400 hover:text-blue-600 hover:bg-blue-50 rounded-md"
                                        title="编辑表关系"
                                        on:click={() => openRelations(d)}
                                        ><Link size={14} /></button
                                    >
                                    <button
                                        class="p-1.5 text-slate-400 hover:text-primary-600 hover:bg-primary-50 rounded-md"
                                        title="编辑"
                                        on:click={() => openEdit(d)}
                                        ><Pencil size={14} /></button
                                    >
                                    <button
                                        class="p-1.5 text-slate-400 hover:text-emerald-600 hover:bg-emerald-50 rounded-md"
                                        title="AI分析表关系"
                                        on:click={() => aiAnalyzeRelationsDS(d)}
                                        >🧠</button
                                    >
                                    <button
                                        class="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-md"
                                        title="删除"
                                        on:click={() => handleDelete(d)}
                                        ><Trash2 size={14} /></button
                                    >
                                </div>
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    {/if}
</div>

<!-- Add/Edit Modal -->
{#if showModal}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        on:click|self={() => (showModal = false)}
        on:keydown={(e) => e.key === "Escape" && (showModal = false)}
    >
        <div
            class="bg-white rounded-2xl shadow-xl w-full max-w-xl max-h-[85vh] overflow-y-auto p-6"
        >
            <h3 class="text-lg font-bold text-slate-800 mb-4">
                {editing ? "编辑" : "添加"}数据源
            </h3>
            <div class="space-y-4">
                <div>
                    <label class="block text-sm font-medium text-slate-700 mb-1"
                        >名称 *</label
                    ><input
                        type="text"
                        bind:value={fName}
                        placeholder="数据源名称"
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm"
                    />
                </div>
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                            >类型</label
                        ><select
                            bind:value={fType}
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm"
                            >{#each types as t}<option value={t}>{t}</option
                                >{/each}</select
                        >
                    </div>
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                            >端口</label
                        ><input
                            type="number"
                            bind:value={fPort}
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm"
                        />
                    </div>
                </div>
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                            >主机</label
                        ><input
                            type="text"
                            bind:value={fHost}
                            placeholder="localhost"
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                            >数据库</label
                        ><input
                            type="text"
                            bind:value={fDatabase}
                            placeholder="mydb"
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm"
                        />
                    </div>
                </div>
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                            >用户名</label
                        ><input
                            type="text"
                            bind:value={fUser}
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                            >密码</label
                        ><input
                            type="password"
                            bind:value={fPassword}
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm"
                        />
                    </div>
                </div>
                <label class="flex items-center gap-2 text-sm"
                    ><input type="checkbox" bind:checked={fEnabled} /> 启用</label
                >
                {#if testResult}<div
                        class="p-3 rounded-lg text-sm bg-slate-50 border border-slate-200"
                    >
                        {testResult}
                    </div>{/if}
            </div>
            <div
                class="flex justify-between mt-6 pt-4 border-t border-slate-100"
            >
                <button
                    class="text-sm text-primary-600 hover:text-primary-800"
                    on:click={handleTest}
                    disabled={testing}
                    >{testing ? "测试中..." : "测试连接"}</button
                >
                <div class="flex gap-2">
                    <button
                        class="px-4 py-2 bg-slate-100 rounded-lg text-sm"
                        on:click={() => (showModal = false)}>取消</button
                    >
                    <button
                        class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm"
                        on:click={handleSave}
                        disabled={saving}
                        >{saving ? "保存中..." : "保存"}</button
                    >
                </div>
            </div>
        </div>
    </div>
{/if}

<!-- Relations Editor Modal -->
{#if showRelationsModal && relationsDS}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        on:click|self={() => (showRelationsModal = false)}
    >
        <div
            class="bg-white rounded-2xl shadow-xl w-full max-w-2xl max-h-[80vh] overflow-y-auto p-6"
        >
            <h3 class="text-lg font-bold text-slate-800 mb-2">
                编辑表关系 - {relationsDS.name}
            </h3>
            <div class="flex items-center gap-3 mb-3">
                <p class="text-sm text-slate-500 flex-1">
                    手动配置或使用 AI 自动识别表之间的关联关系。
                </p>
                <button
                    class="px-3 py-1.5 text-xs font-medium bg-emerald-50 text-emerald-700 rounded-lg hover:bg-emerald-100 transition-colors disabled:opacity-50"
                    disabled={analyzingDS === relationsDS?.id}
                    on:click={() => aiAnalyzeInRelationsModal()}
                >
                    {analyzingDS === relationsDS?.id
                        ? "AI分析中..."
                        : "🧠 AI 自动识别"}
                </button>
            </div>
            <div class="space-y-2 mb-4">
                {#each relationsParsed as rel, i}
                    <div
                        class="flex items-center gap-2 p-2 bg-slate-50 rounded-lg"
                    >
                        <select
                            bind:value={rel.from_table}
                            class="flex-1 px-2 py-1.5 border rounded text-xs"
                            on:change={() => (rel.from_column = "")}
                        >
                            <option value="">选择源表</option>
                            {#if relationsDS}
                                {#each schemaTables(relationsDS) as t}
                                    <option value={t.name}>{t.name}</option>
                                {/each}
                            {/if}
                        </select>
                        <span class="text-slate-400 text-xs">.</span>
                        <select
                            bind:value={rel.from_column}
                            class="w-28 px-2 py-1.5 border rounded text-xs"
                        >
                            <option value="">列</option>
                            {#if relationsDS && rel.from_table}
                                {@const cols =
                                    schemaTables(relationsDS).find(
                                        (t) => t.name === rel.from_table,
                                    )?.columns || []}
                                {#each cols as col}
                                    <option value={col.name}>{col.name}</option>
                                {/each}
                            {/if}
                        </select>
                        <span class="text-slate-400 text-xs font-bold">→</span>
                        <select
                            bind:value={rel.to_table}
                            class="flex-1 px-2 py-1.5 border rounded text-xs"
                            on:change={() => (rel.to_column = "")}
                        >
                            <option value="">选择目标表</option>
                            {#if relationsDS}
                                {#each schemaTables(relationsDS) as t}
                                    <option value={t.name}>{t.name}</option>
                                {/each}
                            {/if}
                        </select>
                        <span class="text-slate-400 text-xs">.</span>
                        <select
                            bind:value={rel.to_column}
                            class="w-28 px-2 py-1.5 border rounded text-xs"
                        >
                            <option value="">列</option>
                            {#if relationsDS && rel.to_table}
                                {@const cols =
                                    schemaTables(relationsDS).find(
                                        (t) => t.name === rel.to_table,
                                    )?.columns || []}
                                {#each cols as col}
                                    <option value={col.name}>{col.name}</option>
                                {/each}
                            {/if}
                        </select>
                        <button
                            class="p-1 text-red-400 hover:text-red-600"
                            on:click={() => removeRelation(i)}
                            ><Trash2 size={14} /></button
                        >
                    </div>
                {/each}
            </div>
            <button
                class="text-sm text-primary-600 hover:text-primary-800 mb-4 flex items-center gap-1"
                on:click={addRelation}><Plus size={14} /> 添加关系</button
            >
            <div class="flex justify-end gap-2 pt-4 border-t border-slate-100">
                <button
                    class="px-4 py-2 bg-slate-100 rounded-lg text-sm"
                    on:click={() => (showRelationsModal = false)}>取消</button
                >
                <button
                    class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm"
                    on:click={saveRelations}
                    disabled={savingRelations}
                    >{savingRelations ? "保存中..." : "保存"}</button
                >
            </div>
        </div>
    </div>
{/if}

<!-- Schema Viewer Modal -->
{#if showSchemaModal && schemaDS}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        on:click|self={() => (showSchemaModal = false)}
    >
        <div
            class="bg-white rounded-2xl shadow-xl w-full max-w-2xl max-h-[80vh] overflow-y-auto p-6"
        >
            <h3 class="text-lg font-bold text-slate-800 mb-2">
                表结构 - {schemaDS.name}
            </h3>
            <p class="text-sm text-slate-500 mb-4">通过分析数据源自动生成</p>
            {#if schemaDS.schema_info && schemaDS.schema_info !== "[]" && schemaDS.schema_info.length > 10}
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-4">
                    <div
                        class="border border-slate-200 rounded-xl p-3 bg-slate-50"
                    >
                        <div class="text-xs font-semibold text-slate-500 mb-2">
                            ER 关系图（简化）
                        </div>
                        <div class="space-y-2 max-h-72 overflow-y-auto">
                            {#each schemaRelations(schemaDS) as rel}
                                <div class="flex items-center gap-2 text-xs">
                                    <span
                                        class="px-2 py-1 rounded bg-white border font-mono text-slate-700"
                                        >{rel.from_table}.{rel.from_column}</span
                                    >
                                    <span class="text-primary-500 font-bold"
                                        >→</span
                                    >
                                    <span
                                        class="px-2 py-1 rounded bg-white border font-mono text-slate-700"
                                        >{rel.to_table}.{rel.to_column}</span
                                    >
                                </div>
                            {:else}
                                <div class="text-xs text-slate-400">
                                    暂无已确认关系，可点击“关系”手动维护。
                                </div>
                            {/each}
                        </div>
                    </div>
                    <div class="border border-slate-200 rounded-xl p-3">
                        <div class="text-xs font-semibold text-slate-500 mb-2">
                            数据表 / 字段
                        </div>
                        <div
                            class="grid grid-cols-1 gap-2 max-h-72 overflow-y-auto"
                        >
                            {#each schemaTables(schemaDS) as table}
                                <details
                                    class="rounded-lg border border-slate-200 bg-white"
                                    open
                                >
                                    <summary
                                        class="cursor-pointer px-3 py-2 text-sm font-medium text-slate-700 bg-slate-50"
                                    >
                                        {table.name}
                                        <span
                                            class="text-xs text-slate-400 ml-1"
                                            >{table.columns?.length || 0} fields</span
                                        >
                                    </summary>
                                    <div
                                        class="px-3 py-2 flex flex-wrap gap-1.5"
                                    >
                                        {#each table.columns || [] as col}
                                            <span
                                                class="text-[11px] px-2 py-1 rounded bg-slate-100 text-slate-600 font-mono"
                                                title={col.type || ""}
                                                >{col.name}</span
                                            >
                                        {/each}
                                    </div>
                                </details>
                            {/each}
                        </div>
                    </div>
                </div>
                <details>
                    <summary class="cursor-pointer text-xs text-slate-500 mb-2"
                        >查看原始 Schema</summary
                    >
                    <pre
                        class="bg-slate-900 text-green-400 p-4 rounded-lg text-xs overflow-x-auto whitespace-pre-wrap font-mono max-h-72">{schemaDS.schema_info}</pre>
                </details>
            {:else}
                <div class="text-center py-8 text-slate-400">
                    <p>暂未分析表结构</p>
                    <button
                        class="mt-2 text-primary-600 text-sm hover:underline"
                        on:click={() => {
                            showSchemaModal = false;
                            if (schemaDS) handleAnalyze(schemaDS);
                        }}>点击分析</button
                    >
                </div>
            {/if}
            <div class="flex justify-end pt-4 border-t border-slate-100 mt-4">
                <button
                    class="px-4 py-2 bg-slate-100 rounded-lg text-sm"
                    on:click={() => (showSchemaModal = false)}>关闭</button
                >
            </div>
        </div>
    </div>
{/if}
