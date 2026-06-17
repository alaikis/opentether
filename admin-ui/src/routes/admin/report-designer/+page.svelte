<script lang="ts">
    let sections = [{ title: "销售趋势", type: "chart", content: "按月销售额柱状图" }, { title: "关键结论", type: "markdown", content: "自动总结环比、峰值、异常月份。" }];
    let title = "经营分析报告";
    function add(type: string){ sections=[...sections,{title:type==='chart'?'新图表':'新文本',type,content:''}]; }
    function remove(i:number){ sections=sections.filter((_,idx)=>idx!==i); }
</script>
<div class="space-y-6">
    <div><h1 class="text-2xl font-bold">报表设计器</h1><p class="text-sm text-slate-500">多图表、多数据源、IM/PDF/HTML 统一输出设计入口。</p></div>
    <div class="grid lg:grid-cols-3 gap-4">
        <div class="card space-y-3"><input class="input font-semibold" bind:value={title}/><div class="flex gap-2"><button class="btn-primary" on:click={()=>add('chart')}>+ 图表</button><button class="px-4 py-2 rounded-lg border text-sm" on:click={()=>add('markdown')}>+ 文本</button></div>{#each sections as s,i}<div class="border rounded p-2 space-y-2"><input class="input" bind:value={s.title}/><select class="input" bind:value={s.type}><option value="chart">chart</option><option value="table">table</option><option value="markdown">markdown</option></select><textarea class="input" rows="3" bind:value={s.content}/><button class="text-xs text-red-600" on:click={()=>remove(i)}>删除</button></div>{/each}</div>
        <div class="card lg:col-span-2"><h2 class="text-xl font-bold mb-4">{title}</h2><div class="space-y-4">{#each sections as s}<div class="border rounded-lg p-4"><h3 class="font-semibold mb-2">{s.title}</h3>{#if s.type==='chart'}<div class="h-40 bg-blue-50 rounded flex items-center justify-center text-blue-500">图表区：{s.content}</div>{:else if s.type==='table'}<div class="h-28 bg-slate-50 rounded flex items-center justify-center text-slate-400">表格区</div>{:else}<p class="text-sm text-slate-600 whitespace-pre-wrap">{s.content}</p>{/if}</div>{/each}</div></div>
    </div>
</div>
