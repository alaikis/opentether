<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    let policies: any[] = []; let loading = true; let form: any = { name:"", scope:"", resource:"", effect:"allow", rules_json:"{}", enabled:true };
    onMount(load);
    async function load(){ loading=true; try{policies = await api.get<any[]>("/admin/policies").catch(()=>[])}finally{loading=false} }
    async function save(){ await api.post("/admin/policies", form); toast.success("已保存"); form={name:"",scope:"",resource:"",effect:"allow",rules_json:"{}",enabled:true}; load(); }
    async function remove(id:string){ if(!confirm("确定删除？"))return; await api.delete(`/admin/policies/${id}`); toast.success("已删除"); load(); }
</script>
<div class="space-y-6">
    <div><h1 class="text-2xl font-bold">策略中心</h1><p class="text-sm text-slate-500">资源级 RBAC、ABAC、字段脱敏、高风险工具审批</p></div>
    <div class="card space-y-3"><h3 class="font-semibold">新增策略</h3><div class="grid grid-cols-4 gap-2"><input class="input" placeholder="名称" bind:value={form.name}/><input class="input" placeholder="scope" bind:value={form.scope}/><input class="input" placeholder="resource" bind:value={form.resource}/><select class="input" bind:value={form.effect}><option value="allow">allow</option><option value="deny">deny</option></select></div><textarea class="input font-mono" rows="4" bind:value={form.rules_json} placeholder="规则 JSON"/><button class="btn-primary" on:click={save}>保存</button></div>
    {#if loading}<div class="text-slate-400">加载中...</div>{:else if policies.length===0}<div class="text-slate-400">暂无策略</div>{:else}
    <div class="card"><table class="w-full text-sm"><thead><tr class="border-b text-xs text-slate-500"><th class="py-2 px-3">名称</th><th class="py-2 px-3">scope</th><th class="py-2 px-3">resource</th><th class="py-2 px-3">effect</th><th class="py-2 px-3">操作</th></tr></thead><tbody>{#each policies as p}<tr class="border-b"><td class="py-2 px-3 text-xs">{p.name}</td><td class="py-2 px-3 text-xs">{p.scope}</td><td class="py-2 px-3 text-xs font-mono">{p.resource}</td><td class="py-2 px-3 text-xs"><span class="px-2 py-0.5 rounded-full {p.effect==='deny'?'bg-red-100 text-red-700':'bg-green-100 text-green-700'}">{p.effect}</span></td><td class="py-2 px-3"><button class="text-xs text-red-500" on:click={()=>remove(p.id)}>删除</button></td></tr>{/each}</tbody></table></div>
    {/if}
</div>