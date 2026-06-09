<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { FileText, Activity, Download } from "lucide-svelte";

    interface AuditLog {
        id: string;
        timestamp: string;
        user_name: string;
        action: string;
        resource_type: string;
        details: string;
        ip_address: string;
    }

    interface RequestLog {
        id: string;
        timestamp: string;
        method: string;
        path: string;
        status: number;
        latency_ms: number;
        user_name: string;
    }

    let activeTab: "audit" | "request" = "audit";
    let auditLogs: AuditLog[] = [];
    let requestLogs: RequestLog[] = [];
    let loading = true;
    let error = "";

    onMount(async () => {
        await loadLogs();
    });

    async function loadLogs() {
        loading = true;
        error = "";
        try {
            if (activeTab === "audit") {
                const data = await api.get<AuditLog[]>("/admin/logs/audit", {
                    params: { limit: "50" },
                });
                auditLogs = Array.isArray(data) ? data : [];
            } else {
                const data = await api.get<RequestLog[]>(
                    "/admin/logs/request",
                    { params: { limit: "50" } },
                );
                requestLogs = Array.isArray(data) ? data : [];
            }
        } catch (e: any) {
            error = e.message || "加载日志失败";
        } finally {
            loading = false;
        }
    }

    function switchTab(tab: "audit" | "request") {
        activeTab = tab;
        loadLogs();
    }

    async function handleExport() {
        try {
            const data = await api.get<{ url: string }>("/admin/logs/export");
            if (data.url) {
                window.open(data.url, "_blank");
            }
            toast.success("导出成功");
        } catch (e: any) {
            toast.error(e.message || "导出失败");
        }
    }

    function formatTime(t: string) {
        return new Date(t).toLocaleString("zh-CN");
    }

    function getActionColor(action: string) {
        const createActions = ["create", "login", "register"];
        const updateActions = ["update", "edit", "modify"];
        const deleteActions = ["delete", "remove"];

        if (createActions.some((a) => action.toLowerCase().includes(a)))
            return "bg-blue-50 text-blue-700";
        if (updateActions.some((a) => action.toLowerCase().includes(a)))
            return "bg-amber-50 text-amber-700";
        if (deleteActions.some((a) => action.toLowerCase().includes(a)))
            return "bg-red-50 text-red-700";
        return "bg-slate-100 text-slate-600";
    }

    function getMethodColor(method: string) {
        const colors: Record<string, string> = {
            GET: "bg-emerald-50 text-emerald-700",
            POST: "bg-blue-50 text-blue-700",
            PUT: "bg-amber-50 text-amber-700",
            DELETE: "bg-red-50 text-red-700",
        };
        return colors[method] || "bg-slate-100 text-slate-600";
    }

    function getStatusColor(status: number) {
        if (status < 300) return "text-emerald-600";
        if (status < 400) return "text-amber-600";
        return "text-red-600";
    }
</script>

<svelte:head><title>系统日志 - OpenTether</title></svelte:head>

