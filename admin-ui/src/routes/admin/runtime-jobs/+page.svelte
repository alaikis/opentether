<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { RefreshCcw, Play, XCircle, Eye, Clock } from "lucide-svelte";

    interface RuntimeJob {
        id: string;
        conversation_id: string;
        skill_id: string;
        job_type: string;
        status: string;
        input: string;
        output: string;
        error: string;
        current_step: number;
        max_steps: number;
        recoverable: boolean;
        lease_owner: string;
        lease_expires_at: string;
        started_at: string;
        finished_at?: string;
        updated_at: string;
    }

    interface RuntimeCheckpoint {
        id: string;
        job_id: string;
        step: number;
        type: string;
        state: string;
        idempotency_key: string;
        created_at: string;
    }

    let jobs: RuntimeJob[] = [];
    let checkpoints: RuntimeCheckpoint[] = [];
    let selectedJob: RuntimeJob | null = null;
    let loading = true;
    let status = "";

    const statuses = ["", "running", "paused", "failed", "recovering", "succeeded", "cancelled"];

    onMount(loadJobs);

    async function loadJobs() {
        loading = true;
        try {
            const qs = status ? `?status=${status}` : "";
            const data = await api.get<{ jobs: RuntimeJob[] }>(`/user/runtime/jobs${qs}`);
            jobs = data.jobs || [];
        } catch (e: any) {
            toast.error(e.message || "加载运行任务失败");
        } finally {
            loading = false;
        }
    }

    async function viewJob(job: RuntimeJob) {
        try {
            const data = await api.get<{ job: RuntimeJob; checkpoints: RuntimeCheckpoint[] }>(`/user/runtime/jobs/${job.id}`);
            selectedJob = data.job;
            checkpoints = data.checkpoints || [];
        } catch (e: any) {
            toast.error(e.message || "加载任务详情失败");
        }
    }

    async function retryJob(job: RuntimeJob) {
        try {
            await api.post(`/user/runtime/jobs/${job.id}/retry`, {});
            toast.success("任务已提交恢复/重试");
            await loadJobs();
            await viewJob(job);
        } catch (e: any) {
            toast.error(e.message || "恢复失败");
        }
    }

    async function cancelJob(job: RuntimeJob) {
        if (!confirm("确定取消该任务？")) return;
        try {
            await api.post(`/user/runtime/jobs/${job.id}/cancel`, {});
            toast.success("任务已取消");
            await loadJobs();
            if (selectedJob?.id === job.id) await viewJob(job);
        } catch (e: any) {
            toast.error(e.message || "取消失败");
        }
    }

    function statusClass(s: string) {
        if (s === "succeeded") return "bg-green-50 text-green-700 border-green-200";
        if (s === "running") return "bg-blue-50 text-blue-700 border-blue-200";
        if (s === "paused" || s === "recovering") return "bg-amber-50 text-amber-700 border-amber-200";
        if (s === "failed" || s === "cancelled") return "bg-red-50 text-red-700 border-red-200";
        return "bg-slate-50 text-slate-600 border-slate-200";
    }

    function formatInput(input: string) {
        try {
            const parsed = JSON.parse(input || "{}");
            return parsed.query || input;
        } catch {
            return input;
        }
    }

    function formatTime(t: string) {
        return t ? new Date(t).toLocaleString("zh-CN") : "-";
    }
</script>

<svelte:head><title>运行任务 - OpenTether</title></svelte:head>

