<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    let audits: any[] = [];
    let requests: any[] = [];
    let loading = true;
    onMount(async () => {
        try { audits = await api.get<any[]>("/admin/logs/audit?limit=50").catch(() => []); } catch {}
        try { requests = await api.get<any[]>("/admin/logs/request?limit=50").catch(() => []); } catch {}
        loading = false;
    });
</script>
<div class="space-y-6">
    <div><h1 class="text-2xl font-bold">观测看板</h1><p class="text-sm text-slate-500">请求日志、审计日志、慢查询入口</p></div>
    {#if loading}<div class="text-slate-400">加载中...</div>{:else}
    <div class="grid md:grid-cols-3 gap-4">
        <div class="card"><div class="text-xs text-slate-500">审计日志</div><div class="text-2xl font-bold">{audits.length}</div></div>
        <div class="card"><div class="text-xs text-slate-500">请求日志</div><div class="text-2xl font-bold">{requests.length}</div></div>
        <div class="card"><div class="text-xs text-slate-500">慢查询阈值</div><div class="text-2xl font-bold">3s</div></div>
    </div>
    <div class="card"><h2 class="font-semibold mb-2">最近请求</h2><div class="overflow-x-auto"><table class="w-full text-sm"><thead><tr class="border-b text-xs text-slate-500"><th class="py-2 px-3">方法</th><th class="py-2 px-3">路径</th><th class="py-2 px-3">状态</th><th class="py-2 px-3">耗时</th><th class="py-2 px-3">时间</th></tr></thead><tbody>{#each requests.slice(0,20) as r}<tr class="border-b"><td class="py-2 px-3 text-xs">{r.method}</td><td class="py-2 px-3 text-xs font-mono">{r.path}</td><td class="py-2 px-3 text-xs">{r.status}</td><td class="py-2 px-3 text-xs">{r.latency_ms}ms</td><td class="py-2 px-3 text-xs text-slate-400">{r.created_at}</td></tr>{/each}</tbody></table></div></div>
    <div class="card"><h2 class="font-semibold mb-2">最近审计</h2><div class="overflow-x-auto"><table class="w-full text-sm"><thead><tr class="border-b text-xs text-slate-500"><th class="py-2 px-3">动作</th><th class="py-2 px-3">资源</th><th class="py-2 px-3">用户</th><th class="py-2 px-3">时间</th></tr></thead><tbody>{#each audits.slice(0,20) as a}<tr class="border-b"><td class="py-2 px-3 text-xs">{a.action}</td><td class="py-2 px-3 text-xs">{a.resource}</td><td class="py-2 px-3 text-xs">{a.user_id}</td><td class="py-2 px-3 text-xs text-slate-400">{a.created_at}</td></tr>{/each}</tbody></table></div></div>
    {/if}
</div>