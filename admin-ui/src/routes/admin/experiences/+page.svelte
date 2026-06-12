<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import {
        Brain,
        Trash2,
        CheckCircle,
        XCircle,
        Globe,
        Eye,
    } from "lucide-svelte";

    interface Experience {
        id: string;
        name: string;
        description: string;
        trigger_pattern: string;
        scope: string;
        status: string;
        usage_count: number;
        success_count: number;
        avg_tokens_saved: number;
        created_by: string;
        reviewed_by: string;
        review_note: string;
        created_at: string;
        updated_at: string;
    }

    const tabs = [
        { key: "pending_review", label: "待审核" },
        { key: "active", label: "已激活" },
        { key: "rejected", label: "已拒绝" },
        { key: "disabled", label: "已禁用" },
    ];

    let activeTab = "pending_review";
    let experiences: Experience[] = [];
    let loading = true;
    let error = "";

    // Detail modal
    let showDetail = false;
    let detailExp: Experience | null = null;

    $: filteredExperiences = experiences;

    onMount(() => {
        loadExperiences();
    });

    async function loadExperiences() {
        loading = true;
        error = "";
        try {
            const params: Record<string, string> = {};
            if (activeTab) params.status = activeTab;
            experiences = await api.get<Experience[]>("/admin/experiences", {
                params,
            });
        } catch (e: any) {
            error = e.message || "加载失败";
        } finally {
            loading = false;
        }
    }

    async function approveExperience(id: string) {
        try {
            await api.post(`/admin/experiences/${id}/review`, {
                status: "active",
                note: "审核通过",
            });
            toast.success("经验已激活");
            loadExperiences();
        } catch (e: any) {
            toast.error(e.message || "操作失败");
        }
    }

    async function rejectExperience(id: string) {
        const note = prompt("请输入拒绝原因（可选）：");
        if (note === null) return; // cancelled
        try {
            await api.post(`/admin/experiences/${id}/review`, {
                status: "rejected",
                note: note || "",
            });
            toast.success("经验已拒绝");
            loadExperiences();
        } catch (e: any) {
            toast.error(e.message || "操作失败");
        }
    }

    async function promoteExperience(id: string) {
        if (!confirm("确认将此经验升级为全局经验？")) return;
        try {
            await api.post(`/admin/experiences/${id}/promote`);
            toast.success("已升级为全局经验");
            loadExperiences();
        } catch (e: any) {
            toast.error(e.message || "操作失败");
        }
    }

    async function deleteExperience(id: string) {
        if (!confirm("确认删除此经验？")) return;
        try {
            await api.delete(`/admin/experiences/${id}`);
            toast.success("经验已删除");
            loadExperiences();
        } catch (e: any) {
            toast.error(e.message || "删除失败");
        }
    }

    function showDetailModal(exp: Experience) {
        detailExp = exp;
        showDetail = true;
    }

    function statusBadge(status: string): { label: string; cls: string } {
        switch (status) {
            case "active":
                return { label: "已激活", cls: "bg-green-100 text-green-800" };
            case "pending_review":
                return {
                    label: "待审核",
                    cls: "bg-yellow-100 text-yellow-800",
                };
            case "rejected":
                return { label: "已拒绝", cls: "bg-red-100 text-red-800" };
            case "disabled":
                return { label: "已禁用", cls: "bg-gray-100 text-gray-600" };
            default:
                return { label: status, cls: "bg-gray-100 text-gray-600" };
        }
    }

    function scopeBadge(scope: string): string {
        if (scope === "global") return "全局";
        if (scope.startsWith("user:")) return "个人";
        return scope;
    }

    function parsePatterns(patternStr: string): string[] {
        try {
            return JSON.parse(patternStr);
        } catch {
            return [patternStr];
        }
    }

    function formatDate(dateStr: string): string {
        if (!dateStr) return "-";
        return new Date(dateStr).toLocaleString("zh-CN");
    }
</script>

