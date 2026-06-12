<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import {
        Shield,
        CheckCircle,
        XCircle,
        Clock,
        Database,
        Eye,
        User,
    } from "lucide-svelte";

    interface SQLAudit {
        id: string;
        user_id: string;
        skill_id: string;
        question: string;
        generated_sql: string;
        data_source_id: string;
        status: string; // pending | approved | rejected | auto_approved | executed
        approved_by: string;
        approved_at: string;
        reject_reason: string;
        row_count: number;
        exec_time: string;
        created_at: string;
    }

    const tabs = [
        { key: "pending", label: "待审批", icon: Clock },
        { key: "approved", label: "已通过", icon: CheckCircle },
        { key: "executed", label: "已执行", icon: Database },
        { key: "rejected", label: "已拒绝", icon: XCircle },
        { key: "", label: "全部", icon: Shield },
    ];

    let activeTab = "pending";
    let audits: SQLAudit[] = [];
    let loading = true;
    let error = "";

    // SQL 详情弹窗
    let showSQL = false;
    let sqlContent = "";
    let sqlQuestion = "";

    // 拒绝理由弹窗
    let showReject = false;
    let rejectAuditID = "";
    let rejectReason = "";

    $: filteredAudits = activeTab
        ? audits.filter((a) => a.status === activeTab)
        : audits;

    onMount(() => {
        loadAudits();
    });

    async function loadAudits() {
        loading = true;
        error = "";
        try {
            let params: Record<string, string> = {};
            if (activeTab) params.status = activeTab;
            audits =
                (await api.get<SQLAudit[]>("/admin/sql-audits", { params })) ||
                [];
        } catch (e: any) {
            error = e.message || "加载 SQL 审计记录失败";
            audits = [];
        } finally {
            loading = false;
        }
    }

    async function approveAudit(id: string) {
        try {
            await api.post(`/admin/sql-audits/${id}/approve`, {});
            toast.success("SQL 已审批通过");
            await loadAudits();
        } catch (e: any) {
            toast.error(e.message || "审批失败");
        }
    }

    async function rejectAudit(id: string) {
        try {
            await api.post(`/admin/sql-audits/${id}/reject`, {
                reason: rejectReason || "管理员拒绝",
            });
            toast.success("SQL 已拒绝");
            showReject = false;
            rejectReason = "";
            await loadAudits();
        } catch (e: any) {
            toast.error(e.message || "拒绝失败");
        }
    }

    function viewSQL(audit: SQLAudit) {
        sqlContent = audit.generated_sql;
        sqlQuestion = audit.question;
        showSQL = true;
    }

    function statusBadge(s: string) {
        switch (s) {
            case "pending":
                return "bg-yellow-100 text-yellow-800";
            case "approved":
                return "bg-green-100 text-green-800";
            case "executed":
                return "bg-blue-100 text-blue-800";
            case "rejected":
                return "bg-red-100 text-red-800";
            default:
                return "bg-gray-100 text-gray-800";
        }
    }

    function statusLabel(s: string) {
        switch (s) {
            case "pending":
                return "待审批";
            case "approved":
            case "auto_approved":
                return "已通过";
            case "executed":
                return "已执行";
            case "rejected":
                return "已拒绝";
            default:
                return s;
        }
    }

    $: if (activeTab) loadAudits();
</script>

<svelte:head><title>SQL 审计 - OpenTether</title></svelte:head>

<div class="mb-6">
    <h2 class="text-2xl font-bold text-slate-800">SQL 审计</h2>
    <p class="text-slate-500 mt-1">
        审查和管理所有由 text2sql 技能生成的 SQL 查询
    </p>
</div>

