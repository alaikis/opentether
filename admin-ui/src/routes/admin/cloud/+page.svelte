<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    let products: any[]=[]; let releases: any[]=[]; let artifacts: any[]=[]; let loading=true; let stats: any={};
    let product: any={name:"",slug:"",description:"",status:"active"};
    let release: any={product_id:"",version:"",channel:"stable",changelog:"",status:"draft"};
    let artifact: any={release_id:"",os:"linux",arch:"amd64",file_name:"",file_url:"",checksum:""};
    let tab="releases";
    onMount(loadAll);
    async function loadAll(){ loading=true; try{[products,releases,artifacts,stats]=await Promise.all([api.get<any[]>("/admin/cloud/products"),api.get<any[]>("/admin/cloud/releases"),api.get<any[]>("/admin/cloud/artifacts"),api.get<any>("/admin/cloud/stats")])}finally{loading=false} }
    async function saveProduct(){ await api.post("/admin/cloud/products",product); toast.success("已保存"); product={name:"",slug:"",description:"",status:"active"}; loadAll(); }
    async function saveRelease(){ await api.post("/admin/cloud/releases",release); toast.success("已保存"); release={product_id:"",version:"",channel:"stable",changelog:"",status:"draft"}; loadAll(); }
    async function publish(release:any){ await api.post(`/admin/cloud/releases/${release.id}/publish?published=${release.status!=='published'}`); toast.success(release.status!=='published'?"已发布":"已取消发布"); loadAll(); }
    async function saveArtifact(){ await api.post("/admin/cloud/artifacts",artifact); toast.success("已保存"); artifact={release_id:"",os:"linux",arch:"amd64",file_name:"",file_url:"",checksum:""}; loadAll(); }
</script>
<div class="space-y-6">
    <div><h1 class="text-2xl font-bold">云端管理</h1><p class="text-sm text-slate-500">产品、版本、下载文件、统计</p></div>
    <div class="grid grid-cols-3 gap-4"><div class="card"><div class="text-xs text-slate-500">版本</div><div class="text-2xl font-bold">{releases.length}</div></div><div class="card"><div class="text-xs text-slate-500">文件</div><div class="text-2xl font-bold">{stats.artifacts||artifacts.length}</div></div><div class="card"><div class="text-xs text-slate-500">下载</div><div class="text-2xl font-bold">{stats.downloads||0}</div></div></div>
    <div class="flex gap-1 border-b border-slate-200">{#each ["products","releases","artifacts"] as t}<button class="px-4 py-2 text-sm border-b-2 {tab===t?'border-primary-600 text-primary-600':'border-transparent text-slate-500'}" on:click={()=>tab=t}>{t}</button>{/each}</div>
    {#if loading}<div class="text-slate-400">加载中...</div>{:else if tab==="products"}
        <div class="card space-y-3"><h3 class="font-semibold">产品</h3><div class="grid grid-cols-4 gap-2"><input class="input" placeholder="名称" bind:value={product.name}/><input class="input" placeholder="slug" bind:value={product.slug}/><input class="input col-span-2" placeholder="描述" bind:value={product.description}/></div><button class="btn-primary" on:click={saveProduct}>保存</button>{#each products as p}<div class="border-t py-2 text-sm">{p.name} / {p.slug} / {p.status}</div>{/each}</div>
    {:else if tab==="releases"}
        <div class="card space-y-3"><h3 class="font-semibold">版本</h3><div class="grid grid-cols-4 gap-2"><select class="input" bind:value={release.product_id}><option value="">产品</option>{#each products as p}<option value={p.id}>{p.name}</option>{/each}</select><input class="input" placeholder="版本号" bind:value={release.version}/><input class="input" placeholder="channel" bind:value={release.channel}/><select class="input" bind:value={release.status}><option value="draft">draft</option><option value="published">published</option></select></div><textarea class="input" rows="3" placeholder="changelog" bind:value={release.changelog}/><button class="btn-primary" on:click={saveRelease}>保存</button>{#each releases as r}<div class="border-t py-2 text-sm flex justify-between"><span>{r.version} / {r.channel} / {r.status}</span><button class="text-primary-600" on:click={()=>publish(r)}>{r.status==='published'?'取消发布':'发布'}</button></div>{/each}</div>
    {:else}
        <div class="card space-y-3"><h3 class="font-semibold">下载文件</h3><div class="grid grid-cols-3 gap-2"><select class="input" bind:value={artifact.release_id}><option value="">版本</option>{#each releases as r}<option value={r.id}>{r.version}</option>{/each}</select><input class="input" placeholder="os" bind:value={artifact.os}/><input class="input" placeholder="arch" bind:value={artifact.arch}/><input class="input" placeholder="文件名" bind:value={artifact.file_name}/><input class="input col-span-2" placeholder="文件URL" bind:value={artifact.file_url}/></div><button class="btn-primary" on:click={saveArtifact}>保存</button>{#each artifacts as a}<div class="border-t py-2 text-sm">{a.file_name} / {a.os}-{a.arch} / <a class="text-primary-600" href={a.file_url} target="_blank">打开</a></div>{/each}</div>
    {/if}
</div>