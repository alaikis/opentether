<script lang="ts">
    import { onMount } from "svelte";
    import { api, ApiError } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { Plus, Pencil, Trash2, Search, X } from "lucide-svelte";

    interface User {
        id: string;
        global_user_id: string;
        external_employee_id: string;
        name: string;
        email: string;
        department: string;
        position: string;
        role: string;
        status: string;
        created_at: string;
    }

    let users: User[] = [];
    let loading = true;
    let error = "";
    let searchQuery = "";

    let showModal = false;
    let editingUser: User | null = null;
    let saving = false;

    let formGlobalUserID = "";
    let formExternalEmployeeID = "";
    let formName = "";
    let formEmail = "";
    let formDepartment = "";
    let formPosition = "";
    let formRole = "user";
    let formStatus = "active";
    let formPassword = "";

    $: filteredUsers = users.filter(
        (u) =>
            !searchQuery ||
            u.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
            u.global_user_id
                .toLowerCase()
                .includes(searchQuery.toLowerCase()) ||
            u.email.toLowerCase().includes(searchQuery.toLowerCase()),
    );

    onMount(async () => {
        await loadUsers();
    });

    async function loadUsers() {
        loading = true;
        error = "";
        try {
            const data = await api.get<User[]>("/admin/users");
            users = Array.isArray(data) ? data : [];
        } catch (e: any) {
            error = e.message || "加载用户列表失败";
            users = [];
        } finally {
            loading = false;
        }
    }

    function openAddModal() {
        editingUser = null;
        formGlobalUserID = "";
        formExternalEmployeeID = "";
        formName = "";
        formEmail = "";
        formDepartment = "";
        formPosition = "";
        formRole = "user";
        formStatus = "active";
        formPassword = "";
        showModal = true;
    }

    function openEditModal(u: User) {
        editingUser = u;
        formGlobalUserID = u.global_user_id;
        formExternalEmployeeID = u.external_employee_id || "";
        formName = u.name;
        formEmail = u.email || "";
        formDepartment = u.department || "";
        formPosition = u.position || "";
        formRole = u.role;
        formStatus = u.status;
        formPassword = "";
        showModal = true;
    }

    function closeModal() {
        showModal = false;
        editingUser = null;
    }

    async function handleSave() {
        if (!formGlobalUserID || !formName) {
            toast.error("请填写必填字段");
            return;
        }
        saving = true;
        try {
            const body: Record<string, any> = {
                global_user_id: formGlobalUserID,
                external_employee_id: formExternalEmployeeID,
                name: formName,
                email: formEmail,
                department: formDepartment,
                position: formPosition,
                role: formRole,
                status: formStatus,
            };
            if (formPassword) {
                body.password = formPassword;
            }

            if (editingUser) {
                await api.put(`/admin/users/${editingUser.id}`, body);
                toast.success("用户已更新");
            } else {
                body.password = formPassword || "123456";
                await api.post("/admin/users", body);
                toast.success("用户已创建");
            }
            closeModal();
            await loadUsers();
        } catch (e: any) {
            toast.error(e.message || "保存失败");
        } finally {
            saving = false;
        }
    }

    async function handleDelete(u: User) {
        if (!confirm(`确定删除用户 "${u.name}" 吗？此操作不可撤销。`)) return;
        try {
            await api.delete(`/admin/users/${u.id}`);
            toast.success("用户已删除");
            await loadUsers();
        } catch (e: any) {
            toast.error(e.message || "删除失败");
        }
    }

    function getRoleLabel(role: string) {
        const map: Record<string, string> = {
            admin: "管理员",
            user: "普通用户",
            guest: "访客",
        };
        return map[role] || role;
    }

    function getRoleColor(role: string) {
        const map: Record<string, string> = {
            admin: "bg-primary-50 text-primary-700",
            user: "bg-slate-100 text-slate-600",
            guest: "bg-amber-50 text-amber-700",
        };
        return map[role] || "bg-slate-100 text-slate-600";
    }

    function getStatusColor(status: string) {
        return status === "active"
            ? "text-emerald-600"
            : status === "suspended"
              ? "text-red-600"
              : "text-slate-400";
    }

    function getStatusLabel(status: string) {
        const map: Record<string, string> = {
            active: "活跃",
            inactive: "停用",
            suspended: "已暂停",
        };
        return map[status] || status;
    }
</script>

<svelte:head><title>用户管理 - OpenTether</title></svelte:head>

