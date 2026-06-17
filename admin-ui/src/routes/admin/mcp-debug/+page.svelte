<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    let configs: any[] = [];
    let selected = "";
    let selectedConfig: any = null;
    let tools: any[] = [];
    let toolName = "";
    let args = "{}";
    let result = "";
    let loading = false;
    onMount(load);
    async function load(){ configs = await api.get<any[]>("/admin/mcp/configs").catch(()=>[]); }
    async function loadTools(){ selectedConfig=configs.find(c=>c.id===selected); tools=[]; result=""; if(!selected)return; loading=true; try{ tools = await api.get<any[]>(`/admin/mcp/configs/${selected}/tools`).catch(()=>[]); } finally{loading=false;} }
    async function start(){ if(!selected)return; await api.post(`/admin/mcp/configs/${selected}/start`); toast.success("已启动"); await load(); await loadTools(); }
    async function stop(){ if(!selected)return; await api.post(`/admin/mcp/configs/${selected}/stop`); toast.success("已停止"); await load(); await loadTools(); }
    async function callTool(){ try{ const parsed=JSON.parse(args||"{}"); const r=await api.post<any>(`/admin/mcp/configs/${selected}/call`, { tool_name: toolName, arguments: parsed }); result=JSON.stringify(r,null,2); }catch(e:any){ result=e.message||"调用失败"; toast.error(e.message||"调用失败"); } }
    $: if (toolName) { const t=tools.find(x=>x.name===toolName); if(t?.input_schema){ args=JSON.stringify(exampleFromSchema(t.input_schema),null,2); } }
    function exampleFromSchema(schema:any){ const out:any={}; const props=schema?.properties||{}; for(const k of Object.keys(props)){ const type=props[k]?.type; out[k]=type==='number'?0:type==='boolean'?false:""; } return out; }
</script>
<div class="space-y-6">
    <div><h1 class="text-2xl font-bold">MCP 调试台</h1><p class="text-sm text-slate-500">启动/停止、tools/list、schema 参数样例、tool call 测试。</p></div>
    <div class="card space-y-3">
        <select class="input" bind:value={selected} on:change={loadTools}><option value="">选择 MCP</option>{#each configs as c}<option value={c.id}>{c.name} / {c.transport || 'stdio'} / {c.status}</option>{/each}</select>
        {#if selectedConfig}<div class="text-xs text-slate-500">URL: {selectedConfig.url || '-'} | Command: {selectedConfig.command || '-'}</div><div class="flex gap-2"><button class="btn-primary" on:click={start}>启动</button><button class="px-4 py-2 rounded-lg border text-sm" on:click={stop}>停止</button></div>{/if}
        {#if loading}<div class="text-sm text-slate-400">加载工具...</div>{/if}
        <select class="input" bind:value={toolName}><option value="">选择工具</option>{#each tools as t}<option value={t.name}>{t.name}</option>{/each}</select>
        {#if toolName}<pre class="bg-slate-100 p-3 rounded text-xs overflow-auto">{JSON.stringify(tools.find(t=>t.name===toolName),null,2)}</pre>{/if}
        <textarea class="input font-mono" rows="6" bind:value={args} />
        <button class="btn-primary" on:click={callTool}>调用工具</button>
        <pre class="bg-slate-950 text-slate-100 p-3 rounded-lg text-xs overflow-auto">{result}</pre>
    </div>
</div>
