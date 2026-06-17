<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    let audits: any[] = [];
    let requests: any[] = [];
    onMount(async () => {
        audits = await api.get<any[]>("/admin/logs/audit?limit=50").catch(() => []);
        requests = await api.get<any[]>("/admin/logs/request?limit=50").catch(() => []);
    });
</script>

<div class="space-y-6">
    <div><h1 class="text-2xl font-bold">慢路径与成本看板</h1><p class="text-sm text-slate-500">展示请求、审计、LLM/SQL/工具耗时元数据。后续可接入 Prometheus/OpenTelemetry。</p></div>
    <div class="grid md:grid-cols-3 gap-4">
        <div class="card"><div class="text-xs text-slate-500">审计日志</div><div class="text-2xl font-bold">{audits.length}</div></div>
        <div class="card"><div class="text-xs text-slate-500">请求日志</div><div class="text-2xl font-bold">{requests.length}</div></div>
        <div class="card"><div class="text-xs text-slate-500">慢路径阈值</div><div class="text-2xl font-bold">3s</div></div>
    </div>
    <div class="card"><h2 class="font-semibold mb-2">最近审计</h2>{#each audits.slice(0,20) as a}<div class="text-xs border-b py-1">{a.action || a.method} / {a.resource || a.path} / {a.created_at}</div>{/each}</div>
</div>
