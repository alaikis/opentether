<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { Plus, Pencil, Trash2, Play, Search } from "lucide-svelte";

    interface ScheduledTask {
        id: string;
        name: string;
        description: string;
        cron_expression: string;
        executor_type: string;
        enabled: boolean;
        status: string;
        last_run_at: string | null;
        next_run_at: string | null;
        created_at: string;
    }

    let tasks: ScheduledTask[] = [];
    let loading = true;
    let error = "";
    let searchQuery = "";

    let showModal = false;
    let editingTask: ScheduledTask | null = null;
    let saving = false;

    let formName = "";
    let formDescription = "";
    let formCron = "";
    let formExecutorType = "script";
    let formScriptContent = "";
    let formEnabled = true;

    const executorTypes = [
        { value: "script", label: "Shell 脚本" },
        { value: "python", label: "Python 脚本" },
        { value: "api", label: "API 调用" },
    ];

    $: filteredTasks = tasks.filter(
        (t) =>
            !searchQuery ||
            t.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
            (t.description || "")
                .toLowerCase()
                .includes(searchQuery.toLowerCase()),
    );

    onMount(async () => {
        await loadTasks();
    });

    async function loadTasks() {
        loading = true;
        error = "";
        try {
            const data = await api.get<ScheduledTask[]>("/admin/tasks");
            tasks = Array.isArray(data) ? data : [];
        } catch (e: any) {
            error = e.message || "加载任务列表失败";
            tasks = [];
        } finally {
            loading = false;
        }
    }

    function openAddModal() {
        editingTask = null;
        formName = "";
        formDescription = "";
        formCron = "";
        formExecutorType = "script";
        formScriptContent = "";
        formEnabled = true;
        showModal = true;
    }

    function openEditModal(t: ScheduledTask) {
        editingTask = t;
        formName = t.name;
        formDescription = t.description || "";
        formCron = t.cron_expression || "";
        formExecutorType = t.executor_type || "script";
        formScriptContent = "";
        formEnabled = t.enabled;
        showModal = true;
    }

    function closeModal() {
        showModal = false;
        editingTask = null;
    }

    async function handleSave() {
        if (!formName || !formCron) {
            toast.error("请填写任务名称和 Cron 表达式");
            return;
        }
        saving = true;
        try {
            const body: Record<string, any> = {
                name: formName,
                description: formDescription,
                cron_expression: formCron,
                executor_type: formExecutorType,
                enabled: formEnabled,
            };
            if (formScriptContent) {
                body.script_content = formScriptContent;
            }

            if (editingTask) {
                await api.put(`/admin/tasks/${editingTask.id}`, body);
                toast.success("任务已更新");
            } else {
                await api.post("/admin/tasks", body);
                toast.success("任务已创建");
            }
            closeModal();
            await loadTasks();
        } catch (e: any) {
            toast.error(e.message || "保存失败");
        } finally {
            saving = false;
        }
    }

    async function handleRun(t: ScheduledTask) {
        if (!confirm(`确定立即执行任务 "${t.name}" 吗？`)) return;
        try {
            await api.post(`/admin/tasks/${t.id}/run`);
            toast.success("任务已触发执行");
            await loadTasks();
        } catch (e: any) {
            toast.error(e.message || "执行失败");
        }
    }

    async function handleDelete(t: ScheduledTask) {
        if (!confirm(`确定删除任务 "${t.name}" 吗？`)) return;
        try {
            await api.delete(`/admin/tasks/${t.id}`);
            toast.success("任务已删除");
            await loadTasks();
        } catch (e: any) {
            toast.error(e.message || "删除失败");
        }
    }

    function formatTime(t: string | null) {
        if (!t) return "-";
        return new Date(t).toLocaleString("zh-CN", {
            month: "2-digit",
            day: "2-digit",
            hour: "2-digit",
            minute: "2-digit",
        });
    }

    function getStatusColor(status: string) {
        switch (status) {
            case "running":
                return "text-blue-600";
            case "paused":
                return "text-amber-600";
            default:
                return "text-slate-400";
        }
    }

    function getStatusLabel(status: string) {
        const map: Record<string, string> = {
            idle: "待机",
            running: "运行中",
            paused: "已暂停",
        };
        return map[status] || status;
    }
</script>

<svelte:head><title>定时任务 - OpenTether</title></svelte:head>

