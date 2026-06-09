<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import { toast } from "$lib/stores/toast";
    import { Plus, Pencil, Trash2, Zap, Search, Database } from "lucide-svelte";
    import { page } from "$app/stores";

    interface Skill {
        id: string;
        name: string;
        skill_type: string;
        description: string;
        category: string;
        enabled: boolean;
        keywords: string;
        config: string;
        prompt_template: string;
        created_at: string;
    }

    let skills: Skill[] = [];
    let loading = true;
    let error = "";
    let searchQuery = "";

    let showModal = false;
    let editingSkill: Skill | null = null;
    let saving = false;

    let formName = "";
    let formType = "chat";
    let formDescription = "";
    let formCategory = "";
    let formKeywords = "";
    let formEnabled = true;
    let formPromptTemplate = "";
    let formDataSourceID = "";

    let datasources: { id: string; name: string; source_type: string }[] = [];

    const typeOptions = [
        { value: "chat", label: "通用对话", needsDS: false },
        {
            value: "text2sql",
            label: "Text2SQL",
            needsDS: true,
            desc: "自然语言 → SQL 查询，需要关联数据源",
        },
        {
            value: "schema_analysis",
            label: "Schema 分析",
            needsDS: true,
            desc: "自动分析数据库结构，生成数据洞察",
        },
        { value: "file_process", label: "文件处理", needsDS: false },
        { value: "api_caller", label: "API 调用", needsDS: false },
        { value: "report", label: "报告生成", needsDS: false },
        { value: "web_search", label: "网络搜索", needsDS: false },
        { value: "custom", label: "自定义", needsDS: false },
    ];

    // 是否从数据源页面跳转过来（预填充数据源）
    let prefillDS = "";

    $: needsDataSource =
        typeOptions.find((t) => t.value === formType)?.needsDS || false;
    $: currentTypeDesc =
        typeOptions.find((t) => t.value === formType)?.desc || "";

    $: filteredSkills = skills.filter(
        (s) =>
            !searchQuery ||
            s.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
            (s.description || "")
                .toLowerCase()
                .includes(searchQuery.toLowerCase()) ||
            (s.category || "")
                .toLowerCase()
                .includes(searchQuery.toLowerCase()),
    );

    onMount(async () => {
        // 检查是否从数据源页面携带参数跳转
        prefillDS = $page.url.searchParams.get("ds") || "";
        await loadSkills();
        if (prefillDS) {
            await loadDataSources();
            openAddModal();
            formType = "text2sql";
            formDataSourceID = prefillDS;
            const ds = datasources.find((d) => d.id === prefillDS);
            if (ds) {
                formName = `${ds.name} 查询助手`;
                formCategory = "data";
                formDescription = `对 ${ds.name} (${ds.source_type}) 进行自然语言查询`;
            }
        }
    });

    async function loadSkills() {
        loading = true;
        error = "";
        try {
            const data = await api.get<Skill[]>("/admin/skills");
            skills = Array.isArray(data) ? data : [];
        } catch (e: any) {
            error = e.message || "加载技能列表失败";
            skills = [];
        } finally {
            loading = false;
        }
    }

    async function loadDataSources() {
        try {
            const data = await api.get<any[]>("/admin/datasources");
            datasources = Array.isArray(data) ? data : data?.datasources || [];
        } catch {
            datasources = [];
        }
    }

    function openAddModal() {
        editingSkill = null;
        formName = "";
        formType = "chat";
        formDescription = "";
        formCategory = "";
        formKeywords = "";
        formEnabled = true;
        formPromptTemplate = "";
        formDataSourceID = prefillDS || "";
        if (!prefillDS) loadDataSources();
        showModal = true;
    }

    function openEditModal(s: Skill) {
        editingSkill = s;
        formName = s.name;
        formType = s.skill_type;
        formDescription = s.description || "";
        formCategory = s.category || "";
        formKeywords = typeof s.keywords === "string" ? s.keywords : "";
        formEnabled = s.enabled;
        // 尝试从 config 解析 data_source_id
        try {
            const cfg = JSON.parse(s.config || "{}");
            formDataSourceID = cfg.data_source_id || "";
        } catch {
            formDataSourceID = "";
        }
        formPromptTemplate = s.prompt_template || "";
        loadDataSources();
        showModal = true;
    }

    function closeModal() {
        showModal = false;
        editingSkill = null;
        prefillDS = "";
    }

    async function handleSave() {
        if (!formName) {
            toast.error("请填写技能名称");
            return;
        }
        if (needsDataSource && !formDataSourceID) {
            toast.error("此类型技能需要关联数据源");
            return;
        }
        saving = true;
        try {
            const body: Record<string, any> = {
                name: formName,
                skill_type: formType,
                description: formDescription,
                category: formCategory,
                enabled: formEnabled,
            };
            if (formKeywords) {
                body.keywords = formKeywords
                    .split(",")
                    .map((k) => k.trim())
                    .filter(Boolean);
            }
            if (formPromptTemplate) {
                body.prompt_template = formPromptTemplate;
            }
            if (formDataSourceID && needsDataSource) {
                const ds = datasources.find((d) => d.id === formDataSourceID);
                body.config = JSON.stringify({
                    data_source_id: formDataSourceID,
                    data_source_name: ds?.name || "",
                });
            }

            if (editingSkill) {
                await api.put(`/admin/skills/${editingSkill.id}`, body);
                toast.success("技能已更新");
            } else {
                await api.post("/admin/skills", body);
                toast.success("技能已创建");
            }
            closeModal();
            await loadSkills();
        } catch (e: any) {
            toast.error(e.message || "保存失败");
        } finally {
            saving = false;
        }
    }

    async function handleToggle(s: Skill) {
        try {
            await api.put(`/admin/skills/${s.id}`, { enabled: !s.enabled });
            toast.success(s.enabled ? "技能已停用" : "技能已启用");
            await loadSkills();
        } catch (e: any) {
            toast.error(e.message || "操作失败");
        }
    }

    async function handleDelete(s: Skill) {
        if (!confirm(`确定删除技能 "${s.name}" 吗？`)) return;
        try {
            await api.delete(`/admin/skills/${s.id}`);
            toast.success("技能已删除");
            await loadSkills();
        } catch (e: any) {
            toast.error(e.message || "删除失败");
        }
    }

    function getTypeLabel(type: string) {
        const opt = typeOptions.find((t) => t.value === type);
        return opt?.label || type;
    }

    function getCategoryColor(cat: string) {
        const colors: Record<string, string> = {
            data: "bg-blue-50 text-blue-700",
            report: "bg-purple-50 text-purple-700",
            automation: "bg-cyan-50 text-cyan-700",
            chat: "bg-emerald-50 text-emerald-700",
            integration: "bg-amber-50 text-amber-700",
        };
        return colors[cat] || "bg-slate-100 text-slate-600";
    }

    function getSkillDataSource(s: Skill): string {
        try {
            const cfg = JSON.parse(s.config || "{}");
            return cfg.data_source_name || cfg.data_source_id || "";
        } catch {
            return "";
        }
    }
