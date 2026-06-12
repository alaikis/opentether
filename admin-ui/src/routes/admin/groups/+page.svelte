<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { Plus, Pencil, Trash2, Search, Users } from "lucide-svelte";

    interface UserGroup {
        id: string;
        group_name: string;
        group_code: string;
        external_group_id: string;
        description: string;
        data_access_scope: string;
        created_at: string;
    }

    let groups: UserGroup[] = [];
    let loading = true;
    let error = "";
    let searchQuery = "";

    let showModal = false;
    let editingGroup: UserGroup | null = null;
    let saving = false;

    let formGroupName = "";
    let formGroupCode = "";
    let formExternalGroupID = "";
    let formDescription = "";
    let formDataAccessScope = "self";

    const scopeOptions = [
        { value: "all", label: "全部数据" },
        { value: "self", label: "仅自己" },
        { value: "department", label: "部门数据" },
        { value: "custom", label: "自定义" },
    ];

    $: filteredGroups = groups.filter(
        (g) =>
            !searchQuery ||
            g.group_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
            g.group_code.toLowerCase().includes(searchQuery.toLowerCase()) ||
            (g.description || "")
                .toLowerCase()
                .includes(searchQuery.toLowerCase()),
    );

    onMount(async () => {
        await loadGroups();
    });

    async function loadGroups() {
        loading = true;
        error = "";
        try {
            const data = await api.get<UserGroup[]>("/admin/groups");
            groups = Array.isArray(data) ? data : [];
        } catch (e: any) {
            error = e.message || "加载用户组列表失败";
            groups = [];
        } finally {
            loading = false;
        }
    }

    function openAddModal() {
        editingGroup = null;
        formGroupName = "";
        formGroupCode = "";
        formExternalGroupID = "";
        formDescription = "";
        formDataAccessScope = "self";
        showModal = true;
    }

    function openEditModal(g: UserGroup) {
        editingGroup = g;
        formGroupName = g.group_name;
        formGroupCode = g.group_code;
        formExternalGroupID = g.external_group_id || "";
        formDescription = g.description || "";
        formDataAccessScope = g.data_access_scope || "self";
        showModal = true;
    }

    function closeModal() {
        showModal = false;
        editingGroup = null;
    }

    async function handleSave() {
        if (!formGroupName || !formGroupCode) {
            toast.error("请填写必填字段");
            return;
        }
        saving = true;
        try {
            const body = {
                group_name: formGroupName,
                group_code: formGroupCode,
                external_group_id: formExternalGroupID,
                description: formDescription,
                data_access_scope: formDataAccessScope,
            };

            if (editingGroup) {
                await api.put(`/admin/groups/${editingGroup.id}`, body);
                toast.success("用户组已更新");
            } else {
                await api.post("/admin/groups", body);
                toast.success("用户组已创建");
            }
            closeModal();
            await loadGroups();
        } catch (e: any) {
            toast.error(e.message || "保存失败");
        } finally {
            saving = false;
        }
    }

    async function handleDelete(g: UserGroup) {
        if (!confirm(`确定删除用户组 "${g.group_name}" 吗？`)) return;
        try {
            await api.delete(`/admin/groups/${g.id}`);
            toast.success("用户组已删除");
            await loadGroups();
        } catch (e: any) {
            toast.error(e.message || "删除失败");
        }
    }

    function getScopeLabel(scope: string) {
        const opt = scopeOptions.find((o) => o.value === scope);
        return opt?.label || scope;
    }
</script>

<svelte:head><title>用户组 - OpenTether</title></svelte:head>

