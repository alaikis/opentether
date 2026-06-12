<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { MessageSquare, ArrowLeft, Trash2 } from "lucide-svelte";
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
            const data = await api.get<Conversation[]>(
                "/user/conversations?source=web",
            );
            conversations = Array.isArray(data) ? data : [];
        } catch (e: any) {
            error = e.message || "加载对话列表失败";
            conversations = [];
        } finally {
            loading = false;
        }
    }

    function openConversation(conv: Conversation) {
        goto(`/admin/user?conv=${conv.id}`);
    }

    async function deleteConversation(id: string, e: Event) {
        e.stopPropagation();
        if (!confirm("确认删除此对话？")) return;
        try {
            await api.delete(`/user/conversations/${id}`);
            conversations = conversations.filter((c) => c.id !== id);
            toast.success("对话已删除");
        } catch (err: any) {
            toast.error(err.message || "删除失败");
        }
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

<div class="max-w-3xl mx-auto p-6">
    <div class="flex items-center gap-3 mb-6">
        <button
            class="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
            on:click={() => goto("/admin/user")}
        >
            <ArrowLeft size={18} />
        </button>
        <h2 class="text-xl font-bold text-gray-800">对话历史</h2>
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
        <div class="text-center py-12 text-gray-400">加载中...</div>
    {:else if conversations.length === 0}
        <div class="text-center py-12">
            <div class="text-4xl mb-3">💬</div>
            <p class="text-gray-500">暂无对话记录</p>
            <p class="text-sm text-gray-400 mt-1">开始一个新对话吧</p>
            <button
                class="mt-4 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors"
                on:click={() => goto("/admin/user")}
            >
                返回 AI 对话
            </button>
        </div>
    {:else}
        <div class="space-y-2">
            {#each conversations as conv}
                <button
                    class="w-full text-left p-4 border border-gray-200 rounded-lg flex items-center justify-between hover:border-blue-300 hover:shadow-sm transition-all group"
                    on:click={() => openConversation(conv)}
                >
                    <div class="flex items-center gap-3">
                        <div
                            class="w-10 h-10 rounded-lg bg-gray-100 flex items-center justify-center"
                        >
                            <MessageSquare size={18} class="text-gray-500" />
                        </div>
                        <div>
                            <div
                                class="font-medium text-gray-800 group-hover:text-blue-600 transition-colors"
                            >
                                {conv.title || "未命名对话"}
                            </div>
                            <div class="text-xs text-gray-400 mt-0.5">
                                {formatTime(conv.created_at)}
                                <span class="mx-1">·</span>
                                {getSourceLabel(conv.source)}
                            </div>
                        </div>
                    </div>
                    <div class="flex items-center gap-1">
                        <button
                            class="p-1.5 rounded opacity-0 group-hover:opacity-100 hover:bg-red-50 text-gray-400 hover:text-red-500"
                            title="删除"
                            on:click={(e) => deleteConversation(conv.id, e)}
                        >
                            <Trash2 size={16} />
                        </button>
                        <span
                            class="text-gray-300 group-hover:text-blue-500 transition-colors"
                            >→</span
                        >
                    </div>
                </button>
            {/each}
        </div>
    {/if}
</div>
