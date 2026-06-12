<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { Plus, Play, Square, RefreshCw, Trash2, Settings, Wrench, Server } from "lucide-svelte";

    interface MCPConfig {
        id: string;
        name: string;
        transport: string;
        command?: string;
        args?: string;
        env?: string;
        url?: string;
        enabled: boolean;
        created_at: string;
    }

    interface MCPTool {
        name: string;
        description: string;
    }

    let configs: MCPConfig[] = [];
    let loading = true;
    let showModal = false;
    let editing: MCPConfig | null = null;
    let fName = "";
    let fTransport = "stdio";
    let fCommand = "";
    let fArgs = "";
    let fEnv = "";
    let fUrl = "";
    let fEnabled = true;
    let saving = false;

    let serverStatuses: Record<string, string> = {};
    let toolsMap: Record<string, MCPTool[]> = {};

    onMount(() => loadConfigs());

    async function loadConfigs() {
        loading = true;
        try {
            const data = await api.get<any>("/admin/mcp/configs");
            configs = data.configs || data || [];
        } catch (e: any) {
            toast.error(e.message || "加载失败");
        } finally {
            loading = false;
        }
    }

    function openAdd() {
        editing = null;
        fName = "";
        fTransport = "stdio";
        fCommand = "";
        fArgs = "";
        fEnv = "";
        fUrl = "";
        fEnabled = true;
        showModal = true;
    }

    function openEdit(c: MCPConfig) {
        editing = c;
        fName = c.name;
        fTransport = c.transport || "stdio";
        fCommand = c.command || "";
        fArgs = c.args || "";
        fEnv = c.env || "";
        fUrl = c.url || "";
        fEnabled = c.enabled;
        showModal = true;
    }

    async function handleSave() {
        if (!fName) return toast.error("请输入名称");
        saving = true;
        try {
            const body: any = {
                name: fName,
                transport: fTransport,
                command: fCommand,
                args: fArgs,
                env: fEnv,
                url: fUrl,
                enabled: fEnabled,
            };
            if (editing) {
                await api.put(`/admin/mcp/configs/${editing.id}`, body);
                toast.success("已更新");
            } else {
                await api.post("/admin/mcp/configs", body);
                toast.success("已创建");
            }
            showModal = false;
            await loadConfigs();
        } catch (e: any) {
            toast.error(e.message || "保存失败");
        } finally {
            saving = false;
        }
    }

    async function handleDelete(c: MCPConfig) {
        if (!confirm(`确定删除 "${c.name}" 吗？`)) return;
        try {
            await api.delete(`/admin/mcp/configs/${c.id}`);
            toast.success("已删除");
            await loadConfigs();
        } catch (e: any) {
            toast.error(e.message || "删除失败");
        }
    }

    async function toggleServer(c: MCPConfig) {
        try {
            const status = serverStatuses[c.id];
            if (status === "running") {
                await api.post(`/admin/mcp/configs/${c.id}/stop`);
                serverStatuses[c.id] = "stopped";
                toast.success("已停止");
            } else {
                await api.post(`/admin/mcp/configs/${c.id}/start`);
                serverStatuses[c.id] = "running";
                toast.success("已启动");
            }
        } catch (e: any) {
            toast.error(e.message || "操作失败");
        }
    }

    async function checkStatus(c: MCPConfig) {
        try {
            const data = await api.get<any>(`/admin/mcp/configs/${c.id}/status`);
            serverStatuses[c.id] = data.status || "unknown";
            serverStatuses = serverStatuses;
        } catch {
            serverStatuses[c.id] = "error";
        }
    }

    async function loadTools(c: MCPConfig) {
        try {
            const data = await api.get<any>(`/admin/mcp/configs/${c.id}/tools`);
            toolsMap[c.id] = data.tools || data || [];
            toolsMap = toolsMap;
        } catch {
            toolsMap[c.id] = [];
        }
    }

    async function refreshAll() {
        await loadConfigs();
        for (const c of configs) {
            if (c.enabled) {
                await checkStatus(c);
            }
        }
    }
</script>

<svelte:head><title>MCP 服务 - OpenTether</title></svelte:head>