<!-- Tabs -->
<div class="flex gap-2 mb-6 flex-wrap">
    {#each tabs as t}
        <button
            class="px-4 py-2 text-sm font-medium rounded-lg transition-colors {activeTab === t.key ? 'bg-blue-600 text-white' : 'bg-white text-slate-600 border border-slate-200 hover:bg-slate-50'}"
            on:click={() => (activeTab = t.key)}
        >
            <span class="inline-flex items-center gap-1.5">
                {#if t.icon}
                    <svelte:component this={t.icon} size={16} />
                {/if}
                {t.label}
            </span>
        </button>
    {/each}
</div>

{#if loading}
    <div class="text-center py-12 text-gray-400">加载中...</div>
{:else if error}
    <div class="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm">
        {error}
        <button class="ml-2 underline" on:click={loadAudits}>重试</button>
    </div>
{:else if filteredAudits.length === 0}
    <div class="text-center py-12 text-gray-400 text-sm">暂无 {activeTab ? tabs.find(t => t.key === activeTab)?.label : ''} 审计记录</div>
{:else}
    <div class="overflow-x-auto">
        <table class="w-full text-sm">
            <thead>
                <tr class="border-b border-slate-200 text-left">
                    <th class="py-2 px-3 font-medium text-slate-500">问题</th>
                    <th class="py-2 px-3 font-medium text-slate-500 w-24">状态</th>
                    <th class="py-2 px-3 font-medium text-slate-500 w-24">操作</th>
                </tr>
            </thead>
            <tbody class="divide-y divide-slate-100">
                {#each filteredAudits as a}
                    <tr class="hover:bg-slate-50">
                        <td class="py-3 px-3">
                            <div class="font-medium text-slate-800 truncate max-w-md">{a.question || "(无问题描述)"}</div>
                            <div class="text-xs text-slate-400 mt-0.5">
                                <span class="inline-flex items-center gap-1">
                                    <User size={12} />
                                    {a.user_id?.slice(0, 8)}...
                                </span>
                                <span class="mx-1">·</span>
                                {new Date(a.created_at).toLocaleString("zh-CN")}
                                {#if a.exec_time}
                                    <span class="mx-1">·</span>
                                    {a.exec_time}
                                {/if}
                            </div>
                        </td>
                        <td class="py-3 px-3">
                            <span class="text-xs font-medium px-2 py-0.5 rounded-full {statusBadge(a.status)}">
                                {statusLabel(a.status)}
                            </span>
                            {#if a.row_count}
                                <div class="text-xs text-slate-400 mt-0.5">{a.row_count} 行</div>
                            {/if}
                        </td>
                        <td class="py-3 px-3">
                            <div class="flex gap-1">
                                <button
                                    class="p-1.5 rounded hover:bg-slate-100 text-slate-500"
                                    title="查看 SQL"
                                    on:click={() => viewSQL(a)}
                                >
                                    <Eye size={16} />
                                </button>
                                {#if a.status === "pending"}
                                    <button
                                        class="p-1.5 rounded hover:bg-green-50 text-green-600"
                                        title="审批通过"
                                        on:click={() => approveAudit(a.id)}
                                    >
                                        <CheckCircle size={16} />
                                    </button>
                                    <button
                                        class="p-1.5 rounded hover:bg-red-50 text-red-600"
                                        title="拒绝"
                                        on:click={() => {
                                            rejectAuditID = a.id;
                                            showReject = true;
                                        }}
                                    >
                                        <XCircle size={16} />
                                    </button>
                                {/if}
                            </div>
                        </td>
                    </tr>
                {/each}
            </tbody>
        </table>
    </div>
{/if}

<!-- SQL 查看弹窗 -->
{#if showSQL}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        on:click|self={() => (showSQL = false)}
        on:keydown={(e) => e.key === "Escape" && (showSQL = false)}
        role="dialog"
        tabindex="-1"
    >
        <div class="bg-white rounded-xl shadow-xl w-full max-w-2xl mx-4 max-h-[80vh] overflow-auto">
            <div class="p-6">
                <h3 class="font-semibold text-lg text-slate-800 mb-2">SQL 详情</h3>
                <p class="text-sm text-slate-500 mb-4">{sqlQuestion}</p>
                <pre class="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs overflow-x-auto">{sqlContent}</pre>
                <div class="mt-4 flex justify-end">
                    <button
                        class="px-4 py-2 text-sm bg-slate-100 rounded-lg hover:bg-slate-200"
                        on:click={() => (showSQL = false)}
                    >关闭</button>
                </div>
            </div>
        </div>
    </div>
{/if}

<!-- 拒绝理由弹窗 -->
{#if showReject}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        on:click|self={() => (showReject = false)}
        on:keydown={(e) => e.key === "Escape" && (showReject = false)}
        role="dialog"
        tabindex="-1"
    >
        <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4">
            <div class="p-6">
                <h3 class="font-semibold text-lg text-slate-800 mb-4">拒绝理由</h3>
                <textarea
                    bind:value={rejectReason}
                    rows={3}
                    placeholder="请输入拒绝原因..."
                    class="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-blue-200 resize-none"
                ></textarea>
                <div class="mt-4 flex justify-end gap-2">
                    <button
                        class="px-4 py-2 text-sm bg-slate-100 rounded-lg hover:bg-slate-200"
                        on:click={() => (showReject = false)}
                    >取消</button>
                    <button
                        class="px-4 py-2 text-sm bg-red-600 text-white rounded-lg hover:bg-red-700"
                        on:click={() => rejectAudit(rejectAuditID)}
                    >确认拒绝</button>
                </div>
            </div>
        </div>
    </div>
{/if}
