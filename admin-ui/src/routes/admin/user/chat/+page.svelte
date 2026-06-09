<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { page } from "$app/stores";
    import { goto } from "$app/navigation";
    import { Send, User, Bot, Loader2, ArrowLeft } from "lucide-svelte";

    interface Message {
        role: "user" | "assistant";
        content: string;
    }

    let messages: Message[] = [];
    let input = "";
    let sending = false;
    let loading = true;

    let conversationId = "";

    onMount(async () => {
        // 从 URL 参数获取会话 ID
        const id = $page.url.searchParams.get("id");
        if (id) {
            conversationId = id;
            await loadMessages(id);
        } else {
            loading = false;
        }
    });

    async function loadMessages(convId: string) {
        loading = true;
        try {
            const data = await api.get<any>(`/user/conversations/${convId}`);
            if (data && data.messages) {
                messages = data.messages.map((m: any) => ({
                    role: m.role,
                    content: m.content,
                }));
            } else {
                messages = [];
            }
        } catch (e: any) {
            toast.error(e.message || "加载对话失败");
        } finally {
            loading = false;
        }
    }

    async function handleSend() {
        const content = input.trim();
        if (!content || sending) return;

        messages = [...messages, { role: "user", content }];
        input = "";
        sending = true;

        try {
            const body: Record<string, any> = {
                message: content,
            };
            if (conversationId) {
                body.conversation_id = conversationId;
            }

            const data = await api.post<any>("/user/chat", body);

            if (data.conversation_id && !conversationId) {
                conversationId = data.conversation_id;
            }

            const reply = data.reply || data.message || data.response || "";
            messages = [...messages, { role: "assistant", content: reply }];
        } catch (e: any) {
            toast.error(e.message || "发送失败");
            messages = [
                ...messages,
                { role: "assistant", content: "抱歉，消息发送失败，请重试。" },
            ];
        } finally {
            sending = false;
        }
    }

    function handleKeydown(e: KeyboardEvent) {
        if (e.key === "Enter" && !e.shiftKey) {
            e.preventDefault();
            handleSend();
        }
    }
</script>

<svelte:head><title>对话 - OpenTether</title></svelte:head>

<div class="max-w-3xl mx-auto">
    <div class="flex items-center gap-3 mb-6">
        <button
            class="p-2 text-slate-400 hover:text-slate-600 hover:bg-slate-100 rounded-lg transition-colors"
            on:click={() => goto("/admin/user")}
        >
            <ArrowLeft size={18} />
        </button>
        <h2 class="text-xl font-bold text-slate-800">
            {conversationId ? "继续对话" : "新建对话"}
        </h2>
    </div>

    <div class="card min-h-[500px] flex flex-col">
        <div
            class="flex-1 overflow-y-auto space-y-4 min-h-[350px] max-h-[500px] mb-4"
        >
            {#if loading}
                <div class="flex items-center justify-center h-full">
                    <Loader2 size={24} class="animate-spin text-slate-400" />
                </div>
            {:else if messages.length === 0}
                <div
                    class="flex items-center justify-center h-full text-slate-400"
                >
                    <div class="text-center">
                        <div class="text-5xl mb-4">🤖</div>
                        <p class="text-lg font-medium text-slate-600">
                            开始新的对话
                        </p>
                        <p class="text-sm text-slate-400 mt-1">
                            输入你的问题开始
                        </p>
                    </div>
                </div>
            {:else}
                {#each messages as msg}
                    <div
                        class="flex gap-3 {msg.role === 'user'
                            ? 'justify-end'
                            : ''}"
                    >
                        {#if msg.role === "assistant"}
                            <div
                                class="w-8 h-8 rounded-lg bg-primary-100 flex items-center justify-center flex-shrink-0"
                            >
                                <Bot size={16} class="text-primary-600" />
                            </div>
                        {/if}
                        <div
                            class="max-w-[80%] p-3 rounded-xl text-sm {msg.role ===
                            'user'
                                ? 'bg-primary-600 text-white rounded-br-sm'
                                : 'bg-slate-100 text-slate-700 rounded-bl-sm'}"
                        >
                            {msg.content}
                        </div>
                        {#if msg.role === "user"}
                            <div
                                class="w-8 h-8 rounded-lg bg-slate-200 flex items-center justify-center flex-shrink-0"
                            >
                                <User size={16} class="text-slate-600" />
                            </div>
                        {/if}
                    </div>
                {/each}
            {/if}
        </div>

        <div class="flex gap-3 pt-4 border-t border-slate-100">
            <textarea
                bind:value={input}
                placeholder="输入你的问题... (Enter 发送, Shift+Enter 换行)"
                on:keydown={handleKeydown}
                rows={2}
                disabled={sending}
                class="flex-1 px-4 py-2.5 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200 resize-none disabled:bg-slate-50"
            />
            <button
                class="px-5 py-2.5 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 transition-colors disabled:opacity-50 flex items-center gap-1.5 self-end"
                on:click={handleSend}
                disabled={sending || !input.trim()}
            >
                {#if sending}
                    <Loader2 size={16} class="animate-spin" />
                {:else}
                    <Send size={16} />
                {/if}
                <span class="hidden sm:inline">发送</span>
            </button>
        </div>
    </div>
</div>
