<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";

    let products: any[] = [];
    let releases: any[] = [];
    let artifacts: any[] = [];
    let contents: any[] = [];
    let stats: any = {};
    let loading = true;
    let tab = "releases";

    let product = { name: "", slug: "", description: "", status: "active" };
    let release = { product_id: "", version: "", channel: "stable", changelog: "", status: "draft" };
    let artifact = { release_id: "", os: "linux", arch: "amd64", file_name: "", file_url: "", checksum: "", size: 0 };
    let content = { key: "home", title: "", body_md: "", status: "published" };

    onMount(loadAll);

    async function loadAll() {
        loading = true;
        try {
            const [p, r, a, c, s] = await Promise.all([
                api.get<any[]>("/admin/cloud/products"),
                api.get<any[]>("/admin/cloud/releases"),
                api.get<any[]>("/admin/cloud/artifacts"),
                api.get<any[]>("/admin/cloud/site-contents"),
                api.get<any>("/admin/cloud/stats"),
            ]);
            products = p || [];
            releases = r || [];
            artifacts = a || [];
            contents = c || [];
            stats = s || {};
        } finally {
            loading = false;
        }
    }

    async function saveProduct() {
        await api.post("/admin/cloud/products", product);
        toast.success("产品已保存");
        product = { name: "", slug: "", description: "", status: "active" };
        loadAll();
    }
    async function saveRelease() {
        await api.post("/admin/cloud/releases", release);
        toast.success("版本已保存");
        release = { product_id: "", version: "", channel: "stable", changelog: "", status: "draft" };
        loadAll();
    }
    async function publishRelease(id: string, published: boolean) {
        await api.post(`/admin/cloud/releases/${id}/publish?published=${published}`);
        toast.success(published ? "已发布" : "已取消发布");
        loadAll();
    }
    async function saveArtifact() {
        await api.post("/admin/cloud/artifacts", artifact);
        toast.success("文件已保存");
        artifact = { release_id: "", os: "linux", arch: "amd64", file_name: "", file_url: "", checksum: "", size: 0 };
        loadAll();
    }
    async function saveContent() {
        await api.post("/admin/cloud/site-contents", content);
        toast.success("内容已保存");
        content = { key: "home", title: "", body_md: "", status: "published" };
        loadAll();
    }
</script>

<svelte:head><title>云端管理 - OpenTether</title></svelte:head>

<div class="max-w-6xl space-y-6">
    <div>
        <h1 class="text-2xl font-bold text-slate-800">云端官网与版本管理</h1>
        <p class="text-sm text-slate-500 mt-1">统一管理官网内容、版本、下载文件和下载统计</p>
    </div>

    <div class="grid grid-cols-3 gap-4">
        <div class="card"><div class="text-xs text-slate-500">版本数</div><div class="text-2xl font-bold">{releases.length}</div></div>
        <div class="card"><div class="text-xs text-slate-500">文件数</div><div class="text-2xl font-bold">{stats.artifacts || artifacts.length}</div></div>
        <div class="card"><div class="text-xs text-slate-500">下载数</div><div class="text-2xl font-bold">{stats.downloads || 0}</div></div>
    </div>

    <div class="flex gap-1 border-b border-slate-200">
        {#each ["products", "releases", "artifacts", "contents"] as t}
            <button class="px-4 py-2 text-sm border-b-2 {tab === t ? 'border-primary-600 text-primary-600' : 'border-transparent text-slate-500'}" on:click={() => (tab = t)}>{t}</button>
        {/each}
    </div>

    {#if loading}
        <div class="text-slate-400">加载中...</div>
    {:else if tab === "products"}
        <div class="card space-y-3">
            <h3 class="font-semibold">产品</h3>
            <div class="grid grid-cols-4 gap-2">
                <input class="input" placeholder="名称" bind:value={product.name} />
                <input class="input" placeholder="slug" bind:value={product.slug} />
                <input class="input col-span-2" placeholder="描述" bind:value={product.description} />
            </div>
            <button class="btn-primary" on:click={saveProduct}>保存产品</button>
            {#each products as p}<div class="border-t py-2 text-sm">{p.name} / {p.slug} / {p.status}</div>{/each}
        </div>
    {:else if tab === "releases"}
        <div class="card space-y-3">
            <h3 class="font-semibold">版本</h3>
            <div class="grid grid-cols-4 gap-2">
                <select class="input" bind:value={release.product_id}><option value="">产品</option>{#each products as p}<option value={p.id}>{p.name}</option>{/each}</select>
                <input class="input" placeholder="版本号" bind:value={release.version} />
                <input class="input" placeholder="channel" bind:value={release.channel} />
                <select class="input" bind:value={release.status}><option value="draft">draft</option><option value="published">published</option></select>
            </div>
            <textarea class="input" rows="3" placeholder="changelog" bind:value={release.changelog} />
            <button class="btn-primary" on:click={saveRelease}>保存版本</button>
            {#each releases as r}
                <div class="border-t py-2 text-sm flex justify-between"><span>{r.version} / {r.channel} / {r.status}</span><button class="text-primary-600" on:click={() => publishRelease(r.id, r.status !== 'published')}>{r.status === 'published' ? '取消发布' : '发布'}</button></div>
            {/each}
        </div>
    {:else if tab === "artifacts"}
        <div class="card space-y-3">
            <h3 class="font-semibold">下载文件</h3>
            <div class="grid grid-cols-3 gap-2">
                <select class="input" bind:value={artifact.release_id}><option value="">版本</option>{#each releases as r}<option value={r.id}>{r.version}</option>{/each}</select>
                <input class="input" placeholder="os" bind:value={artifact.os} />
                <input class="input" placeholder="arch" bind:value={artifact.arch} />
                <input class="input" placeholder="文件名" bind:value={artifact.file_name} />
                <input class="input col-span-2" placeholder="文件URL" bind:value={artifact.file_url} />
            </div>
            <button class="btn-primary" on:click={saveArtifact}>保存文件</button>
            {#each artifacts as a}<div class="border-t py-2 text-sm">{a.file_name} / {a.os}-{a.arch} / <a class="text-primary-600" href={a.file_url} target="_blank">打开</a></div>{/each}
        </div>
    {:else}
        <div class="card space-y-3">
            <h3 class="font-semibold">官网内容</h3>
            <div class="grid grid-cols-3 gap-2">
                <input class="input" placeholder="key" bind:value={content.key} />
                <input class="input" placeholder="标题" bind:value={content.title} />
                <select class="input" bind:value={content.status}><option value="draft">draft</option><option value="published">published</option></select>
            </div>
            <textarea class="input font-mono" rows="8" placeholder="Markdown 内容" bind:value={content.body_md} />
            <button class="btn-primary" on:click={saveContent}>保存内容</button>
            {#each contents as c}<div class="border-t py-2 text-sm">{c.key} / {c.title} / {c.status}</div>{/each}
        </div>
    {/if}
</div>
