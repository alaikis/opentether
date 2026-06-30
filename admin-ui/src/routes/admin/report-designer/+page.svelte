<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    let templates: any[] = []; let sections: any[] = [{title:"销售趋势",type:"chart",content:"按月销售额"},{title:"结论",type:"markdown",content:"自动总结"}]; let title="经营分析报告"; let loading=true;
    onMount(load);
    async function load(){ loading=true; try{templates = await api.get<any[]>("/admin/report-templates").catch(()=>[])}finally{loading=false} }
    function add(type:string){ sections=[...sections,{title:type==='chart'?'新图表':'新文本',type,content:''}]; }
    function remove(i:number){ sections=sections.filter((_,idx)=>idx!==i); }
    async function save(){ await api.post("/admin/report-templates", {name:title, sections_json:JSON.stringify(sections)}); toast.success("已保存"); load(); }
</script>
<div class="space-y-6">
    <div><h1 class="text-2xl font-bold">报表设计器</h1><p class="text-sm text-slate-500">多图表、多数据源统一输出</p></div>
    <div class="grid lg:grid-cols-3 gap-4">
        <div class="card space-y-3"><input class="input font-semibold" bind:value={title}/><div class="flex gap-2"><button class="btn-primary" on:click={()=>add('chart')}>+ 图表</button><button class="px-4 py-2 border rounded-lg text-sm" on:click={()=>add('table')}>+ 表格</button><button class="px-4 py-2 border rounded-lg text-sm" on:click={()=>add('markdown')}>+ 文本</button></div>{#each sections as s,i}<div class="border rounded p-2 space-y-2"><div class="flex justify-between"><input class="input" bind:value={s.title}/><select class="input max-w-[100px]" bind:value={s.type}><option value="chart">chart</option><option value="table">table</option><option value="markdown">markdown</option></select></div><textarea class="input" rows="2" bind:value={s.content}/><button class="text-xs text-red-500" on:click={()=>remove(i)}>删除</button></div>{/each}<button class="btn-primary w-full" on:click={save}>保存模板</button></div>
        <div class="card lg:col-span-2"><h2 class="text-xl font-bold mb-4">{title}</h2><div class="space-y-4">{#each sections as s}<div class="border rounded-lg p-4"><h3 class="font-semibold mb-2">{s.title}</h3>{#if s.type==='chart'}<div class="h-40 bg-blue-50 rounded flex items-center justify-center text-blue-500">图表区：{s.content}</div>{:else if s.type==='table'}<div class="h-28 bg-slate-50 rounded flex items-center justify-center text-slate-400">表格区</div>{:else}<p class="text-sm text-slate-600 whitespace-pre-wrap">{s.content}</p>{/if}</div>{/each}</div></div>
    </div>
    {#if templates.length>0}<div class="card"><h2 class="font-semibold mb-2">已保存模板</h2><div class="space-y-2">{#each templates as t}<div class="border rounded-lg p-3 flex justify-between"><div><div class="font-medium">{t.name}</div><div class="text-xs text-slate-500">{t.category||'-'}</div></div><button class="text-xs text-red-500" on:click={async()=>{await api.delete(`/admin/report-templates/${t.id}`);load()}}>删除</button></div>{/each}</div></div>{/if}
</div>