<div class="card">
    <div class="flex items-center justify-between mb-6">
        <div>
            <h2 class="text-xl font-bold text-slate-800">用户组管理</h2>
            <p class="text-sm text-slate-500 mt-1">管理用户组和数据访问权限</p>
        </div>
        <button
            class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 transition-colors flex items-center gap-1.5"
            on:click={openAddModal}
        >
            <Plus size={16} />
            创建用户组
        </button>
    </div>

    {#if error}
        <div
            class="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm"
        >
            {error}
            <button class="ml-2 underline" on:click={loadGroups}>重试</button>
        </div>
    {/if}

    <div class="mb-4 relative">
        <Search
            size={16}
            class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
        />
        <input
            type="text"
            placeholder="搜索组名、编码..."
            bind:value={searchQuery}
            class="w-full pl-9 pr-4 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
        />
    </div>

    {#if loading}
        <div class="text-center py-12 text-slate-400">加载中...</div>
    {:else if filteredGroups.length === 0}
        <div class="text-center py-12 space-y-3">
            <div class="text-4xl">👨‍👩‍👧‍👦</div>
            <p class="text-slate-500">
                {searchQuery ? "未找到匹配的用户组" : "暂无用户组"}
            </p>
            {#if !searchQuery}
                <p class="text-sm text-slate-400">点击"创建用户组"开始</p>
            {/if}
        </div>
    {:else}
        <div class="overflow-x-auto">
            <table class="w-full text-sm">
                <thead>
                    <tr class="border-b border-slate-200">
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >组名</th
                        >
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >编码</th
                        >
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >外部组识别号</th
                        >
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >数据权限</th
                        >
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >描述</th
                        >
                        <th
                            class="text-right py-3 px-4 font-medium text-slate-500"
                            >操作</th
                        >
                    </tr>
                </thead>
                <tbody>
                    {#each filteredGroups as g}
                        <tr class="border-b border-slate-100 hover:bg-slate-50">
                            <td
                                class="py-3 px-4 font-medium flex items-center gap-2"
                            >
                                <Users size={14} class="text-slate-400" />
                                {g.group_name}
                            </td>
                            <td class="py-3 px-4">
                                <span
                                    class="px-2 py-0.5 bg-slate-100 text-slate-600 rounded text-xs font-mono"
                                >
                                    {g.group_code}
                                </span>
                            </td>
                            <td
                                class="py-3 px-4 font-mono text-xs text-slate-500"
                                >{g.external_group_id || "-"}</td
                            >
                            <td class="py-3 px-4">
                                <span
                                    class="px-2 py-0.5 bg-indigo-50 text-indigo-700 rounded-full text-xs"
                                >
                                    {getScopeLabel(g.data_access_scope)}
                                </span>
                            </td>
                            <td
                                class="py-3 px-4 text-slate-500 text-xs max-w-[200px] truncate"
                            >
                                {g.description || "-"}
                            </td>
                            <td class="py-3 px-4 text-right">
                                <div
                                    class="flex items-center justify-end gap-1"
                                >
                                    <button
                                        class="p-1.5 text-slate-400 hover:text-primary-600 hover:bg-primary-50 rounded-md transition-colors"
                                        title="编辑"
                                        on:click={() => openEditModal(g)}
                                    >
                                        <Pencil size={14} />
                                    </button>
                                    <button
                                        class="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-md transition-colors"
                                        title="删除"
                                        on:click={() => handleDelete(g)}
                                    >
                                        <Trash2 size={14} />
                                    </button>
                                </div>
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    {/if}
</div>

<!-- Modal -->
{#if showModal}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        on:click|self={closeModal}
        on:keydown={(e) => e.key === "Escape" && closeModal()}
    >
        <div
            class="bg-white rounded-2xl shadow-xl border border-slate-100 w-full max-w-md max-h-[85vh] overflow-y-auto p-6"
        >
            <h3 class="text-lg font-bold text-slate-800 mb-4">
                {editingGroup ? "编辑用户组" : "创建用户组"}
            </h3>

            <div class="space-y-4">
                <div>
                    <label
                        class="block text-sm font-medium text-slate-700 mb-1"
                    >
                        组名 <span class="text-red-500">*</span>
                    </label>
                    <input
                        type="text"
                        bind:value={formGroupName}
                        placeholder="用户组名称"
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                    />
                </div>

                <div>
                    <label
                        class="block text-sm font-medium text-slate-700 mb-1"
                    >
                        组编码 <span class="text-red-500">*</span>
                    </label>
                    <input
                        type="text"
                        bind:value={formGroupCode}
                        placeholder="唯一编码，如 admins"
                        disabled={!!editingGroup}
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200 disabled:bg-slate-50"
                    />
                </div>

                <div>
                    <label
                        class="block text-sm font-medium text-slate-700 mb-1"
                    >
                        公司外部组识别号
                    </label>
                    <input
                        type="text"
                        bind:value={formExternalGroupID}
                        placeholder="如 HR/ERP 系统组织编码"
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                    />
                    <p class="text-xs text-slate-400 mt-1">
                        用于多系统集成和外部组织映射
                    </p>
                </div>

                <div>
                    <label
                        class="block text-sm font-medium text-slate-700 mb-1"
                    >
                        数据访问权限
                    </label>
                    <select
                        bind:value={formDataAccessScope}
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                    >
                        {#each scopeOptions as opt}
                            <option value={opt.value}>{opt.label}</option>
                        {/each}
                    </select>
                </div>

                <div>
                    <label
                        class="block text-sm font-medium text-slate-700 mb-1"
                    >
                        描述
                    </label>
                    <textarea
                        bind:value={formDescription}
                        placeholder="用户组描述信息"
                        rows={3}
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200 resize-none"
                    />
                </div>
            </div>

            <div
                class="flex justify-end gap-2 mt-6 pt-4 border-t border-slate-100"
            >
                <button
                    class="px-4 py-2 bg-slate-100 text-slate-700 rounded-lg text-sm font-medium hover:bg-slate-200 transition-colors"
                    on:click={closeModal}
                    disabled={saving}
                >
                    取消
                </button>
                <button
                    class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 transition-colors disabled:opacity-50"
                    on:click={handleSave}
                    disabled={saving}
                >
                    {#if saving}保存中...{:else}保存{/if}
                </button>
            </div>
        </div>
    </div>
{/if}