<div class="card">
    <div class="flex items-center justify-between mb-6">
        <div>
            <h2 class="text-xl font-bold text-slate-800">用户管理</h2>
            <p class="text-sm text-slate-500 mt-1">
                管理系统用户、权限和角色分配
            </p>
        </div>
        <button
            class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 transition-colors flex items-center gap-1.5"
            on:click={openAddModal}
        >
            <Plus size={16} />
            添加用户
        </button>
    </div>

    {#if error}
        <div
            class="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm"
        >
            {error}
            <button class="ml-2 underline" on:click={loadUsers}>重试</button>
        </div>
    {/if}

    <!-- Search -->
    <div class="mb-4 relative">
        <Search
            size={16}
            class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
        />
        <input
            type="text"
            placeholder="搜索用户名、邮箱..."
            bind:value={searchQuery}
            class="w-full pl-9 pr-4 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
        />
        {#if searchQuery}
            <button
                class="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
                on:click={() => (searchQuery = "")}
            >
                <X size={14} />
            </button>
        {/if}
    </div>

    {#if loading}
        <div class="text-center py-12 text-slate-400">加载中...</div>
    {:else if filteredUsers.length === 0}
        <div class="text-center py-12 space-y-3">
            <div class="text-4xl">👥</div>
            <p class="text-slate-500">
                {searchQuery ? "未找到匹配的用户" : "暂无用户"}
            </p>
            {#if !searchQuery}
                <p class="text-sm text-slate-400">
                    点击"添加用户"创建第一个用户
                </p>
            {/if}
        </div>
    {:else}
        <div class="overflow-x-auto">
            <table class="w-full text-sm">
                <thead>
                    <tr class="border-b border-slate-200">
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >用户名</th
                        >
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >员工识别号</th
                        >
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >姓名</th
                        >
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >邮箱</th
                        >
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >角色</th
                        >
                        <th
                            class="text-left py-3 px-4 font-medium text-slate-500"
                            >状态</th
                        >
                        <th
                            class="text-right py-3 px-4 font-medium text-slate-500"
                            >操作</th
                        >
                    </tr>
                </thead>
                <tbody>
                    {#each filteredUsers as u}
                        <tr class="border-b border-slate-100 hover:bg-slate-50">
                            <td class="py-3 px-4 font-medium"
                                >{u.global_user_id}</td
                            >
                            <td
                                class="py-3 px-4 text-slate-500 font-mono text-xs"
                                >{u.external_employee_id || "-"}</td
                            >
                            <td class="py-3 px-4">{u.name}</td>
                            <td class="py-3 px-4 text-slate-500"
                                >{u.email || "-"}</td
                            >
                            <td class="py-3 px-4">
                                <span
                                    class="px-2 py-0.5 rounded-full text-xs {getRoleColor(
                                        u.role,
                                    )}"
                                >
                                    {getRoleLabel(u.role)}
                                </span>
                            </td>
                            <td class="py-3 px-4">
                                <span
                                    class="flex items-center gap-1.5 text-xs {getStatusColor(
                                        u.status,
                                    )}"
                                >
                                    <span
                                        class="w-1.5 h-1.5 rounded-full"
                                        class:bg-emerald-500={u.status ===
                                            "active"}
                                        class:bg-red-500={u.status ===
                                            "suspended"}
                                        class:bg-slate-300={u.status ===
                                            "inactive"}
                                    />
                                    {getStatusLabel(u.status)}
                                </span>
                            </td>
                            <td class="py-3 px-4 text-right">
                                <div
                                    class="flex items-center justify-end gap-1"
                                >
                                    <button
                                        class="p-1.5 text-slate-400 hover:text-primary-600 hover:bg-primary-50 rounded-md transition-colors"
                                        title="编辑"
                                        on:click={() => openEditModal(u)}
                                    >
                                        <Pencil size={14} />
                                    </button>
                                    <button
                                        class="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-md transition-colors"
                                        title="删除"
                                        on:click={() => handleDelete(u)}
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
            class="bg-white rounded-2xl shadow-xl border border-slate-100 w-full max-w-lg max-h-[85vh] overflow-y-auto p-6"
        >
            <h3 class="text-lg font-bold text-slate-800 mb-4">
                {editingUser ? "编辑用户" : "添加用户"}
            </h3>

            <div class="space-y-4">
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                        >
                            用户名 <span class="text-red-500">*</span>
                        </label>
                        <input
                            type="text"
                            bind:value={formGlobalUserID}
                            placeholder="用户登录名"
                            disabled={!!editingUser}
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200 disabled:bg-slate-50"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                        >
                            姓名 <span class="text-red-500">*</span>
                        </label>
                        <input
                            type="text"
                            bind:value={formName}
                            placeholder="用户姓名"
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                        />
                    </div>
                </div>

                <div>
                    <label
                        class="block text-sm font-medium text-slate-700 mb-1"
                    >
                        公司员工识别号
                    </label>
                    <input
                        type="text"
                        bind:value={formExternalEmployeeID}
                        placeholder="如 HR/ERP/企业微信员工ID"
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                    />
                    <p class="text-xs text-slate-400 mt-1">
                        用于多系统集成和外部员工身份映射
                    </p>
                </div>

                <div>
                    <label
                        class="block text-sm font-medium text-slate-700 mb-1"
                    >
                        邮箱
                    </label>
                    <input
                        type="email"
                        bind:value={formEmail}
                        placeholder="email@company.com"
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                    />
                </div>

                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                        >
                            部门
                        </label>
                        <input
                            type="text"
                            bind:value={formDepartment}
                            placeholder="所属部门"
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                        >
                            职位
                        </label>
                        <input
                            type="text"
                            bind:value={formPosition}
                            placeholder="职位名称"
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                        />
                    </div>
                </div>

                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                        >
                            角色
                        </label>
                        <select
                            bind:value={formRole}
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                        >
                            <option value="user">普通用户</option>
                            <option value="admin">管理员</option>
                            <option value="guest">访客</option>
                        </select>
                    </div>
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                        >
                            状态
                        </label>
                        <select
                            bind:value={formStatus}
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                        >
                            <option value="active">活跃</option>
                            <option value="inactive">停用</option>
                            <option value="suspended">已暂停</option>
                        </select>
                    </div>
                </div>

                <div>
                    <label
                        class="block text-sm font-medium text-slate-700 mb-1"
                    >
                        密码 {editingUser ? "(留空则不修改)" : ""}
                        <span class="text-red-500"
                            >{editingUser ? "" : " *"}</span
                        >
                    </label>
                    <input
                        type="password"
                        bind:value={formPassword}
                        placeholder={editingUser
                            ? "留空则不修改密码"
                            : "设置登录密码"}
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
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