<div class="space-y-5">
    <div class="card">
        <div class="flex items-center justify-between mb-5">
            <div>
                <h2 class="text-xl font-bold text-slate-800">运行任务</h2>
                <p class="text-sm text-slate-500 mt-1">查看 Agent RuntimeJob、Checkpoint，并恢复或取消任务</p>
            </div>
            <button class="px-3 py-2 rounded-lg bg-slate-100 text-sm flex items-center gap-1" on:click={loadJobs}>
                <RefreshCcw size={15} /> 刷新
            </button>
        </div>

        <div class="flex items-center gap-2 mb-4">
            <span class="text-sm text-slate-500">状态</span>
            <select bind:value={status} on:change={loadJobs} class="border border-slate-200 rounded-lg px-3 py-1.5 text-sm">
                {#each statuses as s}
                    <option value={s}>{s || "全部"}</option>
                {/each}
            </select>
        </div>

        {#if loading}
            <div class="text-center py-10 text-slate-400">加载中...</div>
        {:else if jobs.length === 0}
            <div class="text-center py-10 text-slate-400">暂无运行任务</div>
        {:else}
            <div class="overflow-x-auto">
                <table class="w-full text-sm">
                    <thead><tr class="border-b text-left text-slate-500">
                        <th class="py-2 px-3">状态</th>
                        <th class="py-2 px-3">输入</th>
                        <th class="py-2 px-3">步骤</th>
                        <th class="py-2 px-3">更新时间</th>
                        <th class="py-2 px-3 text-right">操作</th>
                    </tr></thead>
                    <tbody>
                        {#each jobs as job}
                            <tr class="border-b hover:bg-slate-50">
                                <td class="py-2 px-3"><span class="px-2 py-1 rounded border text-xs {statusClass(job.status)}">{job.status}</span></td>
                                <td class="py-2 px-3 max-w-md truncate" title={formatInput(job.input)}>{formatInput(job.input)}</td>
                                <td class="py-2 px-3 text-slate-500">{job.current_step}/{job.max_steps}</td>
                                <td class="py-2 px-3 text-slate-500">{formatTime(job.updated_at)}</td>
                                <td class="py-2 px-3 text-right">
                                    <button class="p-1.5 text-slate-500 hover:text-primary-600" title="查看" on:click={() => viewJob(job)}><Eye size={15} /></button>
                                    {#if ["paused", "failed", "recovering"].includes(job.status)}
                                        <button class="p-1.5 text-green-600 hover:text-green-700" title="恢复/重试" on:click={() => retryJob(job)}><Play size={15} /></button>
                                    {/if}
                                    {#if ["pending", "running", "paused", "recovering"].includes(job.status)}
                                        <button class="p-1.5 text-red-500 hover:text-red-700" title="取消" on:click={() => cancelJob(job)}><XCircle size={15} /></button>
                                    {/if}
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    </div>

    {#if selectedJob}
        <div class="card">
            <div class="flex items-center justify-between mb-4">
                <div>
                    <h3 class="font-semibold text-slate-800">任务详情</h3>
                    <p class="text-xs text-slate-400 font-mono">{selectedJob.id}</p>
                </div>
                <span class="px-2 py-1 rounded border text-xs {statusClass(selectedJob.status)}">{selectedJob.status}</span>
            </div>
            <div class="grid grid-cols-2 gap-3 text-sm mb-4">
                <div class="p-3 bg-slate-50 rounded-lg"><div class="text-xs text-slate-400">会话</div><div class="font-mono text-xs">{selectedJob.conversation_id}</div></div>
                <div class="p-3 bg-slate-50 rounded-lg"><div class="text-xs text-slate-400">Lease</div><div class="font-mono text-xs">{selectedJob.lease_owner || "-"}</div></div>
                <div class="p-3 bg-slate-50 rounded-lg"><div class="text-xs text-slate-400">开始</div><div>{formatTime(selectedJob.started_at)}</div></div>
                <div class="p-3 bg-slate-50 rounded-lg"><div class="text-xs text-slate-400">Lease 到期</div><div>{formatTime(selectedJob.lease_expires_at)}</div></div>
            </div>

            <h4 class="text-sm font-semibold text-slate-700 mb-2 flex items-center gap-1"><Clock size={14} /> Checkpoints</h4>
            <div class="space-y-2 max-h-[420px] overflow-y-auto">
                {#each checkpoints as ckpt}
                    <details class="border border-slate-200 rounded-lg bg-white">
                        <summary class="cursor-pointer px-3 py-2 text-sm bg-slate-50">
                            <span class="font-mono text-xs text-slate-400">#{ckpt.step}</span>
                            <span class="ml-2 font-medium">{ckpt.type}</span>
                            <span class="float-right text-xs text-slate-400">{formatTime(ckpt.created_at)}</span>
                        </summary>
                        <pre class="p-3 text-xs whitespace-pre-wrap overflow-x-auto bg-slate-900 text-slate-100 rounded-b-lg">{ckpt.state}</pre>
                    </details>
                {:else}
                    <div class="text-sm text-slate-400">暂无 checkpoint</div>
                {/each}
            </div>
        </div>
    {/if}
</div>