<div class="card">
    <div class="flex items-center justify-between mb-6">
        <div>
            <h2 class="text-xl font-bold text-slate-800 mb-1">系统日志</h2>
            <p class="text-sm text-slate-500">审计日志和请求日志</p>
        </div>
        <button
            class="px-4 py-2 bg-slate-100 text-slate-700 rounded-lg text-sm font-medium hover:bg-slate-200 transition-colors flex items-center gap-1.5"
            on:click={handleExport}
        >
            <Download size={16} />
            导出
        </button>
    </div>

    <!-- Tabs -->
    <div class="flex gap-1 mb-4 bg-slate-100 rounded-lg p-1 w-fit">
        <button
            class="px-4 py-1.5 rounded-md text-sm font-medium transition-colors {activeTab ===
            'audit'
                ? 'bg-white text-slate-800 shadow-sm'
                : 'text-slate-500 hover:text-slate-700'}"
            on:click={() => switchTab("audit")}
        >
            <Activity size={14} class="inline mr-1.5" />
            审计日志
        </button>
        <button
            class="px-4 py-1.5 rounded-md text-sm font-medium transition-colors {activeTab ===
            'request'
                ? 'bg-white text-slate-800 shadow-sm'
                : 'text-slate-500 hover:text-slate-700'}"
            on:click={() => switchTab("request")}
        >
            <FileText size={14} class="inline mr-1.5" />
            请求日志
        </button>
    </div>

    {#if error}
        <div
            class="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm"
        >
            {error}
            <button class="ml-2 underline" on:click={loadLogs}>重试</button>
        </div>
    {/if}

    {#if loading}
        <div class="text-center py-12 text-slate-400">加载中...</div>
    {:else if activeTab === "audit"}
        {#if auditLogs.length === 0}
            <div class="text-center py-12 space-y-3">
                <div class="text-4xl">📋</div>
                <p class="text-slate-500">暂无审计日志</p>
            </div>
        {:else}
            <div class="overflow-x-auto">
                <table class="w-full text-sm">
                    <thead>
                        <tr class="border-b border-slate-200">
                            <th
                                class="text-left py-3 px-4 font-medium text-slate-500"
                                >时间</th
                            >
                            <th
                                class="text-left py-3 px-4 font-medium text-slate-500"
                                >用户</th
                            >
                            <th
                                class="text-left py-3 px-4 font-medium text-slate-500"
                                >操作</th
                            >
                            <th
                                class="text-left py-3 px-4 font-medium text-slate-500"
                                >资源</th
                            >
                            <th
                                class="text-left py-3 px-4 font-medium text-slate-500"
                                >IP</th
                            >
                        </tr>
                    </thead>
                    <tbody>
                        {#each auditLogs as log}
                            <tr
                                class="border-b border-slate-100 hover:bg-slate-50"
                            >
                                <td class="py-3 px-4 text-slate-500 text-xs">
                                    {formatTime(log.timestamp)}
                                </td>
                                <td class="py-3 px-4 font-medium"
                                    >{log.user_name || "-"}</td
                                >
                                <td class="py-3 px-4">
                                    <span
                                        class="px-2 py-0.5 rounded-full text-xs {getActionColor(
                                            log.action,
                                        )}"
                                    >
                                        {log.action}
                                    </span>
                                </td>
                                <td class="py-3 px-4 text-slate-500 text-xs">
                                    {log.resource_type || "-"}
                                </td>
                                <td
                                    class="py-3 px-4 text-slate-400 text-xs font-mono"
                                >
                                    {log.ip_address || "-"}
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    {:else}
        {#if requestLogs.length === 0}
            <div class="text-center py-12 space-y-3">
                <div class="text-4xl">📡</div>
                <p class="text-slate-500">暂无请求日志</p>
            </div>
        {:else}
            <div class="overflow-x-auto">
                <table class="w-full text-sm">
                    <thead>
                        <tr class="border-b border-slate-200">
                            <th
                                class="text-left py-3 px-4 font-medium text-slate-500"
                                >时间</th
                            >
                            <th
                                class="text-left py-3 px-4 font-medium text-slate-500"
                                >方法</th
                            >
                            <th
                                class="text-left py-3 px-4 font-medium text-slate-500"
                                >路径</th
                            >
                            <th
                                class="text-left py-3 px-4 font-medium text-slate-500"
                                >状态</th
                            >
                            <th
                                class="text-left py-3 px-4 font-medium text-slate-500"
                                >延迟</th
                            >
                            <th
                                class="text-left py-3 px-4 font-medium text-slate-500"
                                >用户</th
                            >
                        </tr>
                    </thead>
                    <tbody>
                        {#each requestLogs as log}
                            <tr
                                class="border-b border-slate-100 hover:bg-slate-50"
                            >
                                <td class="py-3 px-4 text-slate-500 text-xs">
                                    {formatTime(log.timestamp)}
                                </td>
                                <td class="py-3 px-4">
                                    <span
                                        class="px-2 py-0.5 rounded text-xs font-mono {getMethodColor(
                                            log.method,
                                        )}"
                                    >
                                        {log.method}
                                    </span>
                                </td>
                                <td
                                    class="py-3 px-4 text-slate-600 font-mono text-xs max-w-[300px] truncate"
                                >
                                    {log.path}
                                </td>
                                <td class="py-3 px-4">
                                    <span
                                        class="text-xs font-mono font-medium {getStatusColor(
                                            log.status,
                                        )}"
                                    >
                                        {log.status}
                                    </span>
                                </td>
                                <td class="py-3 px-4 text-slate-500 text-xs">
                                    {log.latency_ms != null
                                        ? `${log.latency_ms}ms`
                                        : "-"}
                                </td>
                                <td class="py-3 px-4 text-slate-500 text-xs">
                                    {log.user_name || "-"}
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    {/if}
</div>