<div class="card">
    <div class="flex items-center justify-between mb-6">
        <div>
            <h2 class="text-xl font-bold text-slate-800">定时任务</h2>
            <p class="text-sm text-slate-500 mt-1">配置和管理定时执行的任务</p>
        </div>
        <button
            class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 transition-colors flex items-center gap-1.5"
            on:click={openAddModal}
        >
            <Plus size={16} />
            创建任务
        </button>
    </div>

    {#if error}
        <div
            class="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm"
        >
            {error}
            <button class="ml-2 underline" on:click={loadTasks}>重试</button>
        </div>
    {/if}

    <div class="mb-4 relative">
        <Search
            size={16}
            class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
        />
        <input
            type="text"
            placeholder="搜索任务名称..."
            bind:value={searchQuery}
            class="w-full pl-9 pr-4 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
        />
    </div>

    {#if loading}
        <div class="text-center py-12 text-slate-400">加载中...</div>
    {:else if filteredTasks.length === 0}
        <div class="text-center py-12 space-y-3">
            <div class="text-4xl">📅</div>
            <p class="text-slate-500">
                {searchQuery ? "未找到匹配的任务" : "暂无定时任务"}
            </p>
            {#if !searchQuery}
                <p class="text-sm text-slate-400">点击"创建任务"开始</p>
            {/if}
        </div>
    {:else}
        <div class="overflow-x-auto">
            <table class="w-full text-sm">
                <thead>
                    <tr class="border-b border-slate-200">
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >任务名</th
                        >
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >Cron</th
                        >
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >状态</th
                        >
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >上次执行</th
                        >
                        <th
                            class="text-right py-3 px-4 font-medium text-slate-500"
                            >操作</th
                        >
                    </tr>
                </thead>
                <tbody>
                    {#each filteredTasks as t}
                        <tr class="border-b border-slate-100 hover:bg-slate-50">
                            <td class="py-3 px-4 font-medium">{t.name}</td>
                            <td class="py-3 px-4">
                                <code
                                    class="text-xs bg-slate-100 px-1.5 py-0.5 rounded text-slate-600 font-mono"
                                >
                                    {t.cron_expression || "-"}
                                </code>
                            </td>
                            <td class="py-3 px-4">
                                <span
                                    class="flex items-center gap-1.5 text-xs {getStatusColor(
                                        t.status,
                                    )}"
                                >
                                    <span
                                        class="w-1.5 h-1.5 rounded-full"
                                        class:bg-blue-500={t.status ===
                                            "running"}
                                        class:bg-amber-500={t.status ===
                                            "paused"}
                                        class:bg-slate-300={t.status === "idle"}
                                    />
                                    {getStatusLabel(t.status)}
                                </span>
                            </td>
                            <td class="py-3 px-4 text-slate-500 text-xs">
                                {formatTime(t.last_run_at)}
                            </td>
                            <td class="py-3 px-4 text-right">
                                <div
                                    class="flex items-center justify-end gap-1"
                                >
                                    <button
                                        class="p-1.5 text-slate-400 hover:text-emerald-600 hover:bg-emerald-50 rounded-md transition-colors"
                                        title="立即执行"
                                        on:click={() => handleRun(t)}
                                    >
                                        <Play size={14} />
                                    </button>
                                    <button
                                        class="p-1.5 text-slate-400 hover:text-primary-600 hover:bg-primary-50 rounded-md transition-colors"
                                        title="编辑"
                                        on:click={() => openEditModal(t)}
                                    >
                                        <Pencil size={14} />
                                    </button>
                                    <button
                                        class="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-md transition-colors"
                                        title="删除"
                                        on:click={() => handleDelete(t)}
                                    >
                                        <Trash2 size={14} />
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
                {editingTask ? "编辑任务" : "创建任务"}
            </h3>

            <div class="space-y-4">
                <div>
                    <label
                        class="block text-sm font-medium text-slate-700 mb-1"
                    >
                        任务名 <span class="text-red-500">*</span>
                    </label>
                    <input
                        type="text"
                        bind:value={formName}
                        placeholder="任务名称"
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                    />
                </div>

                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                        >
                            Cron 表达式 <span class="text-red-500">*</span>
                        </label>
                        <input
                            type="text"
                            bind:value={formCron}
                            placeholder="0 */6 * * *"
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200 font-mono"
                        />
                        <p class="text-xs text-slate-400 mt-1">
                            分 时 日 月 周
                        </p>
                    </div>
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                        >
                            执行器类型
                        </label>
                        <select
                            bind:value={formExecutorType}
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                        >
                            {#each executorTypes as opt}
                                <option value={opt.value}>{opt.label}</option>
                            {/each}
                        </select>
                    </div>
                </div>

                <div>
                    <label
                        class="block text-sm font-medium text-slate-700 mb-1"
                    >
                        描述
                    </label>
                    <textarea
                        bind:value={formDescription}
                        placeholder="任务描述"
                        rows={2}
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200 resize-none"
                    />
                </div>

                <div>
                    <label
                        class="block text-sm font-medium text-slate-700 mb-1"
                    >
                        脚本内容
                    </label>
                    <textarea
                        bind:value={formScriptContent}
                        placeholder="#!/bin/bash\necho 'hello'"
                        rows={6}
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200 resize-none font-mono text-xs"
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
                        启用任务
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