<div class="card">
    <div class="flex items-center justify-between mb-6">
        <div>
            <h2 class="text-xl font-bold text-slate-800">MCP 服务管理</h2>
            <p class="text-sm text-slate-500 mt-1">配置和管理 Model Context Protocol 服务器，扩展 Agent 能力</p>
        </div>
        <div class="flex items-center gap-2">
            <button class="px-3 py-2 rounded-lg bg-slate-100 text-sm flex items-center gap-1" on:click={refreshAll}>
                <RefreshCw size={15} /> 刷新
            </button>
            <button class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium flex items-center gap-1.5" on:click={openAdd}>
                <Plus size={16} /> 添加服务
            </button>
        </div>
    </div>

    {#if loading}
        <div class="text-center py-12 text-slate-400">加载中...</div>
    {:else if configs.length === 0}
        <div class="text-center py-12 space-y-3">
            <div class="text-4xl">🔌</div>
            <p class="text-slate-500">还没有配置 MCP 服务</p>
            <p class="text-sm text-slate-400">MCP 服务可以扩展 Agent 的文件系统、数据库、API 等外部能力</p>
        </div>
    {:else}
        <div class="space-y-3">
            {#each configs as c}
                <div class="p-4 rounded-xl border border-slate-200 hover:border-slate-300 transition-colors">
                    <div class="flex items-start justify-between mb-3">
                        <div class="flex items-center gap-3">
                            <div class="w-10 h-10 rounded-lg {c.enabled ? 'bg-emerald-50' : 'bg-slate-100'} flex items-center justify-center">
                                <Server size={20} class={c.enabled ? 'text-emerald-600' : 'text-slate-400'} />
                            </div>
                            <div>
                                <div class="font-semibold text-slate-800">{c.name}</div>
                                <div class="text-xs text-slate-400">传输方式: {c.transport} {c.url ? ` | ${c.url}` : ""} {c.command ? ` | ${c.command}` : ""}</div>
                            </div>
                        </div>
                        <div class="flex items-center gap-2">
                            <span class="px-2 py-0.5 rounded-full text-[11px] font-medium {c.enabled ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-500'}">
                                {c.enabled ? "启用" : "禁用"}
                            </span>
                            {#if serverStatuses[c.id]}
                                <span class="px-2 py-0.5 rounded-full text-[11px] font-medium {serverStatuses[c.id] === 'running' ? 'bg-blue-50 text-blue-700' : 'bg-slate-100 text-slate-500'}">
                                    {serverStatuses[c.id]}
                                </span>
                            {/if}
                        </div>
                    </div>

                    <div class="flex items-center gap-2">
                        {#if c.enabled}
                            <button class="px-3 py-1.5 text-xs rounded-lg {serverStatuses[c.id] === 'running' ? 'bg-red-50 text-red-600 hover:bg-red-100' : 'bg-emerald-50 text-emerald-600 hover:bg-emerald-100'} transition-colors" on:click={() => toggleServer(c)}>
                                {#if serverStatuses[c.id] === "running"}
                                    <Square size={12} class="inline mr-1" />停止
                                {:else}
                                    <Play size={12} class="inline mr-1" />启动
                                {/if}
                            </button>
                            <button class="px-3 py-1.5 text-xs rounded-lg bg-slate-50 text-slate-600 hover:bg-slate-100 transition-colors" on:click={() => loadTools(c)}>
                                <Wrench size={12} class="inline mr-1" />工具列表
                            </button>
                        {/if}
                        <button class="px-3 py-1.5 text-xs rounded-lg bg-slate-50 text-slate-600 hover:bg-slate-100 transition-colors" on:click={() => openEdit(c)}>
                            <Settings size={12} class="inline mr-1" />编辑
                        </button>
                        <button class="px-3 py-1.5 text-xs rounded-lg bg-slate-50 text-red-500 hover:bg-red-50 transition-colors" on:click={() => handleDelete(c)}>
                            <Trash2 size={12} class="inline mr-1" />删除
                        </button>
                    </div>

                    {#if toolsMap[c.id] && toolsMap[c.id].length > 0}
                        <div class="mt-3 pt-3 border-t border-slate-100">
                            <div class="text-xs font-medium text-slate-500 mb-1.5">可用工具 ({toolsMap[c.id].length}):</div>
                            <div class="flex flex-wrap gap-1.5">
                                {#each toolsMap[c.id] as tool}
                                    <span class="px-2 py-0.5 rounded text-[11px] bg-slate-100 text-slate-600" title={tool.description}>
                                        {tool.name}
                                    </span>
                                {/each}
                            </div>
                        </div>
                    {/if}
                </div>
            {/each}
        </div>
    {/if}
</div>

{#if showModal}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40" on:click|self={() => (showModal = false)}>
        <div class="bg-white rounded-2xl shadow-xl w-full max-w-lg max-h-[85vh] overflow-y-auto p-6">
            <h3 class="text-lg font-bold text-slate-800 mb-4">{editing ? "编辑" : "添加"} MCP 服务</h3>
            <div class="space-y-4">
                <div>
                    <label class="block text-sm font-medium text-slate-700 mb-1">名称 *</label>
                    <input type="text" bind:value={fName} class="w-full px-3 py-2 border rounded-lg text-sm" placeholder="例如: filesystem-server" />
                </div>
                <div>
                    <label class="block text-sm font-medium text-slate-700 mb-1">传输方式</label>
                    <select bind:value={fTransport} class="w-full px-3 py-2 border rounded-lg text-sm">
                        <option value="stdio">stdio (命令行)</option>
                        <option value="sse">SSE (HTTP 流)</option>
                    </select>
                </div>
                {#if fTransport === "stdio"}
                    <div>
                        <label class="block text-sm font-medium text-slate-700 mb-1">命令</label>
                        <input type="text" bind:value={fCommand} class="w-full px-3 py-2 border rounded-lg text-sm" placeholder="npx @modelcontextprotocol/server-filesystem" />
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-slate-700 mb-1">参数</label>
                        <input type="text" bind:value={fArgs} class="w-full px-3 py-2 border rounded-lg text-sm" placeholder="/path/to/allowed" />
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-slate-700 mb-1">环境变量 (KEY=VAL 每行一个)</label>
                        <textarea bind:value={fEnv} class="w-full px-3 py-2 border rounded-lg text-sm" rows={2} placeholder="API_KEY=xxx" />
                    </div>
                {:else}
                    <div>
                        <label class="block text-sm font-medium text-slate-700 mb-1">URL</label>
                        <input type="text" bind:value={fUrl} class="w-full px-3 py-2 border rounded-lg text-sm" placeholder="http://localhost:8000/sse" />
                    </div>
                {/if}
                <label class="flex items-center gap-2 text-sm">
                    <input type="checkbox" bind:checked={fEnabled} /> 启用
                </label>
            </div>
            <div class="flex justify-end gap-2 mt-6 pt-4 border-t">
                <button class="px-4 py-2 bg-slate-100 rounded-lg text-sm" on:click={() => (showModal = false)}>取消</button>
                <button class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm" on:click={handleSave} disabled={saving}>
                    {saving ? "保存中..." : "保存"}
                </button>
            </div>
        </div>
    </div>
{/if}