</script>

<svelte:head><title>Skills - OpenTether</title></svelte:head>

<div class="card">
    <div class="flex items-center justify-between mb-6">
        <div>
            <h2 class="text-xl font-bold text-slate-800">Skills 配置</h2>
            <p class="text-sm text-slate-500 mt-1">
                管理 AI 技能模块 · 支持关联数据源实现 Text2SQL
            </p>
        </div>
        <button
            class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 transition-colors flex items-center gap-1.5"
            on:click={openAddModal}
        >
            <Plus size={16} />
            创建 Skill
        </button>
    </div>

    {#if error}
        <div
            class="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm"
        >
            {error}
            <button class="ml-2 underline" on:click={loadSkills}>重试</button>
        </div>
    {/if}

    <div class="mb-4 relative">
        <Search
            size={16}
            class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
        />
        <input
            type="text"
            placeholder="搜索技能名称..."
            bind:value={searchQuery}
            class="w-full pl-9 pr-4 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
        />
    </div>

    {#if loading}
        <div class="text-center py-12 text-slate-400">加载中...</div>
    {:else if filteredSkills.length === 0}
        <div class="text-center py-12 space-y-3">
            <div class="text-4xl">⚡</div>
            <p class="text-slate-500">
                {searchQuery ? "未找到匹配的技能" : "暂无技能"}
            </p>
            {#if !searchQuery}
                <p class="text-sm text-slate-400">
                    点击「创建 Skill」开始，或从数据源页面一键生成
                </p>
            {/if}
        </div>
    {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {#each filteredSkills as s}
                {@const dsName = getSkillDataSource(s)}
                <div
                    class="p-5 rounded-xl border border-slate-200 hover:border-primary-200 hover:shadow-sm transition-all"
                >
                    <div class="flex items-start justify-between mb-3">
                        <div class="flex items-center gap-2">
                            <div
                                class="w-9 h-9 rounded-lg bg-slate-100 flex items-center justify-center"
                            >
                                <Zap size={18} class="text-primary-600" />
                            </div>
                            <div>
                                <h3 class="font-semibold text-sm">{s.name}</h3>
                                <p class="text-xs text-slate-400 mt-0.5">
                                    {getTypeLabel(s.skill_type)}
                                    {#if dsName}
                                        <span class="mx-1">·</span>
                                        <Database size={10} class="inline" />
                                        {dsName}
                                    {/if}
                                </p>
                            </div>
                        </div>
                        <button
                            class="relative w-8 h-5 rounded-full transition-colors cursor-pointer {s.enabled
                                ? 'bg-emerald-500'
                                : 'bg-slate-300'}"
                            on:click={() => handleToggle(s)}
                            title={s.enabled ? "点击停用" : "点击启用"}
                        >
                            <span
                                class="absolute top-0.5 w-4 h-4 rounded-full bg-white shadow-sm transition-all {s.enabled
                                    ? 'left-3.5'
                                    : 'left-0.5'}"
                            />
                        </button>
                    </div>

                    <p class="text-xs text-slate-500 mb-3 line-clamp-2">
                        {s.description || "暂无描述"}
                    </p>

                    <div class="flex items-center gap-1.5 flex-wrap">
                        {#if s.category}
                            <span
                                class="px-2 py-0.5 rounded-full text-xs {getCategoryColor(
                                    s.category,
                                )}"
                            >
                                {s.category}
                            </span>
                        {/if}
                        {#if dsName}
                            <span
                                class="px-2 py-0.5 rounded-full text-xs bg-indigo-50 text-indigo-600"
                            >
                                {dsName}
                            </span>
                        {/if}
                    </div>

                    <div
                        class="flex items-center justify-end gap-1 mt-3 pt-3 border-t border-slate-100"
                    >
                        <button
                            class="p-1.5 text-slate-400 hover:text-primary-600 hover:bg-primary-50 rounded-md transition-colors"
                            title="编辑"
                            on:click={() => openEditModal(s)}
                        >
                            <Pencil size={14} />
                        </button>
                        <button
                            class="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-md transition-colors"
                            title="删除"
                            on:click={() => handleDelete(s)}
                        >
                            <Trash2 size={14} />
                        </button>
                    </div>
                </div>
            {/each}
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
                {editingSkill ? "编辑 Skill" : "创建 Skill"}
            </h3>

            <div class="space-y-4">
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                            >名称 <span class="text-red-500">*</span></label
                        >
                        <input
                            type="text"
                            bind:value={formName}
                            placeholder="技能名称"
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                        />
                    </div>
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                            >类型</label
                        >
                        <select
                            bind:value={formType}
                            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                        >
                            {#each typeOptions as opt}
                                <option value={opt.value}>{opt.label}</option>
                            {/each}
                        </select>
                    </div>
                </div>

                {#if currentTypeDesc}
                    <div
                        class="p-2.5 rounded-lg bg-blue-50 border border-blue-100 text-xs text-blue-700"
                    >
                        💡 {currentTypeDesc}
                    </div>
                {/if}

                <!-- 数据源选择（仅 text2sql / schema_analysis） -->
                {#if needsDataSource}
                    <div>
                        <label
                            class="block text-sm font-medium text-slate-700 mb-1"
                        >
                            关联数据源 <span class="text-red-500">*</span>
                        </label>
                        {#if datasources.length === 0}
                            <p class="text-xs text-amber-600">
                                暂无数据源，请先在<a
                                    href="/admin/datasources"
                                    class="underline">数据源管理</a
                                >中添加
                            </p>
                        {:else}
                            <select
                                bind:value={formDataSourceID}
                                class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                            >
                                <option value="">-- 选择数据源 --</option>
                                {#each datasources as ds}
                                    <option value={ds.id}
                                        >{ds.name} ({ds.source_type})</option
                                    >
                                {/each}
                            </select>
                        {/if}
                    </div>
                {/if}

                <div>
                    <label class="block text-sm font-medium text-slate-700 mb-1"
                        >分类</label
                    >
                    <input
                        type="text"
                        bind:value={formCategory}
                        placeholder="如 data, report, chat"
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                    />
                </div>

                <div>
                    <label class="block text-sm font-medium text-slate-700 mb-1"
                        >描述</label
                    >
                    <textarea
                        bind:value={formDescription}
                        placeholder="技能功能描述"
                        rows={3}
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200 resize-none"
                    />
                </div>

                <div>
                    <label class="block text-sm font-medium text-slate-700 mb-1"
                        >关键词 (逗号分隔)</label
                    >
                    <input
                        type="text"
                        bind:value={formKeywords}
                        placeholder="查询, 数据分析, SQL"
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200"
                    />
                </div>

                <div>
                    <label class="block text-sm font-medium text-slate-700 mb-1"
                        >Prompt 模板 (可选)</label
                    >
                    <textarea
                        bind:value={formPromptTemplate}
                        placeholder="自定义 Prompt 模板...（留空使用默认）"
                        rows={4}
                        class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-200 resize-none font-mono text-xs"
                    />
                </div>

                <div>
                    <label
                        class="flex items-center gap-2 text-sm text-slate-700 cursor-pointer"
                    >
                        <input
                            type="checkbox"
                            bind:checked={formEnabled}
                            class="rounded"
                        />
                        启用此技能
                    </label>
                </div>
            </div>

            <div
                class="flex justify-end gap-2 mt-6 pt-4 border-t border-slate-100"
            >
                <button
                    class="px-4 py-2 bg-slate-100 text-slate-700 rounded-lg text-sm font-medium hover:bg-slate-200 transition-colors"
                    on:click={closeModal}
                    disabled={saving}>取消</button
                >
                <button
                    class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 transition-colors disabled:opacity-50"
                    on:click={handleSave}
                    disabled={saving}
                >
                    {saving ? "保存中..." : "保存"}
                </button>
            </div>
        </div>
    </div>
{/if}
