<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { auth, isAuthenticated } from "$lib/stores/auth";
    import { Send, User, Bot, Loader2 } from "lucide-svelte";

    interface Message {
        role: "user" | "assistant";
        content: string;
    }

    let messages: Message[] = [];
    let input = "";
    let sending = false;
    let loadingConversations = false;

    let conversationId: string | null = null;

    onMount(async () => {
        // 检查认证状态
        auth.checkAndSync();
        await new Promise((r) => setTimeout(r, 100));

        if (!$isAuthenticated) {
            toast.error("请先登录");
            return;
        }

        // 加载最近的对话
        await loadLatestConversation();
    });

    async function loadLatestConversation() {
        loadingConversations = true;
        try {
            const data: any[] = await api.get("/user/conversations", {
                params: { limit: "1" },
            });
            if (Array.isArray(data) && data.length > 0) {
                conversationId = data[0].id;
                await loadMessages(conversationId);
            }
        } catch {
            // 无对话也正常
        } finally {
            loadingConversations = false;
        }
    }

    async function loadMessages(convId: string) {
        try {
            const data = await api.get(`/user/conversations/${convId}`);
            if (data && (data as any).messages) {
                messages = (data as any).messages.map((m: any) => ({
                    role: m.role,
                    content: m.content,
                }));
            }
        } catch {
            messages = [];
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

            // 保存对话 ID
            if (data.conversation_id) {
                conversationId = data.conversation_id;
            }

            // 添加回复
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

<svelte:head><title>AI 对话 - OpenTether</title></svelte:head>

<div class="max-w-3xl mx-auto">
    <h2 class="text-2xl font-bold text-slate-800 mb-6">AI 对话</h2>

    <div class="card min-h-[500px] flex flex-col">
        <!-- Messages area -->
        <div
            class="flex-1 overflow-y-auto space-y-4 min-h-[350px] max-h-[500px] mb-4"
        >
            {#if loadingConversations}
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
                            你好！我是 OpenTether AI 助手
                        </p>
                        <p class="text-sm text-slate-400 mt-1">
                            输入你的问题开始对话
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

        <!-- Input area -->
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
