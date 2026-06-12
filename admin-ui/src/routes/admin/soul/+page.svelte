<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";

    interface Soul {
        id?: string;
        user_id?: string;
        persona: string;
        human: string;
        preferences: string;
    }
    interface CompanySoul {
        id?: string;
        name: string;
        persona: string;
        brand_tone: string;
        industry: string;
    }
    interface Memory {
        id: string;
        user_id?: string;
        group_id?: string;
        type: string;
        key: string;
        content: string;
        priority: number;
        created_at: string;
    }

    let tab = "soul"; // soul | company | memories
    let saving = false;
    let loading = true;
    const placeholderPrefs = '{"preferred_skills":[],"language":"zh-CN"}';

    // 用户 Soul
    let userSoul: Soul = { persona: "", human: "", preferences: "{}" };
    let soulUserId = "self";
    let users: { id: string; global_user_id: string; name: string }[] = [];

    // 公司 Soul
    let companySoul: CompanySoul = {
        name: "",
        persona: "",
        brand_tone: "",
        industry: "",
    };

    // 记忆
    let memories: Memory[] = [];
    let memUserId = "self";
    let memGroupId = "";

    onMount(async () => {
        await Promise.all([loadUsers(), loadCompanySoul(), loadUserSoul()]);
        loading = false;
    });

    async function loadUsers() {
        try {
            users = await api.get("/admin/users");
        } catch {
            users = [];
        }
    }

    async function loadUserSoul() {
        try {
            const uid = soulUserId === "self" ? "" : soulUserId;
            userSoul = await api.get(`/admin/soul/user/${uid || "self"}`);
        } catch {
            /* use defaults */
        }
    }

    async function saveUserSoul() {
        saving = true;
        try {
            await api.put(`/admin/soul/user/${soulUserId}`, userSoul);
            toast.success("用户 Soul 已保存");
        } catch (e: any) {
            toast.error(e.message || "保存失败");
        } finally {
            saving = false;
        }
    }

    async function loadCompanySoul() {
        try {
            companySoul = (await api.get("/admin/soul/company")) || {
                name: "",
                persona: "",
                brand_tone: "",
                industry: "",
            };
        } catch {
            /* empty */
        }
    }

    async function saveCompanySoul() {
        saving = true;
        try {
            await api.put("/admin/soul/company", companySoul);
            toast.success("公司 Soul 已保存");
        } catch (e: any) {
            toast.error(e.message || "保存失败");
        } finally {
            saving = false;
        }
    }

    async function loadMemories() {
        if (memUserId) {
            const uid = memUserId === "self" ? "" : memUserId;
            memories =
                (await api.get(`/admin/memories/user/${uid || "self"}`)) || [];
        } else if (memGroupId) {
            memories =
                (await api.get(`/admin/memories/group/${memGroupId}`)) || [];
        }
    }

    async function deleteMemory(id: string) {
        if (!confirm("确定删除此记忆？")) return;
        try {
            await api.delete(`/admin/memories/${id}`);
            toast.success("已删除");
            memories = memories.filter((m) => m.id !== id);
        } catch (e: any) {
            toast.error(e.message || "删除失败");
        }
    }

    async function handleSoulUserChange() {
        await loadUserSoul();
    }

    $: if (tab === "memories" && !loading) loadMemories();
</script>

<svelte:head><title>Soul 管理 - OpenTether</title></svelte:head>