<div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
        <div>
            <h1 class="text-2xl font-bold">经验管理</h1>
            <p class="text-sm text-gray-500 mt-1">
                管理智能体自动积累的经验，审核后可复用，节省 Token
            </p>
        </div>
    </div>

    <!-- Tabs -->
    <div class="flex gap-1 border-b">
        {#each tabs as tab}
            <button
                class="px-4 py-2 text-sm font-medium transition-colors border-b-2 {activeTab ===
                tab.key
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700'}"
                on:click={() => {
                    activeTab = tab.key;
                    loadExperiences();
                }}
            >
                {tab.label}
            </button>
        {/each}
    </div>

    <!-- Loading -->
    {#if loading}
        <div class="text-center py-12 text-gray-400">
            <div
                class="inline-block w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"
            ></div>
            <p class="mt-2">加载中...</p>
        </div>
    {:else if error}
        <div class="text-center py-12 text-red-500">{error}</div>
    {:else if filteredExperiences.length === 0}
        <div class="text-center py-12 text-gray-400">
            <Brain class="w-12 h-12 mx-auto mb-3 opacity-30" />
            <p>暂无{tabs.find((t) => t.key === activeTab)?.label || ""}经验</p>
            <p class="text-xs mt-1">
                {activeTab === "pending_review"
                    ? "智能体在对话中积累的经验将出现在这里"
                    : ""}
            </p>
        </div>
    {:else}
        <!-- Table -->
        <div class="overflow-x-auto border rounded-lg">
            <table class="w-full text-sm">
                <thead class="bg-gray-50 border-b">
                    <tr>
                        <th
                            class="px-4 py-3 text-left font-medium text-gray-600"
                        >
                            名称
                        </th>
                        <th
                            class="px-4 py-3 text-left font-medium text-gray-600"
                        >
                            范围
                        </th>
                        <th
                            class="px-4 py-3 text-left font-medium text-gray-600"
                        >
                            状态
                        </th>
                        <th
                            class="px-4 py-3 text-left font-medium text-gray-600"
                        >
                            使用/成功
                        </th>
                        <th
                            class="px-4 py-3 text-left font-medium text-gray-600"
                        >
                            创建时间
                        </th>
                        <th
                            class="px-4 py-3 text-right font-medium text-gray-600"
                        >
                            操作
                        </th>
                    </tr>
                </thead>
                <tbody class="divide-y">
                    {#each filteredExperiences as exp}
                        <tr class="hover:bg-gray-50">
                            <td class="px-4 py-3">
                                <div class="font-medium">{exp.name}</div>
                                <div
                                    class="text-xs text-gray-400 truncate max-w-xs"
                                >
                                    {exp.description}
                                </div>
                            </td>
                            <td class="px-4 py-3">
                                <span
                                    class="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full {exp.scope ===
                                    'global'
                                        ? 'bg-blue-50 text-blue-700'
                                        : 'bg-purple-50 text-purple-700'}"
                                >
                                    {#if exp.scope === "global"}
                                        <Globe class="w-3 h-3" />
                                    {:else}
                                        <Brain class="w-3 h-3" />
                                    {/if}
                                    {scopeBadge(exp.scope)}
                                </span>
                            </td>
                            <td class="px-4 py-3">
                                <span
                                    class="inline-block text-xs px-2 py-0.5 rounded-full {statusBadge(
                                        exp.status,
                                    ).cls}"
                                >
                                    {statusBadge(exp.status).label}
                                </span>
                            </td>
                            <td class="px-4 py-3 text-gray-600">
                                {exp.usage_count} / {exp.success_count}
                            </td>
                            <td class="px-4 py-3 text-gray-500 text-xs">
                                {formatDate(exp.created_at)}
                            </td>
                            <td class="px-4 py-3 text-right">
                                <div
                                    class="flex items-center justify-end gap-1"
                                >
                                    <button
                                        title="查看详情"
                                        class="p-1.5 rounded hover:bg-gray-100 text-gray-400 hover:text-gray-600"
                                        on:click={() => showDetailModal(exp)}
                                    >
                                        <Eye class="w-4 h-4" />
                                    </button>

                                    {#if activeTab === "pending_review"}
                                        <button
                                            title="通过"
                                            class="p-1.5 rounded hover:bg-green-50 text-gray-400 hover:text-green-600"
                                            on:click={() =>
                                                approveExperience(exp.id)}
                                        >
                                            <CheckCircle class="w-4 h-4" />
                                        </button>
                                        <button
                                            title="拒绝"
                                            class="p-1.5 rounded hover:bg-red-50 text-gray-400 hover:text-red-600"
                                            on:click={() =>
                                                rejectExperience(exp.id)}
                                        >
                                            <XCircle class="w-4 h-4" />
                                        </button>
                                    {/if}

                                    {#if activeTab === "active" && exp.scope !== "global"}
                                        <button
                                            title="升级为全局"
                                            class="p-1.5 rounded hover:bg-blue-50 text-gray-400 hover:text-blue-600"
                                            on:click={() =>
                                                promoteExperience(exp.id)}
                                        >
                                            <Globe class="w-4 h-4" />
                                        </button>
                                    {/if}

                                    <button
                                        title="删除"
                                        class="p-1.5 rounded hover:bg-red-50 text-gray-400 hover:text-red-600"
                                        on:click={() =>
                                            deleteExperience(exp.id)}
                                    >
                                        <Trash2 class="w-4 h-4" />
                                    </button>
                                </div>
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    {/if}
</div>

<!-- Detail Modal -->
{#if showDetail && detailExp}
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        on:click|self={() => (showDetail = false)}
    >
        <div class="bg-white rounded-xl shadow-xl max-w-lg w-full mx-4 p-6">
            <div class="flex items-center justify-between mb-4">
                <h2 class="text-lg font-bold flex items-center gap-2">
                    <Brain class="w-5 h-5 text-blue-500" />
                    {detailExp.name}
                </h2>
                <button
                    class="p-1 rounded hover:bg-gray-100"
                    on:click={() => (showDetail = false)}
                >
                    <XCircle class="w-5 h-5 text-gray-400" />
                </button>
            </div>

            <div class="space-y-3 text-sm">
                <div>
                    <span class="text-gray-500">状态：</span>
                    <span
                        class="px-2 py-0.5 rounded-full text-xs {statusBadge(
                            detailExp.status,
                        ).cls}"
                    >
                        {statusBadge(detailExp.status).label}
                    </span>
                </div>
                <div>
                    <span class="text-gray-500">范围：</span>
                    {scopeBadge(detailExp.scope)}
                </div>
                <div>
                    <span class="text-gray-500">描述：</span>
                    <span class="text-gray-700"
                        >{detailExp.description || "-"}</span
                    >
                </div>
                <div>
                    <span class="text-gray-500">触发关键词：</span>
                    <div class="flex flex-wrap gap-1 mt-1">
                        {#each parsePatterns(detailExp.trigger_pattern) as pattern}
                            <span
                                class="px-2 py-0.5 bg-blue-50 text-blue-700 rounded text-xs"
                            >
                                {pattern}
                            </span>
                        {/each}
                    </div>
                </div>
                <div class="grid grid-cols-3 gap-3">
                    <div>
                        <span class="text-gray-500 text-xs">使用次数</span>
                        <p class="font-medium">{detailExp.usage_count}</p>
                    </div>
                    <div>
                        <span class="text-gray-500 text-xs">成功次数</span>
                        <p class="font-medium">{detailExp.success_count}</p>
                    </div>
                    <div>
                        <span class="text-gray-500 text-xs">平均省 Token</span>
                        <p class="font-medium">{detailExp.avg_tokens_saved}</p>
                    </div>
                </div>
                {#if detailExp.reviewed_by}
                    <div>
                        <span class="text-gray-500">审核人：</span>
                        {detailExp.reviewed_by}
                    </div>
                {/if}
                {#if detailExp.review_note}
                    <div>
                        <span class="text-gray-500">审核备注：</span>
                        <span class="text-gray-700"
                            >{detailExp.review_note}</span
                        >
                    </div>
                {/if}
                <div class="text-xs text-gray-400">
                    创建时间：{formatDate(detailExp.created_at)}
                </div>
            </div>
        </div>
    </div>
{/if}
