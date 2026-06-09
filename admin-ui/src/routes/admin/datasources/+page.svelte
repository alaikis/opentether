<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { Plus, Pencil, Trash2, Search } from "lucide-svelte";

    interface DataSource {
        id: string;
        name: string;
        source_type: string;
        host: string;
        port: number;
        user: string;
        database: string;
        schema_info: string;
        enabled: boolean;
        created_at: string;
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
            ? items.filter(function (ds: DataSource) {
                  return (
                      ds.name.toLowerCase().indexOf(q) !== -1 ||
                      ds.source_type.toLowerCase().indexOf(q) !== -1 ||
                      (ds.host || "").toLowerCase().indexOf(q) !== -1
                  );
              })
            : items;
    }

    onMount(function () {
        loadData();
    });

    async function loadData() {
        loading = true;
        error = "";
        try {
            let data = await api.get<any>("/admin/datasources");
            items = data.datasources || data || [];
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
        testResult = "";
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
        testResult = "";
        showModal = true;
    }

    async function handleSave() {
        if (!fName.trim()) {
            toast.error("请输入名称");
            return;
        }
        saving = true;
        try {
            let body: Record<string, any> = {
                name: fName.trim(),
                source_type: fType,
                host: fHost,
                port: fPort,
                user: fUser,
                database: fDatabase,
                enabled: fEnabled,
            };
            if (fPassword) body.password = fPassword;
            if (editing) {
                await api.put("/admin/datasources/" + editing.id, body);
                toast.success("已更新");
            } else {
                await api.post("/admin/datasources", body);
                toast.success("已添加");
            }
            showModal = false;
            await loadData();
        } catch (e: any) {
            toast.error(e.message || "保存失败");
        } finally {
            saving = false;
        }
    }

    async function handleTest() {
        testing = true;
        testResult = "";
        try {
            let url = editing
                ? "/admin/datasources/" + editing.id + "/test"
                : "/admin/datasources/test";
            let data = await api.post<any>(url, {
                source_type: fType,
                host: fHost,
                port: fPort,
                user: fUser,
                password: fPassword,
                database: fDatabase,
            });
            testResult = "OK: " + (data.message || "连接成功");
        } catch (e: any) {
            testResult = "ERR: " + (e.message || "连接失败");
        } finally {
            testing = false;
        }
    }

    async function handleToggle(d: DataSource) {
        try {
            await api.put("/admin/datasources/" + d.id, {
                enabled: !d.enabled,
            });
            await loadData();
        } catch (e: any) {
            toast.error(e.message || "操作失败");
        }
    }

    async function handleDelete(d: DataSource) {
        if (!confirm("确定删除 " + d.name + "？")) return;
        try {
            await api.delete("/admin/datasources/" + d.id);
            toast.success("已删除");
            await loadData();
        } catch (e: any) {
            toast.error(e.message || "删除失败");
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
            on:click={openAdd}
        >
            <Plus size={16} /> 添加数据源
        </button>
    </div>

    {#if error}
        <div
            class="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm"
        >
            {error}
            <button class="ml-2 underline" on:click={loadData}>重试</button>
        </div>
    {/if}

    <div class="mb-4 relative">
        <Search
            size={16}
            class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
        />
        <input
            type="text"
            placeholder="搜索..."
            bind:value={searchQuery}
            class="w-full pl-9 pr-4 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
        />
    </div>

    {#if loading}
        <div class="text-center py-12 text-slate-400">加载中...</div>
    {:else if filteredItems.length === 0}
        <div class="text-center py-12">
            <p class="text-slate-500">
                {searchQuery ? "未找到" : "暂无数据源"}
            </p>
        </div>
    {:else}
        <div class="overflow-x-auto">
            <table class="w-full text-sm">
                <thead
                    ><tr class="border-b border-slate-200">
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >名称</th
                        >
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >类型</th
                        >
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >主机</th
                        >
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >数据库</th
                        >
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >状态</th
                        >
                        <th
                            class="text-right py-3 px-4 font-medium text-slate-500"
                            >操作</th
                        >
                    </tr></thead
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
                            <td class="py-3 px-4">
                                <button
                                    class="text-xs hover:underline"
                                    on:click={() => handleToggle(d)}
                                >
                                    {d.enabled ? "已启用" : "已停用"}
                                </button>
                            </td>
                            <td class="py-3 px-4 text-right">
                                <div
                                    class="flex items-center justify-end gap-1"
                                >
                                    <button
                                        class="p-1.5 text-slate-400 hover:text-primary-600 hover:bg-primary-50 rounded-md"
                                        title="编辑"
                                        on:click={() => openEdit(d)}
                                        ><Pencil size={14} /></button
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
                    >
                    <input
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
                        >
                        <select
                            bind:value={fType}
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm"
                        >
                            {#each types as t}<option value={t}>{t}</option
                                >{/each}
                        </select>
                    </div>
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                            >端口</label
                        >
                        <input
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
                        >
                        <input
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
                        >
                        <input
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
                        >
                        <input
                            type="text"
                            bind:value={fUser}
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                            >密码</label
                        >
                        <input
                            type="password"
                            bind:value={fPassword}
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm"
                        />
                    </div>
                </div>
                <label class="flex items-center gap-2 text-sm"
                    ><input type="checkbox" bind:checked={fEnabled} /> 启用</label
                >
                {#if testResult}
                    <div
                        class="p-3 rounded-lg text-sm bg-slate-50 border border-slate-200"
                    >
                        {testResult}
                    </div>
                {/if}
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
