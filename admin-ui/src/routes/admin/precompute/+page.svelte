<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    let jobs: any[] = []; let loading = true; let form: any = { name:"", target_table:"", sql:"", schedule:"" };
    onMount(load);
    async function load(){ loading=true; try{jobs = await api.get<any[]>("/admin/precompute/jobs").catch(()=>[])}finally{loading=false} }
    async function save(){ await api.post("/admin/precompute/jobs", form); toast.success("已保存"); form={name:"",target_table:"",sql:"",schedule:""}; load(); }
    async function run(id:string){ await api.post(`/admin/precompute/jobs/${id}/run`); toast.success("已执行"); load(); }
    async function remove(id:string){ if(!confirm("确定删除？"))return; await api.delete(`/admin/precompute/jobs/${id}`); toast.success("已删除"); load(); }
</script>
<div class="space-y-6">
    <div><h1 class="text-2xl font-bold">预计算指标</h1><p class="text-sm text-slate-500">高频管理指标物化表，秒级查询</p></div>
    <div class="card space-y-3"><h3 class="font-semibold">新增任务</h3><div class="grid grid-cols-2 gap-2"><input class="input" placeholder="名称" bind:value={form.name}/><input class="input" placeholder="目标表" bind:value={form.target_table}/></div><textarea class="input font-mono" rows="4" bind:value={form.sql} placeholder="SELECT ..."/><input class="input" placeholder="调度(Cron)" bind:value={form.schedule}/><button class="btn-primary" on:click={save}>保存</button></div>
    {#if loading}<div class="text-slate-400">加载中...</div>{:else if jobs.length===0}<div class="text-slate-400">暂无任务</div>{:else}
    <div class="card"><table class="w-full text-sm"><thead><tr class="border-b text-xs text-slate-500"><th class="py-2 px-3">名称</th><th class="py-2 px-3">目标表</th><th class="py-2 px-3">调度</th><th class="py-2 px-3">状态</th><th class="py-2 px-3">操作</th></tr></thead><tbody>{#each jobs as j}<tr class="border-b"><td class="py-2 px-3 text-xs">{j.name}</td><td class="py-2 px-3 text-xs font-mono">{j.target_table}</td><td class="py-2 px-3 text-xs">{j.schedule||'-'}</td><td class="py-2 px-3 text-xs"><span class="px-2 py-0.5 rounded-full {j.status==='completed'?'bg-green-100 text-green-700':'bg-slate-100 text-slate-600'}">{j.status}</span></td><td class="py-2 px-3 flex gap-2"><button class="text-xs text-primary-600" on:click={()=>run(j.id)}>执行</button><button class="text-xs text-red-500" on:click={()=>remove(j.id)}>删除</button></td></tr>{/each}</tbody></table></div>
    {/if}
</div>