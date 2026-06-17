<script lang="ts">
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    let goal = "";
    let graphID = "";
    let graph: any = null;
    let nodes: any[] = [];
    let outputs: any[] = [];
    let poll: any = null;
    async function createGraph() {
        const g = await api.post<any>("/admin/agent-task-graphs", { goal });
        graphID = g.id;
        toast.success("长任务已创建");
        startPolling();
    }
    async function loadGraph() {
        if (!graphID) return;
        const data = await api.get<any>(`/admin/agent-task-graphs/${graphID}`);
        graph = data.graph; nodes = data.nodes || []; outputs = data.outputs || [];
        if (graph.status === "completed" || graph.status === "failed") stopPolling();
    }
    function startPolling(){ stopPolling(); loadGraph(); poll=setInterval(loadGraph, 1500); }
    function stopPolling(){ if(poll) clearInterval(poll); poll=null; }
    async function retry(n:any){ await api.post(`/admin/agent-task-graphs/nodes/${n.id}/retry`); startPolling(); }
    async function skip(n:any){ await api.post(`/admin/agent-task-graphs/nodes/${n.id}/skip`); startPolling(); }
    async function resume(){ await api.post(`/admin/agent-task-graphs/${graphID}/resume`); startPolling(); }
</script>
<div class="space-y-6">
    <div><h1 class="text-2xl font-bold">长任务执行</h1><p class="text-sm text-slate-500">Task Graph、节点 checkpoint、重试/跳过/恢复。</p></div>
    <div class="card space-y-3"><textarea class="input" rows="4" bind:value={goal} placeholder="输入长任务目标"/><button class="btn-primary" on:click={createGraph}>创建并执行</button><div class="flex gap-2"><input class="input" bind:value={graphID} placeholder="Graph ID"/><button class="btn-primary" on:click={startPolling}>查看</button>{#if graphID}<button class="px-4 py-2 border rounded-lg text-sm" on:click={resume}>恢复执行</button>{/if}</div></div>
    {#if graph}<div class="card"><div class="font-semibold mb-2">{graph.goal}</div><div class="text-sm text-slate-500">状态：{graph.status}</div>{#if graph.summary}<pre class="mt-3 bg-slate-50 p-3 rounded text-xs whitespace-pre-wrap">{graph.summary}</pre>{/if}</div>{/if}
    {#if nodes.length}<div class="card space-y-2"><h2 class="font-semibold">节点</h2>{#each nodes as n}<div class="border rounded-lg p-3"><div class="flex justify-between"><div><div class="font-medium">{n.name}</div><div class="text-xs text-slate-500">{n.type} / {n.status}</div></div><div class="flex gap-2"><button class="text-xs text-blue-600" on:click={()=>retry(n)}>重试</button><button class="text-xs text-orange-600" on:click={()=>skip(n)}>跳过</button></div></div>{#if n.summary}<pre class="mt-2 text-xs bg-slate-50 p-2 rounded whitespace-pre-wrap">{n.summary}</pre>{/if}{#if n.error}<div class="text-xs text-red-600 mt-2">{n.error}</div>{/if}</div>{/each}</div>{/if}
</div>
