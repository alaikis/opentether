<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { Plus, Trash2, Webhook } from "lucide-svelte";

    let configs: any[] = [];
    let loading = true;
    let showModal = false;
    let editing = false;
    let form: any = { name: "", url: "", secret: "", events: "*", enabled: true };

    onMount(load);
    async function load() { loading = true; try { configs = await api.get<any[]>("/admin/webhooks") || []; } finally { loading = false; } }
    function openAdd() { editing = false; form = { name: "", url: "", secret: "", events: "*", enabled: true }; showModal = true; }
    function openEdit(c: any) { editing = true; form = { ...c }; showModal = true; }
    async function save() { await api.post("/admin/webhooks", form); toast.success("已保存"); showModal = false; load(); }
    async function remove(id: string) { if (!confirm("确定删除？")) return; await api.delete(`/admin/webhooks/${id}`); toast.success("已删除"); load(); }
</script>

<div class="space-y-6">
    <div class="flex items-center justify-between">
        <div><h1 class="text-2xl font-bold">Webhook 通知</h1><p class="text-sm text-slate-500">任务完成后通过 webhook 异步通知外部系统</p></div>
        <button class="btn-primary flex items-center gap-2" on:click={openAdd}><Plus size={14} />新增</button>
    </div>
    {#if loading}
        <div class="text-slate-400">加载中...</div>
    {:else if configs.length === 0}
        <div class="card text-center py-10 text-slate-400"><Webhook size={48} class="mx-auto mb-3" />暂无 Webhook 配置</div>
    {:else}
        <div class="card space-y-2">{#each configs as c}<div class="flex items-center justify-between border-b py-3 last:border-0"><div><div class="font-medium">{c.name}</div><div class="text-xs text-slate-500">{c.url}</div><div class="text-xs text-slate-400 mt-1">事件: {c.events || '*'} | {c.enabled ? '启用' : '禁用'}</div></div><div class="flex gap-2"><button class="text-xs text-primary-600" on:click={() => openEdit(c)}>编辑</button><button class="text-xs text-red-500" on:click={() => remove(c.id)}><Trash2 size={14} /></button></div></div>{/each}</div>
    {/if}
</div>

{#if showModal}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40" on:click|self={() => (showModal = false)}>
        <div class="bg-white rounded-2xl shadow-xl w-full max-w-lg p-6 space-y-4">
            <h3 class="font-bold text-lg">{editing ? "编辑" : "新增"} Webhook</h3>
            <input class="input" placeholder="名称" bind:value={form.name} />
            <input class="input" placeholder="URL（可选，https://...）" bind:value={form.url} />
            <input class="input" placeholder="Secret（可选，用于签名验证）" bind:value={form.secret} />
            <input class="input" placeholder="事件过滤（逗号分隔，*表示全部）" bind:value={form.events} />
            <label class="inline-flex items-center gap-2"><input type="checkbox" bind:checked={form.enabled} />启用</label>
            <div class="flex justify-end gap-3 pt-2"><button class="px-4 py-2 border rounded-lg" on:click={() => (showModal = false)}>取消</button><button class="btn-primary" on:click={save}>保存</button></div>
        </div>
    </div>
{/if}