<div class="card">
    <h2 class="text-xl font-bold text-slate-800 mb-6">🧠 Soul & 记忆管理</h2>

    <!-- Tabs -->
    <div class="flex gap-1 mb-6 border-b border-gray-200">
        {#each [{ id: "soul", label: "用户 Soul" }, { id: "company", label: "公司 Soul" }, { id: "memories", label: "记忆管理" }] as t}
            <button
                class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {tab ===
                t.id
                    ? 'border-blue-600 text-blue-700'
                    : 'border-transparent text-gray-500 hover:text-gray-700'}"
                on:click={() => (tab = t.id)}>{t.label}</button
            >
        {/each}
    </div>

    {#if loading}
        <div class="text-center py-12 text-gray-400">加载中...</div>
    {:else if tab === "soul"}
        <!-- 用户 Soul -->
        <div class="max-w-2xl">
            <p class="text-sm text-gray-500 mb-4">
                定义 AI
                助手的人格（Persona）和用户画像（Human）。这些会注入到每次对话的
                System Prompt 中。
            </p>

            <div class="mb-4">
                <label class="block text-sm font-medium text-slate-700 mb-1"
                    >目标用户</label
                >
                <select
                    bind:value={soulUserId}
                    on:change={handleSoulUserChange}
                    class="w-64 px-3 py-2 border border-gray-200 rounded-lg text-sm"
                >
                    <option value="self">当前用户</option>
                    {#each users as u}
                        <option value={u.id}
                            >{u.name || u.global_user_id}</option
                        >
                    {/each}
                </select>
            </div>

            <div class="mb-4">
                <label class="block text-sm font-medium text-slate-700 mb-1"
                    >AI 人格 (Persona)</label
                >
                <textarea
                    bind:value={userSoul.persona}
                    rows={3}
                    placeholder="你是一个专业、友好的 AI 助手..."
                    class="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-blue-200 resize-none"
                ></textarea>
            </div>

            <div class="mb-4">
                <label class="block text-sm font-medium text-slate-700 mb-1"
                    >用户描述 (Human)</label
                >
                <textarea
                    bind:value={userSoul.human}
                    rows={3}
                    placeholder="用户是公司销售部门的员工..."
                    class="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-blue-200 resize-none"
                ></textarea>
            </div>

            <div class="mb-4">
                <label class="block text-sm font-medium text-slate-700 mb-1"
                    >偏好设置 (JSON)</label
                >
                <textarea
                    bind:value={userSoul.preferences}
                    rows={3}
                    placeholder={placeholderPrefs}
                    class="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-blue-200 resize-none font-mono"
                ></textarea>
            </div>

            <button
                class="px-6 py-2.5 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 disabled:opacity-50"
                on:click={saveUserSoul}
                disabled={saving}
            >
                {saving ? "保存中..." : "保存用户 Soul"}
            </button>
        </div>
    {:else if tab === "company"}
        <!-- 公司 Soul -->
        <div class="max-w-2xl">
            <p class="text-sm text-gray-500 mb-4">
                定义公司级别的 AI 人格和语调规则，适用于所有用户。
            </p>

            <div class="grid grid-cols-2 gap-4 mb-4">
                <div>
                    <label class="block text-sm font-medium text-slate-700 mb-1"
                        >公司名称</label
                    >
                    <input
                        type="text"
                        bind:value={companySoul.name}
                        placeholder="企业名称"
                        class="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm"
                    />
                </div>
                <div>
                    <label class="block text-sm font-medium text-slate-700 mb-1"
                        >行业</label
                    >
                    <input
                        type="text"
                        bind:value={companySoul.industry}
                        placeholder="如: 新能源、电商"
                        class="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm"
                    />
                </div>
            </div>

            <div class="mb-4">
                <label class="block text-sm font-medium text-slate-700 mb-1"
                    >AI 人格 (Persona)</label
                >
                <textarea
                    bind:value={companySoul.persona}
                    rows={3}
                    placeholder="公司的 AI 品牌人格..."
                    class="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-blue-200 resize-none"
                ></textarea>
            </div>

            <div class="mb-4">
                <label class="block text-sm font-medium text-slate-700 mb-1"
                    >语调规则 (Brand Tone)</label
                >
                <textarea
                    bind:value={companySoul.brand_tone}
                    rows={3}
                    placeholder="必须使用正式、专业的语调；禁止不实承诺；..."
                    class="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-blue-200 resize-none"
                ></textarea>
            </div>

            <button
                class="px-6 py-2.5 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 disabled:opacity-50"
                on:click={saveCompanySoul}
                disabled={saving}
            >
                {saving ? "保存中..." : "保存公司 Soul"}
            </button>
        </div>
    {:else if tab === "memories"}
        <!-- 记忆管理 -->
        <div>
            <p class="text-sm text-gray-500 mb-4">
                查看和管理用户和组的长期记忆。
            </p>

            <div class="flex gap-4 mb-4">
                <div>
                    <label class="block text-xs font-medium text-slate-600 mb-1"
                        >用户</label
                    >
                    <select
                        bind:value={memUserId}
                        class="w-48 px-3 py-2 border border-gray-200 rounded-lg text-sm"
                        on:change={() => (memGroupId = "")}
                    >
                        <option value="">-- 全部 --</option>
                        <option value="self">当前用户</option>
                        {#each users as u}
                            <option value={u.id}
                                >{u.name || u.global_user_id}</option
                            >
                        {/each}
                    </select>
                </div>
                <button
                    class="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-700 self-end"
                    on:click={loadMemories}>查询</button
                >
            </div>

            {#if memories.length === 0}
                <div class="text-center py-8 text-gray-400 text-sm">
                    暂无记忆记录
                </div>
            {:else}
                <div class="space-y-2">
                    {#each memories as mem}
                        <div
                            class="p-3 border border-gray-200 rounded-lg flex items-start justify-between gap-4"
                        >
                            <div class="flex-1 min-w-0">
                                <div class="flex items-center gap-2 mb-1">
                                    <span
                                        class="text-xs font-medium px-2 py-0.5 bg-gray-100 rounded-full"
                                        >{mem.type}</span
                                    >
                                    <span
                                        class="text-sm font-medium text-gray-800"
                                        >{mem.key}</span
                                    >
                                    <span class="text-xs text-gray-400"
                                        >{new Date(
                                            mem.created_at,
                                        ).toLocaleString("zh-CN")}</span
                                    >
                                </div>
                                <p class="text-xs text-gray-600 line-clamp-2">
                                    {mem.content}
                                </p>
                            </div>
                            <button
                                class="p-1 text-red-400 hover:text-red-600 hover:bg-red-50 rounded shrink-0"
                                title="删除"
                                on:click={() => deleteMemory(mem.id)}>🗑️</button
                            >
                        </div>
                    {/each}
                </div>
            {/if}
        </div>
    {/if}
</div>
