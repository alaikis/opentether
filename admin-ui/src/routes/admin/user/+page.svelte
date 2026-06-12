<script lang="ts">
    import { onMount, tick } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { page } from "$app/stores";
    import { goto } from "$app/navigation";
    import { marked } from "marked";
    import {
        Send,
        Bot,
        User,
        Loader2,
        Plus,
        Trash2,
        Square,
        Copy,
        RotateCcw,
        Zap,
        ChevronDown,
        PanelLeftClose,
        PanelLeft,
        Search,
        AlertCircle,
        MessageSquare,
        Sparkles,
        Database,
        CheckCircle2,
    } from "lucide-svelte";

    marked.setOptions({
        gfm: true,
        breaks: true,
    });

    // ── Types ──────────────────────────────────────
    interface StreamEvent {
        type: "event";
        content: string;
    }

    interface MetricResult {
        label: string;
        value: string;
        unit: string;
    }

    interface Message {
        id?: string;
        role: "user" | "assistant" | "system";
        content: string;
        thinking?: string;
        skill_used?: string;
        error?: string;
        events?: StreamEvent[];
        tool_calls?: { name: string; result: string }[];
    }

    interface Conversation {
        id: string;
        title: string;
        source: string;
        created_at: string;
        updated_at: string;
    }

    interface Skill {
        id: string;
        name: string;
        description: string;
        enabled: boolean;
    }

    // ── State ──────────────────────────────────────
    let conversations: Conversation[] = [];
    let activeConvId = "";
    let messages: Message[] = [];
    let input = "";
    let streaming = false;
    let abortCtrl: AbortController | null = null;
    let sidebarOpen = true;
    let convSearch = "";

    let skills: Skill[] = [];
    let selectedSkillId = "";
    let showSkillDropdown = false;
    let skillSearch = "";

    let messagesEnd: HTMLElement;
    let loadingConversations = true;
    let loadingMessages = false;
    let pageError = "";

    // ── Computed ───────────────────────────────────
    let groupedConversations: { label: string; items: Conversation[] }[] = [];
    $: {
        let filtered = conversations.filter(
            (c) =>
                !convSearch ||
                c.title.toLowerCase().includes(convSearch.toLowerCase()),
        );
        let today: Conversation[] = [];
        let week: Conversation[] = [];
        let older: Conversation[] = [];
        let now = Date.now();
        for (let c of filtered) {
            let d = new Date(c.updated_at).getTime();
            if (now - d < 86400000) today.push(c);
            else if (now - d < 604800000) week.push(c);
            else older.push(c);
        }
        groupedConversations = [
            { label: "今天", items: today },
            { label: "本周", items: week },
            { label: "更早", items: older },
        ].filter((g) => g.items.length > 0);
    }

    let filteredSkills: Skill[] = [];
    $: filteredSkills = skills.filter(
        (s) =>
            !skillSearch ||
            s.name.toLowerCase().includes(skillSearch.toLowerCase()),
    );

    // ── Init ───────────────────────────────────────
    onMount(async () => {
        await tick();

        try {
            await Promise.all([loadConversations(), loadSkills()]);
            // Check for ?conv=id query param (linked from conversations page)
            const convId = $page.url.searchParams.get("conv");
            if (convId && conversations.some((c) => c.id === convId)) {
                await selectConversation(convId);
            } else if (conversations.length > 0) {
                await selectConversation(conversations[0].id);
            }
        } catch {
            pageError = "加载失败，请刷新重试";
        } finally {
            loadingConversations = false;
        }
    });

    // ── Conversations ──────────────────────────────
    async function loadConversations() {
        try {
            let d = await api.get<Conversation[]>(
                "/user/conversations?source=web",
            );
            conversations = Array.isArray(d) ? d : [];
        } catch {
            conversations = [];
        }
    }

    async function selectConversation(id: string) {
        if (id === activeConvId) return;
        activeConvId = id;
        messages = [];
        loadingMessages = true;
        try {
            let data: any = await api.get(`/user/conversations/${id}`);
            if (data?.messages && Array.isArray(data.messages))
                messages = data.messages.map((m: any) => ({
                    id: m.id,
                    role: m.role,
                    content: m.content,
                    skill_used: m.skill_used,
                }));
        } catch {
            toast.error("加载对话失败");
        } finally {
            loadingMessages = false;
        }
        scrollToBottom();
    }

    function newConversation() {
        activeConvId = "";
        messages = [];
        input = "";
        selectedSkillId = "";
        loadingMessages = false;
    }

    async function deleteConversation(id: string, e: Event) {
        e.stopPropagation();
        if (!confirm("确认删除此对话？")) return;
        try {
            await api.delete(`/user/conversations/${id}`);
            conversations = conversations.filter((c) => c.id !== id);
            if (activeConvId === id) newConversation();
            toast.success("已删除");
        } catch (e: any) {
            toast.error(e.message || "删除失败");
        }
    }

    // ── Skills ─────────────────────────────────────
    async function loadSkills() {
        try {
            let d: any = await api.get("/admin/skills");
            skills = (Array.isArray(d) ? d : [])
                .filter((s: Skill) => s.enabled)
                .slice(0, 20);
        } catch {
            skills = [];
        }
    }

    function selectSkill(skillId: string) {
        selectedSkillId = skillId === selectedSkillId ? "" : skillId;
        showSkillDropdown = false;
        skillSearch = "";
    }

    // ── Send / Stream ──────────────────────────────
    async function handleSend() {
        let content = input.trim();
        if (!content || streaming) return;
        messages = [...messages, { role: "user", content }];
        input = "";
        await tick();
        scrollToBottom();
        streaming = true;
        abortCtrl = new AbortController();
        let assistantMsg: Message = {
            role: "assistant",
            content: "",
            thinking: "正在思考...",
        };
        messages = [...messages, assistantMsg];
        let msgIdx = messages.length - 1;

        try {
            let body: Record<string, any> = { message: content };
            if (activeConvId) body.conversation_id = activeConvId;
            if (selectedSkillId) body.skill_id = selectedSkillId;

            let res = await fetch("/api/v1/user/chat/stream", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `Bearer ${localStorage.getItem("token") || ""}`,
                },
                body: JSON.stringify(body),
                signal: abortCtrl.signal,
            });
            if (!res.ok) {
                let err = await res.json().catch(() => ({}));
                throw new Error(err.error || `请求失败 (${res.status})`);
            }

            let reader = res.body?.getReader();
            if (!reader) throw new Error("无法读取响应流");

            let decoder = new TextDecoder();
            let buffer = "";
            let firstChunk = true;

            while (true) {
                let { done, value } = await reader.read();
                if (done) break;
                buffer += decoder.decode(value, { stream: true });
                let lines = buffer.split("\n");
                buffer = lines.pop() || "";
                for (let line of lines) {
                    if (line.startsWith("data: ")) {
                        let data = line.slice(6);
                        if (data === "[DONE]") break;
                        if (firstChunk) {
                            messages[msgIdx].thinking = undefined;
                            firstChunk = false;
                        }

                        const event = parseStreamEvent(data);
                        if (event) {
                            // 检查是否为技能元信息
                            if (event.content.startsWith("__SKILL__")) {
                                const skillVal = event.content.slice(9);
                                // 累积多个 skills（逗号分隔），避免覆盖
                                if (messages[msgIdx].skill_used) {
                                    messages[msgIdx].skill_used +=
                                        "," + skillVal;
                                } else {
                                    messages[msgIdx].skill_used = skillVal;
                                }
                            } else {
                                messages[msgIdx].events = [
                                    ...(messages[msgIdx].events || []),
                                    event,
                                ];
                            }
                        } else {
                            messages[msgIdx].content += data;
                        }
                        messages = messages;
                        scrollToBottom();
                    }
                }
            }
        } catch (e: any) {
            if (e.name !== "AbortError") {
                messages[msgIdx].content =
                    messages[msgIdx].content || "抱歉，请求失败，请重试。";
                messages[msgIdx].error = e.message;
                toast.error(e.message || "发送失败");
            }
        } finally {
            streaming = false;
            abortCtrl = null;
            await loadConversations();
            if (!activeConvId && conversations.length > 0) {
                activeConvId = conversations[0].id;
            }
        }
    }

    function stopStreaming() {
        abortCtrl?.abort();
    }

    async function regenerate() {
        if (messages.length < 2) return;
        let lastUser = [...messages].reverse().find((m) => m.role === "user");
        if (!lastUser) return;
        messages = messages.slice(0, -1);
        input = lastUser.content;
        await tick();
        handleSend();
    }

    function handleKeydown(e: KeyboardEvent) {
        if (e.key === "Enter" && !e.shiftKey) {
            e.preventDefault();
            handleSend();
        }
    }

    // ── Helpers ────────────────────────────────────
    async function copyMessage(content: string) {
        try {
            await navigator.clipboard.writeText(content);
            toast.success("已复制");
        } catch {
            toast.error("复制失败");
        }
    }

    function parseStreamEvent(data: string): StreamEvent | null {
        try {
            const parsed = JSON.parse(data);
            if (
                parsed?.type === "event" &&
                typeof parsed.content === "string"
            ) {
                return parsed as StreamEvent;
            }
            // 元信息事件（skill_used 标签等）
            if (
                parsed?.type === "meta" &&
                typeof parsed.skill_used === "string"
            ) {
                return {
                    type: "event",
                    content: "__SKILL__" + parsed.skill_used,
                } as StreamEvent;
            }
        } catch {
            // 普通文本 token，不是 JSON 事件
        }
        return null;
    }

    function renderMarkdown(text: string): string {
        if (!text) return "";
        return marked.parse(text, { async: false }) as string;
    }

    function getEventKind(
        content: string,
    ): "thinking" | "tool" | "sql" | "topic" | "warning" | "info" {
        if (content.startsWith("💭")) return "thinking";
        if (content.startsWith("🔧")) return "tool";
        if (content.startsWith("🧭")) return "topic";
        if (
            content.startsWith("[SQL]") ||
            content.toLowerCase().includes("select ")
        )
            return "sql";
        if (content.startsWith("⚠️")) return "warning";
        return "info";
    }

    function eventLabel(content: string) {
        const kind = getEventKind(content);
        if (kind === "thinking") return "思考";
        if (kind === "tool") return "工具";
        if (kind === "sql") return "SQL";
        if (kind === "topic") return "话题";
        if (kind === "warning") return "提醒";
        return "过程";
    }

    function cleanEventContent(content: string) {
        return content
            .replace(/^\u{1F4AD}\s*正在思考:\s*/, "")
            .replace(/^\u{1F527}\s*正在调用:\s*/, "")
            .replace(/^\u26A0\uFE0F\s*/, "")
            .replace(/^\u{1F9ED}\s*话题路由:\s*/, "")
            .trim();
    }

    function friendlySkillLabel(raw: string): {
        label: string;
        type: string;
        color: string;
    } {
        const m: Record<
            string,
            { label: string; type: string; color: string }
        > = {
            fast_local: {
                label: "本地应答",
                type: "快捷",
                color: "bg-slate-50 text-slate-600 border-slate-200",
            },
            fast_chat: {
                label: "快捷问答",
                type: "快速",
                color: "bg-blue-50 text-blue-700 border-blue-100",
            },
            fast_text2sql_template: {
                label: "SQL模板",
                type: "快速",
                color: "bg-emerald-50 text-emerald-700 border-emerald-100",
            },
            fast_text2sql_approved_template: {
                label: "已审批SQL",
                type: "快速",
                color: "bg-emerald-50 text-emerald-700 border-emerald-100",
            },
            skill_text2sql: {
                label: "数据查询",
                type: "技能",
                color: "bg-emerald-50 text-emerald-700 border-emerald-100",
            },
            skill_chat: {
                label: "通用对话",
                type: "技能",
                color: "bg-blue-50 text-blue-700 border-blue-100",
            },
            agent_loop: {
                label: "智能推理",
                type: "引擎",
                color: "bg-rose-50 text-rose-700 border-rose-100",
            },
        };
        if (raw.startsWith("experience:"))
            return {
                label: "经验复用",
                type: "经验",
                color: "bg-violet-50 text-violet-700 border-violet-100",
            };
        if (raw.startsWith("fast_"))
            return {
                label: raw.slice(5),
                type: "快速",
                color: "bg-cyan-50 text-cyan-700 border-cyan-100",
            };
        if (raw.startsWith("skill_"))
            return {
                label: raw.slice(6),
                type: "技能",
                color: "bg-indigo-50 text-indigo-700 border-indigo-100",
            };
        if (m[raw]) return m[raw];
        return {
            label: raw,
            type: "",
            color: "bg-slate-50 text-slate-600 border-slate-200",
        };
    }

    function extractMetricResult(text: string): MetricResult | null {
        const normalized = text.trim().replace(/\s+/g, " ");
        const patterns = [
            /(.+?)(?:为|是|共|总计|一共)\s*([\d,]+(?:\.\d+)?)\s*(条|个|笔|单|人|元|万元|%)/,
            /(?:查询完成，)?共找到\s*([\d,]+(?:\.\d+)?)\s*(条|个|笔|单|人)?/,
        ];

        const direct = normalized.match(patterns[0]);
        if (direct) {
            return {
                label: direct[1].replace(/[，。:：]$/g, "").trim(),
                value: direct[2],
                unit: direct[3] || "",
            };
        }

        const count = normalized.match(patterns[1]);
        if (count) {
            return {
                label: "查询结果",
                value: count[1],
                unit: count[2] || "条",
            };
        }

        return null;
    }

    function isConciseMetricAnswer(text: string) {
        return !!extractMetricResult(text) && text.trim().length <= 80;
    }

    function hasProcessEvents(msg: Message) {
        return !!msg.events?.length;
    }

    function scrollToBottom() {
        messagesEnd?.scrollIntoView({ behavior: "smooth" });
    }

    function formatTime(t: string) {
        if (!t) return "";
        return new Date(t).toLocaleString("zh-CN", {
            month: "2-digit",
            day: "2-digit",
            hour: "2-digit",
            minute: "2-digit",
        });
    }

    function getSkillName(id: string) {
        return skills.find((s) => s.id === id)?.name || "";
    }

    function autoResize(e: Event) {
        let el = e.target as HTMLTextAreaElement;
        el.style.height = "auto";
        el.style.height = Math.min(el.scrollHeight, 128) + "px";
    }

    function toggleSidebar() {
        sidebarOpen = !sidebarOpen;
    }

    function isToolOutput(content: string): boolean {
        return (
            content.startsWith("[查询结果]") ||
            content.startsWith("[员工查询]") ||
            content.startsWith("[工具调用]") ||
            content.includes("SQL:") ||
            content.includes("执行时间:")
        );
    }

    function getConvTitle(conv: Conversation) {
        return conv.title || "新对话";
    }

    function currentConvTitle() {
        if (activeConvId) {
            return (
                conversations.find((c) => c.id === activeConvId)?.title ||
                "对话"
            );
        }
        return "新建对话";
    }
