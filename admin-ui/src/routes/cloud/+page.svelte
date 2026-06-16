<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";

    let products: any[] = [];
    let releases: any[] = [];
    let site: any = null;
    let loading = true;

    onMount(async () => {
        try {
            const [p, r, s] = await Promise.all([
                api.get<any[]>("/cloud/public/products"),
                api.get<any[]>("/cloud/public/releases"),
                api.get<any>("/cloud/public/site/home").catch(() => null),
            ]);
            products = p || [];
            releases = r || [];
            site = s;
        } finally {
            loading = false;
        }
    });
</script>

<svelte:head><title>OpenTether Cloud</title></svelte:head>

<div class="min-h-screen bg-slate-950 text-white">
    <section class="max-w-6xl mx-auto px-6 py-20">
        <div class="max-w-3xl">
            <p class="text-primary-300 text-sm font-semibold mb-4">OpenTether Cloud</p>
            <h1 class="text-5xl font-bold tracking-tight mb-6">企业智能体云端官网与版本下载中心</h1>
            <p class="text-slate-300 text-lg leading-8">{site?.body_md || "统一发布产品信息、版本记录、下载文件和更新公告。"}</p>
        </div>
    </section>

    {#if loading}
        <div class="max-w-6xl mx-auto px-6 text-slate-400">加载中...</div>
    {:else}
        <section class="max-w-6xl mx-auto px-6 py-10 grid md:grid-cols-2 gap-6">
            {#each products as p}
                <div class="rounded-2xl bg-white/10 border border-white/10 p-6">
                    <h2 class="text-xl font-semibold mb-2">{p.name}</h2>
                    <p class="text-slate-300 text-sm">{p.description}</p>
                </div>
            {/each}
        </section>

        <section class="max-w-6xl mx-auto px-6 py-10">
            <h2 class="text-2xl font-bold mb-4">最新版本</h2>
            <div class="space-y-3">
                {#each releases as r}
                    <div class="rounded-xl bg-white/10 border border-white/10 p-4 flex items-center justify-between">
                        <div>
                            <div class="font-semibold">{r.version} <span class="text-xs text-slate-400">{r.channel}</span></div>
                            <div class="text-sm text-slate-300">{r.changelog}</div>
                        </div>
                        <span class="text-xs text-green-300">{r.status}</span>
                    </div>
                {/each}
            </div>
        </section>
    {/if}
</div>
