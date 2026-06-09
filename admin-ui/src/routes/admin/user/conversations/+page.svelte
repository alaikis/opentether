<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { MessageSquare, ArrowLeft } from "lucide-svelte";
    import { goto } from "$app/navigation";

    interface Conversation {
        id: string;
        title: string;
        source: string;
        created_at: string;
        updated_at: string;
    }

    let conversations: Conversation[] = [];
    let loading = true;
    let error = "";

    onMount(async () => {
        await loadConversations();
    });

    async function loadConversations() {
        loading = true;
        error = "";
        try {
            const data = await api.get<Conversation[]>("/user/conversations");
            conversations = Array.isArray(data) ? data : [];
        } catch (e: any) {
            error = e.message || "加载对话列表失败";
            conversations = [];
        } finally {
            loading = false;
        }
    }

    function openConversation(conv: Conversation) {
        goto(`/admin/user/chat?id=${conv.id}`);
    }

    function formatTime(t: string) {
        return new Date(t).toLocaleString("zh-CN", {
            month: "2-digit",
            day: "2-digit",
            hour: "2-digit",
            minute: "2-digit",
        });
    }

    function getSourceLabel(source: string) {
        const map: Record<string, string> = {
            web: "Web",
            im: "IM",
            api: "API",
        };
        return map[source] || source;
    }
</script>

<svelte:head><title>对话历史 - OpenTether</title></svelte:head>

<div class="max-w-3xl mx-auto">
    <div class="flex items-center gap-3 mb-6">
        <button
            class="p-2 text-slate-400 hover:text-slate-600 hover:bg-slate-100 rounded-lg transition-colors"
            on:click={() => goto("/admin/user")}
        >
            <ArrowLeft size={18} />
        </button>
        <h2 class="text-xl font-bold text-slate-800">对话历史</h2>
    </div>

    {#if error}
        <div
            class="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm"
        >
            {error}
            <button class="ml-2 underline" on:click={loadConversations}
                >重试</button
            >
        </div>
    {/if}

    {#if loading}
        <div class="text-center py-12 text-slate-400">加载中...</div>
    {:else if conversations.length === 0}
        <div class="card text-center py-12">
            <div class="text-4xl mb-3">💬</div>
            <p class="text-slate-500">暂无对话记录</p>
            <p class="text-sm text-slate-400 mt-1">开始一个新对话吧</p>
            <button
                class="mt-4 px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 transition-colors"
                on:click={() => goto("/admin/user/chat")}
            >
                新建对话
            </button>
        </div>
    {:else}
        <div class="space-y-3">
            {#each conversations as conv}
                <button
                    class="card w-full text-left flex items-center justify-between hover:border-primary-200 hover:shadow-sm transition-all group"
                    on:click={() => openConversation(conv)}
                >
                    <div class="flex items-center gap-3">
                        <div
                            class="w-10 h-10 rounded-lg bg-slate-100 flex items-center justify-center"
                        >
                            <MessageSquare size={18} class="text-slate-500" />
                        </div>
                        <div>
                            <div
                                class="font-medium text-slate-800 group-hover:text-primary-600 transition-colors"
                            >
                                {conv.title || "未命名对话"}
                            </div>
                            <div class="text-xs text-slate-400 mt-0.5">
                                {formatTime(conv.created_at)}
                                <span class="mx-1">·</span>
                                {getSourceLabel(conv.source)}
                            </div>
                        </div>
                    </div>
                    <span
                        class="text-slate-300 group-hover:text-primary-500 transition-colors"
                    >
                        →
                    </span>
                </button>
            {/each}
        </div>
    {/if}
</div>