</script>

<svelte:head><title>AI 对话 - OpenTether</title></svelte:head>

<div class="-m-6 h-[calc(100vh-4rem)] flex bg-white overflow-hidden">
    <!-- ├─ Left panel: Conversation list ────────── -->
    {#if sidebarOpen}
        <div
            class="w-[280px] border-r border-slate-200 flex flex-col bg-slate-50 shrink-0"
        >
            <!-- Top actions -->
            <div class="p-3 border-b border-slate-200 space-y-2.5">
                <button
                    class="w-full flex items-center justify-center gap-2 px-3 py-2.5 bg-primary-600 text-white rounded-xl text-sm font-medium hover:bg-primary-700 transition-colors shadow-sm"
                    on:click={newConversation}
                >
                    <Plus size={16} /> 新对话
                </button>
                <div class="relative">
                    <Search
                        size={14}
                        class="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none"
                    />
                    <input
                        type="text"
                        placeholder="搜索对话..."
                        bind:value={convSearch}
                        class="w-full pl-8 pr-3 py-2 border border-slate-200 rounded-lg text-xs focus:outline-none focus:ring-2 focus:ring-primary-200 bg-white placeholder:text-slate-400"
                    />
                </div>
            </div>

            <!-- Conversation list -->
            <div class="flex-1 overflow-y-auto">
                {#if loadingConversations}
                    <div class="flex justify-center py-12">
                        <Loader2
                            size={22}
                            class="animate-spin text-slate-300"
                        />
                    </div>
                {:else if conversations.length === 0}
                    <div class="text-center py-12 text-slate-400 px-4">
                        <MessageSquare
                            size={32}
                            class="mx-auto mb-2 opacity-40"
                        />
                        <p class="text-sm">暂无对话</p>
                        <p class="text-xs mt-1">点击上方按钮开始新对话</p>
                    </div>
                {:else}
                    {#each groupedConversations as group}
                        <div class="px-3 pt-3 pb-1">
                            <div
                                class="text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1.5 px-1"
                            >
                                {group.label}
                            </div>
                            {#each group.items as conv (conv.id)}
                                <button
                                    class="w-full text-left px-2.5 py-2.5 rounded-lg mb-0.5 hover:bg-slate-100 transition-colors group/conv {conv.id ===
                                    activeConvId
                                        ? 'bg-white ring-1 ring-primary-200 shadow-sm'
                                        : ''}"
                                    on:click={() => selectConversation(conv.id)}
                                >
                                    <div
                                        class="flex items-center justify-between gap-1"
                                    >
                                        <div class="flex-1 min-w-0">
                                            <div
                                                class="text-sm font-medium text-slate-700 truncate"
                                            >
                                                {getConvTitle(conv)}
                                            </div>
                                            <div
                                                class="text-xs text-slate-400 mt-0.5"
                                            >
                                                {formatTime(conv.updated_at)}
                                            </div>
                                        </div>
                                        <button
                                            class="p-0.5 rounded opacity-0 group-hover/conv:opacity-100 hover:bg-red-100 text-slate-300 hover:text-red-500 shrink-0 transition-opacity"
                                            title="删除"
                                            on:click={(e) =>
                                                deleteConversation(conv.id, e)}
                                        >
                                            <Trash2 size={13} />
                                        </button>
                                    </div>
                                </button>
                            {/each}
                        </div>
                    {/each}
                {/if}
            </div>

            <!-- Footer link -->
            <div class="p-3 border-t border-slate-200">
                <a
                    href="/admin/user/conversations"
                    class="flex items-center justify-center gap-1.5 text-xs text-slate-400 hover:text-primary-600 transition-colors py-1.5"
                >
                    <MessageSquare size={13} />
                    查看全部
                </a>
            </div>
        </div>
    {/if}

    <!-- ├─ Right panel: Chat area ───────────────── -->
    <div class="flex-1 flex flex-col min-w-0 bg-white">
        <!-- Top bar -->
        <div
            class="px-4 py-2.5 border-b border-slate-200 bg-white flex items-center gap-3 shrink-0"
        >
            <button
                class="p-1.5 rounded-lg hover:bg-slate-100 text-slate-400 transition-colors"
                title={sidebarOpen ? "收起侧边栏" : "展开侧边栏"}
                on:click={toggleSidebar}
            >
                {#if sidebarOpen}
                    <PanelLeftClose size={18} />
                {:else}
                    <PanelLeft size={18} />
                {/if}
            </button>

            <div class="flex-1 font-medium text-sm text-slate-700 truncate">
                {currentConvTitle()}
            </div>

            <!-- Skill selector -->
            {#if skills.length > 0}
                <div class="relative">
                    <button
                        class="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full border transition-colors {selectedSkillId
                            ? 'border-primary-300 bg-primary-50 text-primary-700'
                            : 'border-slate-200 text-slate-500 hover:border-slate-300'}"
                        on:click={() =>
                            (showSkillDropdown = !showSkillDropdown)}
                    >
                        <Zap size={12} />
                        {selectedSkillId
                            ? getSkillName(selectedSkillId)
                            : "Skill"}
                        <ChevronDown size={12} />
                    </button>

                    {#if showSkillDropdown}
                        <div
                            class="absolute right-0 top-full mt-1 w-64 bg-white border border-slate-200 rounded-xl shadow-xl z-20 max-h-80 overflow-hidden"
                        >
                            <div class="p-2 border-b border-slate-100">
                                <input
                                    type="text"
                                    placeholder="搜索 Skill..."
                                    bind:value={skillSearch}
                                    class="w-full px-2 py-1.5 border border-slate-200 rounded-lg text-xs focus:outline-none focus:ring-1 focus:ring-primary-200"
                                />
                            </div>
                            <div class="overflow-y-auto max-h-60">
                                <button
                                    class="w-full text-left px-3 py-2 text-xs hover:bg-slate-50 {!selectedSkillId
                                        ? 'bg-primary-50 text-primary-700'
                                        : 'text-slate-600'}"
                                    on:click={() => selectSkill("")}
                                >
                                    🤖 自动选择
                                </button>
                                {#each filteredSkills as sk}
                                    <button
                                        class="w-full text-left px-3 py-2 text-xs hover:bg-slate-50 transition-colors {selectedSkillId ===
                                        sk.id
                                            ? 'bg-primary-50 text-primary-700'
                                            : 'text-slate-600'}"
                                        on:click={() => selectSkill(sk.id)}
                                    >
                                        <div class="font-medium">{sk.name}</div>
                                        {#if sk.description}
                                            <div
                                                class="text-slate-400 truncate text-[11px] mt-0.5"
                                            >
                                                {sk.description}
                                            </div>
                                        {/if}
                                    </button>
                                {/each}
                            </div>
                        </div>
                    {/if}
                </div>
            {/if}
        </div>

        <!-- Messages area -->
        <div
            class="flex-1 overflow-y-auto overflow-x-hidden px-4 py-4 scroll-smooth"
        >
            {#if pageError}
                <div class="flex items-center justify-center h-full">
                    <div class="text-center max-w-sm">
                        <AlertCircle
                            size={48}
                            class="mx-auto text-red-300 mb-4"
                        />
                        <p class="text-red-500 font-medium">{pageError}</p>
                        <button
                            class="mt-4 px-4 py-2 bg-primary-600 text-white rounded-lg text-sm hover:bg-primary-700 transition-colors"
                            on:click={() => {
                                pageError = "";
                                window.location.reload();
                            }}
                        >
                            刷新重试
                        </button>
                    </div>
                </div>
            {:else if loadingMessages}
                <div class="flex items-center justify-center h-full">
                    <div class="text-center text-slate-400">
                        <Loader2 size={24} class="animate-spin mx-auto mb-2" />
                        <p class="text-sm">加载对话中...</p>
                    </div>
                </div>
            {:else if messages.length === 0}
                <div
                    class="flex items-center justify-center h-full text-slate-400"
                >
                    <div class="text-center max-w-md">
                        <div class="text-6xl mb-5">
                            <Sparkles
                                size={64}
                                class="mx-auto text-primary-400"
                            />
                        </div>
                        <h2 class="text-xl font-semibold text-slate-700 mb-2">
                            OpenTether AI 助手
                        </h2>
                        <p class="text-sm text-slate-400 mb-6">
                            企业级 AI Agent，支持数据查询、报表生成、智能分析
                        </p>
                        <div class="flex flex-wrap justify-center gap-2">
                            {#each ["查订单数据", "员工销售分析", "生成 PDF 报表", "Excel 数据处理"] as hint}
                                <button
                                    class="px-4 py-2 text-xs border border-slate-200 rounded-full text-slate-500 hover:border-primary-300 hover:text-primary-600 hover:bg-primary-50 transition-colors"
                                    on:click={() => {
                                        input = hint;
                                        handleSend();
                                    }}
                                >
                                    {hint}
                                </button>
                            {/each}
                        </div>
                    </div>
                </div>
            {:else}
                {#each messages as msg, i}
                    {#if isToolOutput(msg.content)}
                        <!-- Tool output card -->
                        <div
                            class="flex justify-center mb-4 max-w-4xl mx-auto w-full min-w-0"
                        >
                            <div
                                class="bg-slate-50 border border-slate-200 rounded-xl px-4 py-2.5 max-w-2xl w-full min-w-0 overflow-hidden"
                            >
                                <div
                                    class="flex items-center gap-2 text-xs text-slate-500 mb-1.5"
                                >
                                    <Zap size={12} class="text-primary-500" />
                                    <span class="font-medium">工具执行结果</span
                                    >
                                </div>
                                <pre
                                    class="text-xs text-slate-700 whitespace-pre-wrap break-words overflow-x-auto font-mono">{msg.content}</pre>
                            </div>
                        </div>
                    {:else}
                        <!-- Regular message -->
                        <div
                            class="flex gap-3 mb-6 max-w-4xl mx-auto w-full min-w-0 {msg.role ===
                            'user'
                                ? 'justify-end'
                                : ''}"
                        >
                            {#if msg.role === "assistant"}
                                <div
                                    class="w-8 h-8 rounded-xl bg-gradient-to-br from-primary-400 to-primary-600 flex items-center justify-center shrink-0 shadow-sm"
                                >
                                    <Bot size={15} class="text-white" />
                                </div>
                            {/if}

                            <div
                                class="min-w-0 max-w-full md:max-w-[75%] group/msg overflow-hidden"
                            >
                                {#if msg.role === "assistant"}
                                    {#if streaming && i === messages.length - 1 && msg.thinking}
                                        <div
                                            class="flex items-center gap-2 text-xs text-slate-400 mb-1 ml-1"
                                        >
                                            <Loader2
                                                size={12}
                                                class="animate-spin"
                                            />
                                            {msg.thinking}
                                        </div>
                                    {/if}

                                    {#if hasProcessEvents(msg)}
                                        <details
                                            class="mb-2 rounded-xl border border-slate-200 bg-white/80 shadow-sm overflow-hidden process-details max-w-full"
                                            open={streaming &&
                                                i === messages.length - 1}
                                        >
                                            <summary
                                                class="cursor-pointer select-none px-3 py-2 text-xs font-medium text-slate-600 bg-slate-50 hover:bg-slate-100 flex items-center gap-2"
                                            >
                                                <Zap
                                                    size={13}
                                                    class="text-primary-500"
                                                />
                                                执行过程
                                                <span
                                                    class="ml-auto text-[11px] text-slate-400"
                                                    >{msg.events?.length || 0} 步</span
                                                >
                                            </summary>
                                            <div class="p-2 space-y-2">
                                                {#each msg.events || [] as evt}
                                                    <div
                                                        class="rounded-lg border px-3 py-2 text-xs {getEventKind(
                                                            evt.content,
                                                        ) === 'sql'
                                                            ? 'border-blue-100 bg-blue-50/70'
                                                            : getEventKind(
                                                                    evt.content,
                                                                ) === 'warning'
                                                              ? 'border-amber-100 bg-amber-50/70'
                                                              : 'border-slate-100 bg-slate-50/70'}"
                                                    >
                                                        <div
                                                            class="flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wide text-slate-500 mb-1"
                                                        >
                                                            {#if getEventKind(evt.content) === "sql"}
                                                                <Database
                                                                    size={12}
                                                                />
                                                            {:else if getEventKind(evt.content) === "tool"}
                                                                <Zap
                                                                    size={12}
                                                                />
                                                            {:else if getEventKind(evt.content) === "warning"}
                                                                <AlertCircle
                                                                    size={12}
                                                                />
                                                            {:else}
                                                                <CheckCircle2
                                                                    size={12}
                                                                />
                                                            {/if}
                                                            {eventLabel(
                                                                evt.content,
                                                            )}
                                                        </div>
                                                        {#if getEventKind(evt.content) === "sql"}
                                                            <pre
                                                                class="text-[11px] leading-relaxed overflow-x-auto whitespace-pre-wrap break-words font-mono text-blue-900">{cleanEventContent(
                                                                    evt.content,
                                                                ).replace(
                                                                    /^\[SQL\]\s*/,
                                                                    "",
                                                                )}</pre>
                                                            >
                                                        {:else}
                                                            <div
                                                                class="text-slate-600 leading-relaxed whitespace-pre-wrap break-words"
                                                            >
                                                                {cleanEventContent(
                                                                    evt.content,
                                                                )}
                                                            </div>
                                                        {/if}
                                                    </div>
                                                {/each}
                                            </div>
                                        </details>
                                    {/if}

                                    {#if msg.content}
                                        {#if isConciseMetricAnswer(msg.content)}
                                            {@const metric =
                                                extractMetricResult(
                                                    msg.content,
                                                )}
                                            <div
                                                class="rounded-2xl rounded-bl-md border border-primary-100 bg-gradient-to-br from-primary-50 to-white px-4 py-3 shadow-sm max-w-full overflow-hidden"
                                            >
                                                <div
                                                    class="flex items-center gap-2 text-xs font-medium text-primary-700 mb-2"
                                                >
                                                    <CheckCircle2 size={14} />
                                                    查询完成
                                                </div>
                                                <div
                                                    class="text-sm text-slate-500 mb-1"
                                                >
                                                    {metric?.label ||
                                                        "查询结果"}
                                                </div>
                                                <div
                                                    class="flex items-end gap-1.5"
                                                >
                                                    <span
                                                        class="text-4xl font-semibold tracking-tight text-slate-900"
                                                        >{metric?.value}</span
                                                    >
                                                    <span
                                                        class="pb-1 text-sm font-medium text-slate-500"
                                                        >{metric?.unit}</span
                                                    >
                                                </div>
                                                <div
                                                    class="mt-3 text-xs text-slate-400"
                                                >
                                                    原始回答：{msg.content}
                                                </div>
                                            </div>
                                        {:else}
                                            <div
                                                class="rounded-2xl rounded-bl-md bg-slate-50 border border-slate-100 px-4 py-2.5 max-w-full overflow-hidden"
                                            >
                                                <!-- svelte-ignore @html-allow -->
                                                <div
                                                    class="prose prose-sm max-w-none min-w-0 prose-p:my-1.5 prose-pre:my-3 prose-code:text-xs prose-table:text-xs"
                                                >
                                                    {@html renderMarkdown(
                                                        msg.content,
                                                    )}
                                                </div>
                                                {#if streaming && i === messages.length - 1}
                                                    <span
                                                        class="inline-block w-1.5 h-4 bg-primary-500 animate-pulse ml-0.5 align-middle rounded-sm"
                                                    ></span>
                                                {/if}
                                            </div>
                                        {/if}
                                    {/if}
                                {:else}
                                    <div
                                        class="bg-primary-600 text-white rounded-2xl rounded-br-md px-4 py-2.5 shadow-sm max-w-full overflow-hidden"
                                    >
                                        <div
                                            class="text-sm whitespace-pre-wrap break-words leading-relaxed"
                                        >
                                            {msg.content}
                                        </div>
                                    </div>
                                {/if}

                                <!-- Skill 标签 -->
                                {#if msg.role === "assistant" && (msg.skill_used || (msg.tool_calls && msg.tool_calls.length > 0))}
                                    <div
                                        class="flex flex-wrap items-center gap-1.5 mt-1.5 ml-1"
                                    >
                                        {#if msg.skill_used}
                                            {#each msg.skill_used.split(",") as skill}
                                                {@const s = skill.trim()}
                                                {@const info =
                                                    friendlySkillLabel(s)}
                                                {#if s}
                                                    <span
                                                        title={s}
                                                        class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium border {info.color}"
                                                    >
                                                        <Zap size={10} />
                                                        <span>{info.label}</span
                                                        >
                                                        <span
                                                            class="text-[10px] opacity-60"
                                                            >{info.type}</span
                                                        >
                                                    </span>
                                                {/if}
                                            {/each}
                                        {/if}
                                        {#if msg.tool_calls}
                                            {#each msg.tool_calls as tc}
                                                <span
                                                    class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium bg-slate-100 text-slate-600 border border-slate-200"
                                                >
                                                    <Database size={11} />
                                                    {tc.name}
                                                </span>
                                            {/each}
                                        {/if}
                                    </div>
                                {/if}

                                <!-- Action buttons -->
                                {#if msg.role === "assistant" && msg.content && !streaming}
                                    <div
                                        class="flex items-center gap-0.5 mt-1 ml-1 opacity-0 group-hover/msg:opacity-100 transition-opacity"
                                    >
                                        <button
                                            class="p-1 rounded hover:bg-slate-100 text-slate-300 hover:text-slate-500 transition-colors"
                                            title="复制"
                                            on:click={() =>
                                                copyMessage(msg.content)}
                                        >
                                            <Copy size={13} />
                                        </button>
                                        {#if i === messages.length - 1}
                                            <button
                                                class="p-1 rounded hover:bg-slate-100 text-slate-300 hover:text-slate-500 transition-colors"
                                                title="重新生成"
                                                on:click={regenerate}
                                            >
                                                <RotateCcw size={13} />
                                            </button>
                                        {/if}
                                    </div>
                                {/if}

                                {#if msg.error && !streaming}
                                    <div
                                        class="flex items-center gap-1 mt-1 ml-1 text-xs text-red-400"
                                    >
                                        <AlertCircle size={12} />
                                        {msg.error}
                                    </div>
                                {/if}
                            </div>

                            {#if msg.role === "user"}
                                <div
                                    class="w-8 h-8 rounded-xl bg-gradient-to-br from-slate-400 to-slate-500 flex items-center justify-center shrink-0 shadow-sm"
                                >
                                    <User size={15} class="text-white" />
                                </div>
                            {/if}
                        </div>
                    {/if}
                {/each}
            {/if}
            <div bind:this={messagesEnd}></div>
        </div>

        <!-- Input area -->
        <div class="border-t border-slate-200 px-4 py-3 bg-white shrink-0">
            <div class="flex gap-3 items-end max-w-4xl mx-auto">
                <textarea
                    bind:value={input}
                    placeholder="输入你的问题... (Enter 发送, Shift+Enter 换行)"
                    on:keydown={handleKeydown}
                    rows={1}
                    disabled={streaming}
                    class="flex-1 px-4 py-2.5 border border-slate-200 rounded-2xl text-sm focus:outline-none focus:ring-2 focus:ring-primary-200 resize-none disabled:bg-slate-50 max-h-32 placeholder:text-slate-400"
                    on:input={autoResize}
                />
                {#if streaming}
                    <button
                        class="px-4 py-2.5 bg-red-500 text-white rounded-2xl text-sm font-medium hover:bg-red-600 transition-colors flex items-center gap-1.5 shrink-0"
                        on:click={stopStreaming}
                    >
                        <Square size={16} fill="currentColor" /> 停止
                    </button>
                {:else}
                    <button
                        class="px-5 py-2.5 bg-primary-600 text-white rounded-2xl text-sm font-medium hover:bg-primary-700 transition-colors disabled:opacity-50 flex items-center gap-1.5 shrink-0 shadow-sm"
                        on:click={handleSend}
                        disabled={!input.trim()}
                    >
                        <Send size={16} /> 发送
                    </button>
                {/if}
            </div>
            <div class="text-center text-[11px] text-slate-400 mt-2">
                OpenTether AI 助手 · 内容由 AI 生成，请核实重要信息
            </div>
        </div>
    </div>
</div>

<style>
    :global(.prose) {
        line-height: 1.65;
        max-width: 100%;
        overflow-wrap: anywhere;
        word-break: break-word;
    }

    :global(.prose pre) {
        padding: 0.85rem 1rem;
        border-radius: 0.75rem;
        max-width: 100%;
        overflow-x: auto;
        background: #f8fafc;
        color: #334155;
        border: 1px solid #e2e8f0;
        box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.8);
        white-space: pre-wrap;
        word-break: break-word;
    }

    :global(.prose code) {
        font-size: 0.85em;
        border-radius: 0.35rem;
        padding: 0.1rem 0.32rem;
        background: #e2e8f0;
        color: #334155;
    }

    :global(.prose pre code) {
        padding: 0;
        background: transparent;
        color: #334155;
        border-radius: 0;
        white-space: inherit;
        word-break: inherit;
    }

    :global(.prose code::before),
    :global(.prose code::after) {
        content: none;
    }

    :global(.prose table) {
        border-collapse: separate;
        border-spacing: 0;
        width: 100%;
        max-width: 100%;
        display: block;
        font-size: 0.75rem;
        overflow: hidden;
        border: 1px solid #e5e7eb;
        border-radius: 0.65rem;
        background: white;
    }

    :global(.prose thead),
    :global(.prose tbody),
    :global(.prose tr) {
        display: table;
        width: 100%;
        table-layout: auto;
    }

    :global(.prose th),
    :global(.prose td) {
        border-bottom: 1px solid #e5e7eb;
        padding: 0.5rem 0.7rem;
        text-align: left;
        vertical-align: top;
    }

    :global(.prose th) {
        background: #f8fafc;
        font-weight: 600;
        color: #475569;
    }

    :global(.prose tr:last-child td) {
        border-bottom: none;
    }

    :global(.prose blockquote) {
        border-left-color: #cbd5e1;
        color: #64748b;
        background: #f8fafc;
        border-radius: 0.5rem;
        padding: 0.45rem 0.75rem;
    }

    .process-details summary::-webkit-details-marker {
        display: none;
    }
</style>
