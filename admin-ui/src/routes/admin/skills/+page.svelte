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
    interface DataSource {
        id: string;
        name: string;
        source_type: string;
        schema_info?: string;
        table_relations?: string;
    }
    interface SchemaTable {
        name: string;
        columns: {
            name: string;
            type?: string;
            key_type?: string;
            comment?: string;
        }[];
    }
    interface Relation {
        from_table: string;
        from_column: string;
        to_table: string;
        to_column: string;
        description?: string;
    }
    interface DataBoundaryRule {
        groups?: string[];
        exclude_groups?: string[];
        users?: string[];
        exclude_users?: string[];
        table: string;
        field: string;
        operator: string;
        user_field: string;
    }
    interface MetricRule {
        metric: string;
        entry_table: string;
        aggregation: string;
        time_field: string;
        filter: string;
    }
    interface EntityRule {
        entity: string;
        table: string;
        name_field: string;
        join_field: string;
    }
    interface UserGroup {
        id: string;
        group_name: string;
        group_code: string;
    }
    interface UserOption {
        id: string;
        global_user_id: string;
        name: string;
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
    let formSelectedTables: Record<string, boolean> = {};
    let formSelectedColumns: Record<string, Record<string, boolean>> = {};
    let formTableDescriptions: Record<string, string> = {};
    let formFieldSemantics: Record<string, Record<string, string>> = {};
    let formRelations: Relation[] = [];
    let formBoundaryRules: DataBoundaryRule[] = [];
    let formEntryTable = "";
    let formMetricRules: MetricRule[] = [];
    let formEntityRules: EntityRule[] = [];
    let formBusinessRules = "";
    let formContextMD = "";
    let activeText2SQLTab:
        | "basic"
        | "relations"
        | "rules"
        | "boundary"
        | "doc" = "basic";
    let generatingAI = false;
    let datasources: DataSource[] = [];
    let userGroups: UserGroup[] = [];
    let users: UserOption[] = [];

    const typeOptions = [
        { value: "chat", label: "通用对话", needsDS: false },
        { value: "text2sql", label: "Text2SQL", needsDS: true },
        { value: "schema_analysis", label: "Schema 分析", needsDS: true },
        { value: "file_process", label: "文件处理", needsDS: false },
        { value: "api_caller", label: "API 调用", needsDS: false },
        { value: "report", label: "报告生成", needsDS: false },
    ];
    let prefillDS = "";
    $: needsDataSource =
        typeOptions.find((t) => t.value === formType)?.needsDS || false;
    $: selectedDataSource =
        datasources.find((d) => d.id === formDataSourceID) || null;
    $: selectedSchemaTables = schemaTables(selectedDataSource);
    $: selectedTablesList = selectedSchemaTables.filter(
        (t) => formSelectedTables[t.name],
    );
    $: selectedTableCount = selectedTablesList.length;
    $: selectedFieldCount = selectedSchemaTables.reduce(
        (sum, t) =>
            sum +
            Object.values(formSelectedColumns[t.name] || {}).filter(Boolean)
                .length,
        0,
    );

    onMount(async () => {
        prefillDS = $page.url.searchParams.get("ds") || "";
        await Promise.all([loadSkills(), loadUsersAndGroups()]);
    });

    async function loadSkills() {
        loading = true;
        try {
            const data = await api.get<Skill[]>("/admin/skills");
            skills = Array.isArray(data) ? data : [];
        } catch (e: any) {
            error = e.message;
        } finally {
            loading = false;
        }
    }
    async function loadUsersAndGroups() {
        try {
            const [gs, us] = await Promise.all([
                api.get<UserGroup[]>("/admin/groups"),
                api.get<UserOption[]>("/admin/users"),
            ]);
            userGroups = Array.isArray(gs) ? gs : [];
            users = Array.isArray(us) ? us : [];
        } catch {
            userGroups = [];
            users = [];
        }
    }
    async function loadDataSources() {
        try {
            const data = await api.get<any>("/admin/datasources");
            datasources = data.datasources || data || [];
        } catch {}
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
        resetText2SQLConfig();
        loadDataSources();
        showModal = true;
    }
    function openEditModal(s: Skill) {
        editingSkill = s;
        formName = s.name;
        formType = s.skill_type;
        formDescription = s.description || "";
        formCategory = s.category || "";
        formKeywords = parseKeywordsDisplay(s.keywords);
        formEnabled = s.enabled;
        formPromptTemplate = s.prompt_template || "";
        resetText2SQLConfig();
        try {
            const cfg = JSON.parse(s.config || "{}");
            formDataSourceID = cfg.data_source_id || "";
            formBusinessRules = cfg.business_rules || "";
            formContextMD = cfg.context_md || "";
            loadText2SQLConfig(cfg);
        } catch {
            formDataSourceID = "";
            resetText2SQLConfig();
        }
        loadDataSources();
        showModal = true;
    }
    function closeModal() {
        showModal = false;
        editingSkill = null;
    }
    function resetText2SQLConfig() {
        formSelectedTables = {};
        formSelectedColumns = {};
        formTableDescriptions = {};
        formFieldSemantics = {};
        formRelations = [];
        formBoundaryRules = [];
        formEntryTable = "";
        formMetricRules = [];
        formEntityRules = [];
        formBusinessRules = "";
        formContextMD = "";
    }
    function parseKeywordsDisplay(val: string): string {
        if (!val) return "";
        const t = val.trim();
        if (t.startsWith("[")) {
            try {
                const arr = JSON.parse(t);
                if (Array.isArray(arr)) return arr.join(", ");
            } catch {}
        }
        return val;
    }
    function schemaTables(ds: DataSource | null): SchemaTable[] {
        if (!ds?.schema_info) return [];
        try {
            if (ds.schema_info.trim().startsWith("[")) {
                const parsed = JSON.parse(ds.schema_info.trim());
                return Array.isArray(parsed) ? parsed : [];
            }
        } catch {}
        return [];
    }
    function toggleTable(table: SchemaTable) {
        resetRulesAfterTableChange();
        formSelectedTables = {
            ...formSelectedTables,
            [table.name]: !formSelectedTables[table.name],
        };
        if (formSelectedTables[table.name]) {
            const cols: Record<string, boolean> = {};
            for (const c of table.columns || []) cols[c.name] = true;
            formSelectedColumns = {
                ...formSelectedColumns,
                [table.name]: cols,
            };
        }
    }
    function toggleColumn(table: string, column: string) {
        const cols = { ...(formSelectedColumns[table] || {}) };
        cols[column] = !cols[column];
        formSelectedColumns = { ...formSelectedColumns, [table]: cols };
    }
    function resetRulesAfterTableChange() {
        formEntryTable = "";
        formRelations = [];
        formMetricRules = [];
        formEntityRules = [];
        formBoundaryRules = [];
        formBusinessRules = "";
        formContextMD = "";
    }
    function selectAllTables() {
        resetRulesAfterTableChange();
        const selected: Record<string, boolean> = {};
        const cols: Record<string, Record<string, boolean>> = {};
        for (const t of selectedSchemaTables) {
            selected[t.name] = true;
            cols[t.name] = {};
            for (const c of t.columns || []) cols[t.name][c.name] = true;
        }
        formSelectedTables = selected;
        formSelectedColumns = cols;
    }
    function clearSelectedTables() {
        resetRulesAfterTableChange();
        formSelectedTables = {};
        formSelectedColumns = {};
    }
    function selectBusinessTables() {
        resetRulesAfterTableChange();
        const patterns = [
            "order",
            "sale",
            "profile",
            "staff",
            "product",
            "customer",
            "goods",
            "pay",
        ];
        const selected: Record<string, boolean> = {};
        const cols: Record<string, Record<string, boolean>> = {};
        for (const t of selectedSchemaTables) {
            const n = t.name.toLowerCase();
            if (patterns.some((p) => n.includes(p))) {
                selected[t.name] = true;
                cols[t.name] = {};
                for (const c of t.columns || []) cols[t.name][c.name] = true;
            }
        }
        formSelectedTables = selected;
        formSelectedColumns = cols;
        toast.success(`已选择 ${Object.keys(selected).length} 张常见业务表`);
    }
    function addRelation() {
        formRelations = [
            ...formRelations,
            { from_table: "", from_column: "", to_table: "", to_column: "" },
        ];
    }
    function removeRelation(i: number) {
        formRelations = formRelations.filter((_, idx) => idx !== i);
    }
    function addMetricRule() {
        formMetricRules = [
            ...formMetricRules,
            {
                metric: "订单数",
                entry_table: formEntryTable,
                aggregation: "COUNT(*)",
                time_field: "create_time",
                filter: "",
            },
        ];
    }
    function removeMetricRule(i: number) {
        formMetricRules = formMetricRules.filter((_, idx) => idx !== i);
    }
    function addEntityRule() {
        formEntityRules = [
            ...formEntityRules,
            {
                entity: "员工",
                table: "t_profile",
                name_field: "real_name",
                join_field: "user_id",
            },
        ];
    }
    function removeEntityRule(i: number) {
        formEntityRules = formEntityRules.filter((_, idx) => idx !== i);
    }
    function addBoundaryRule() {
        formBoundaryRules = [
            ...formBoundaryRules,
            {
                table: "",
                field: "",
                operator: "=",
                user_field: "company_user_id",
                exclude_groups: [],
            },
        ];
    }
    function removeBoundaryRule(i: number) {
        formBoundaryRules = formBoundaryRules.filter((_, idx) => idx !== i);
    }
    function boundaryColumns(tableName: string) {
        return (
            selectedTablesList.find((t) => t.name === tableName)?.columns || []
        );
    }

    function handleFieldSemanticInput(table: string, column: string, e: Event) {
        const fields = { ...(formFieldSemantics[table] || {}) };
        fields[column] = (e.currentTarget as HTMLInputElement).value;
        formFieldSemantics = { ...formFieldSemantics, [table]: fields };
    }

    function loadText2SQLConfig(cfg: any) {
        if (Array.isArray(cfg.selected_tables)) {
            for (const t of cfg.selected_tables) {
                if (typeof t === "string") formSelectedTables[t] = true;
                else if (t?.name) {
                    formSelectedTables[t.name] = true;
                    formTableDescriptions[t.name] = t.description || "";
                    formSelectedColumns[t.name] = {};
                    if (Array.isArray(t.columns)) {
                        for (const c of t.columns) {
                            if (typeof c === "string")
                                formSelectedColumns[t.name][c] = true;
                            else if (c?.name) {
                                formSelectedColumns[t.name][c.name] = true;
                                if (!formFieldSemantics[t.name])
                                    formFieldSemantics[t.name] = {};
                                formFieldSemantics[t.name][c.name] =
                                    c.description || "";
                            }
                        }
                    }
                }
            }
        }
        if (Array.isArray(cfg.table_relations)) {
            formRelations = cfg.table_relations;
            if (!Array.isArray(cfg.selected_tables)) {
                for (const r of cfg.table_relations) {
                    if (r?.from_table) formSelectedTables[r.from_table] = true;
                    if (r?.to_table) formSelectedTables[r.to_table] = true;
                    if (r?.from_table && r?.from_column) {
                        formSelectedColumns[r.from_table] = {
                            ...(formSelectedColumns[r.from_table] || {}),
                            [r.from_column]: true,
                        };
                    }
                    if (r?.to_table && r?.to_column) {
                        formSelectedColumns[r.to_table] = {
                            ...(formSelectedColumns[r.to_table] || {}),
                            [r.to_column]: true,
                        };
                    }
                }
            }
        }
        if (Array.isArray(cfg.data_boundary_rules))
            formBoundaryRules = cfg.data_boundary_rules;
        formEntryTable = cfg.entry_table || "";
        formMetricRules = Array.isArray(cfg.metric_rules)
            ? cfg.metric_rules
            : [];
        formEntityRules = Array.isArray(cfg.entity_rules)
            ? cfg.entity_rules
            : [];
    }

    async function handleSave() {
        if (!formName) {
            toast.error("请填写技能名称");
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
                const cfg: Record<string, any> = {
                    data_source_id: formDataSourceID,
                    data_source_name: ds?.name || "",
                };
                if (formType === "text2sql") {
                    cfg.entry_table = formEntryTable;
                    cfg.table_relations = formRelations.filter(
                        (r) => r.from_table && r.to_table,
                    );
                    cfg.metric_rules = formMetricRules.filter((r) => r.metric);
                    cfg.entity_rules = formEntityRules.filter((r) => r.entity);
                    cfg.data_boundary_rules = formBoundaryRules.filter(
                        (r) => r.table && r.field,
                    );
                    cfg.business_rules = formBusinessRules;
                    cfg.context_md = formContextMD;
                    // 保存选中的表和字段，用于刷新后回显
                    cfg.selected_tables = selectedSchemaTables
                        .filter((t) => formSelectedTables[t.name])
                        .map((t) => ({
                            name: t.name,
                            description: formTableDescriptions[t.name] || "",
                            columns: (t.columns || [])
                                .filter(
                                    (c) =>
                                        formSelectedColumns[t.name]?.[c.name],
                                )
                                .map((c) => ({
                                    name: c.name,
                                    description:
                                        formFieldSemantics[t.name]?.[c.name] ||
                                        "",
                                })),
                        }));
                }
                body.config = JSON.stringify(cfg);
            }
            if (editingSkill) {
                await api.put(`/admin/skills/${editingSkill.id}`, body);
                toast.success("已更新");
            } else {
                await api.post("/admin/skills", body);
                toast.success("已创建");
            }
            closeModal();
            await loadSkills();
        } catch (e: any) {
            toast.error(e.message || "保存失败");
        } finally {
            saving = false;
        }
    }
    async function deleteSkill(s: Skill) {
        if (!confirm(`确定删除 "${s.name}" 吗？`)) return;
        try {
            await api.delete(`/admin/skills/${s.id}`);
            toast.success("已删除");
            await loadSkills();
        } catch (e: any) {
            toast.error(e.message || "删除失败");
        }
    }

    async function handleToggle(s: Skill) {
        try {
            await api.put(`/admin/skills/${s.id}`, { enabled: !s.enabled });
            toast.success(s.enabled ? "已停用" : "已启用");
            await loadSkills();
        } catch (e: any) {
            toast.error(e.message || "操作失败");
        }
    }

    async function handleTestSkill(s: Skill) {
        try {
            const data = await api.post<any>(`/admin/skills/${s.id}/test`, {
                input: "",
            });
            const score =
                data.score !== undefined ? `（评分 ${data.score}/100）` : "";
            if (data.success)
                toast.success(`${data.output || "测试通过"}${score}`);
            else toast.error(`${data.output || "测试未通过"}${score}`);
        } catch (e: any) {
            toast.error(e.message || "测试失败");
        }
    }

    async function handleDelete(s: Skill) {
        await deleteSkill(s);
    }

    function getTypeLabel(type: string) {
        const opt = typeOptions.find((t) => t.value === type);
        return opt?.label || type;
    }

    function getCategoryColor(cat: string) {
        const c: Record<string, string> = {
            data: "bg-blue-50 text-blue-700",
            report: "bg-purple-50 text-purple-700",
            automation: "bg-cyan-50 text-cyan-700",
            chat: "bg-emerald-50 text-emerald-700",
            integration: "bg-amber-50 text-amber-700",
        };
        return c[cat] || "bg-slate-100 text-slate-600";
    }

    function getSkillDataSource(s: Skill): string {
        try {
            const cfg = JSON.parse(s.config || "{}");
            return cfg.data_source_name || cfg.data_source_id || "";
        } catch {
            return "";
        }
    }

    function isBuiltinSkill(s: Skill): boolean {
        if (s.category === "系统内置") return true;
        try {
            const cfg = JSON.parse(s.config || "{}");
            return cfg.builtin === true;
        } catch {
            return false;
        }
    }
</script>

<svelte:head><title>Skills - OpenTether</title></svelte:head>

<div class="card">
    <div class="flex items-center justify-between mb-6">
        <h2 class="text-xl font-bold text-slate-800">Skills 管理</h2>
        <button
            class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium flex items-center gap-1.5"
            on:click={openAddModal}><Plus size={16} /> 创建 Skill</button
        >
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {#each skills as s}
            {@const dsName = getSkillDataSource(s)}
            {@const builtin = isBuiltinSkill(s)}
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
                        class="relative w-8 h-5 rounded-full {builtin
                            ? 'cursor-not-allowed opacity-60'
                            : 'cursor-pointer'} {s.enabled
                            ? 'bg-emerald-500'
                            : 'bg-slate-300'}"
                        on:click|stopPropagation={() => handleToggle(s)}
                        disabled={builtin}
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
                            )}">{s.category}</span
                        >
                    {/if}
                    {#if builtin}
                        <span
                            class="px-2 py-0.5 rounded-full text-xs bg-amber-50 text-amber-700"
                            >系统内置</span
                        >
                    {/if}
                    {#if dsName}
                        <span
                            class="px-2 py-0.5 rounded-full text-xs bg-indigo-50 text-indigo-600"
                            >{dsName}</span
                        >
                    {/if}
                </div>
                <div
                    class="flex items-center justify-end gap-1 mt-3 pt-3 border-t border-slate-100"
                >
                    <button
                        class="p-1.5 rounded-md transition-colors text-slate-400 hover:text-emerald-600 hover:bg-emerald-50"
                        title="测试/诊断"
                        on:click|stopPropagation={() => handleTestSkill(s)}
                        ><Zap size={14} /></button
                    >
                    <button
                        class="p-1.5 rounded-md transition-colors {builtin
                            ? 'text-slate-300 cursor-not-allowed'
                            : 'text-slate-400 hover:text-primary-600 hover:bg-primary-50'}"
                        title={builtin ? "系统内置 Skill 不可编辑" : "编辑"}
                        on:click|stopPropagation={() => openEditModal(s)}
                        disabled={builtin}><Pencil size={14} /></button
                    >
                    <button
                        class="p-1.5 rounded-md transition-colors {builtin
                            ? 'text-slate-300 cursor-not-allowed'
                            : 'text-slate-400 hover:text-red-600 hover:bg-red-50'}"
                        title={builtin ? "系统内置 Skill 不可删除" : "删除"}
                        on:click|stopPropagation={() => handleDelete(s)}
                        disabled={builtin}><Trash2 size={14} /></button
                    >
                </div>
            </div>
        {/each}
    </div>
</div>

{#if showModal}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        on:click|self={closeModal}
    >
        <div
            class="bg-white rounded-2xl shadow-xl border border-slate-100 w-full {formType ===
            'text2sql'
                ? 'max-w-4xl'
                : 'max-w-lg'} max-h-[88vh] overflow-y-auto p-6"
        >
            <h3 class="text-lg font-bold text-slate-800 mb-4">
                {editingSkill ? "编辑 Skill" : "创建 Skill"}
            </h3>
            <div class="grid grid-cols-2 gap-4 mb-4">
                <div>
                    <label class="block text-sm font-medium text-slate-700 mb-1"
                        >名称 *</label
                    ><input
                        type="text"
                        bind:value={formName}
                        class="w-full px-3 py-2 border rounded-lg text-sm"
                        placeholder="技能名称"
                    />
                </div>
                <div>
                    <label class="block text-sm font-medium text-slate-700 mb-1"
                        >类型</label
                    ><select
                        bind:value={formType}
                        class="w-full px-3 py-2 border rounded-lg text-sm"
                        >{#each typeOptions as opt}<option value={opt.value}
                                >{opt.label}</option
                            >{/each}</select
                    >
                </div>
            </div>

            {#if needsDataSource}
                <div class="mb-4">
                    <label class="block text-sm font-medium text-slate-700 mb-1"
                        >关联数据源 *</label
                    >
                    <select
                        bind:value={formDataSourceID}
                        class="w-full px-3 py-2 border rounded-lg text-sm"
                    >
                        <option value="">-- 选择数据源 --</option>
                        {#each datasources as ds}<option value={ds.id}
                                >{ds.name} ({ds.source_type})</option
                            >{/each}
                    </select>
                </div>
            {/if}

            {#if formType === "text2sql" && selectedDataSource}
                <!-- Tab Switcher -->
                <div
                    class="flex items-center gap-1 p-1 bg-slate-100 rounded-xl mb-4 w-fit"
                >
                    <button
                        class="px-4 py-2 rounded-lg text-sm font-medium transition-colors {activeText2SQLTab ===
                        'basic'
                            ? 'bg-white text-slate-800 shadow-sm'
                            : 'text-slate-500'}"
                        on:click={() => (activeText2SQLTab = "basic")}
                        >基础</button
                    >
                    <button
                        class="px-4 py-2 rounded-lg text-sm font-medium transition-colors {activeText2SQLTab ===
                        'relations'
                            ? 'bg-white text-slate-800 shadow-sm'
                            : 'text-slate-500'}"
                        on:click={() => (activeText2SQLTab = "relations")}
                        >关系</button
                    >
                    <button
                        class="px-4 py-2 rounded-lg text-sm font-medium transition-colors {activeText2SQLTab ===
                        'rules'
                            ? 'bg-white text-slate-800 shadow-sm'
                            : 'text-slate-500'}"
                        on:click={() => (activeText2SQLTab = "rules")}
                        >规则</button
                    >
                    <button
                        class="px-4 py-2 rounded-lg text-sm font-medium transition-colors {activeText2SQLTab ===
                        'boundary'
                            ? 'bg-white text-slate-800 shadow-sm'
                            : 'text-slate-500'}"
                        on:click={() => (activeText2SQLTab = "boundary")}
                        >边界</button
                    >
                    <button
                        class="px-4 py-2 rounded-lg text-sm font-medium transition-colors {activeText2SQLTab ===
                        'doc'
                            ? 'bg-white text-slate-800 shadow-sm'
                            : 'text-slate-500'}"
                        on:click={() => (activeText2SQLTab = "doc")}
                        >文档</button
                    >
                </div>

                {#if activeText2SQLTab === "basic"}
                    <div class="bg-white rounded-lg border p-3 space-y-3">
                        <div class="flex items-center justify-between">
                            <div class="font-semibold text-sm text-slate-700">
                                表与字段选择
                            </div>
                            <div class="flex gap-2">
                                <button
                                    class="text-xs px-2 py-1 rounded bg-blue-50 text-blue-700"
                                    on:click={selectBusinessTables}
                                    >智能选业务表</button
                                ><button
                                    class="text-xs px-2 py-1 rounded bg-slate-100"
                                    on:click={selectAllTables}>全选</button
                                ><button
                                    class="text-xs px-2 py-1 rounded bg-slate-100"
                                    on:click={clearSelectedTables}>清空</button
                                >
                            </div>
                        </div>
                        <div
                            class="space-y-2 max-h-64 overflow-y-auto bg-slate-50 rounded-lg border p-2"
                        >
                            {#each selectedSchemaTables as table}
                                <details
                                    class="border border-slate-100 rounded-lg"
                                    open={false}
                                >
                                    <summary
                                        class="cursor-pointer px-2 py-1.5 bg-white flex items-center gap-2"
                                    >
                                        <input
                                            type="checkbox"
                                            checked={!!formSelectedTables[
                                                table.name
                                            ]}
                                            on:click|stopPropagation={() =>
                                                toggleTable(table)}
                                        />
                                        <span
                                            class="font-mono text-xs font-semibold"
                                            >{table.name}</span
                                        >
                                        <span class="text-[11px] text-slate-400"
                                            >{table.columns?.length || 0} fields</span
                                        >
                                    </summary>
                                    {#if formSelectedTables[table.name]}
                                        <div class="p-2 space-y-2">
                                            <input
                                                class="w-full px-2 py-1.5 border rounded text-xs"
                                                placeholder="表作用说明"
                                                bind:value={
                                                    formTableDescriptions[
                                                        table.name
                                                    ]
                                                }
                                            />
                                            {#each table.columns || [] as col}
                                                <div
                                                    class="grid grid-cols-[auto_1fr_1.5fr] items-center gap-2 text-xs"
                                                >
                                                    <input
                                                        type="checkbox"
                                                        checked={!!formSelectedColumns[
                                                            table.name
                                                        ]?.[col.name]}
                                                        on:change={() =>
                                                            toggleColumn(
                                                                table.name,
                                                                col.name,
                                                            )}
                                                    />
                                                    <span
                                                        class="font-mono text-slate-600 truncate"
                                                        title={col.type || ""}
                                                        >{col.name}</span
                                                    >
                                                    <input
                                                        class="px-2 py-1 border rounded"
                                                        placeholder="含义/口径"
                                                        value={formFieldSemantics[
                                                            table.name
                                                        ]?.[col.name] ||
                                                            col.comment ||
                                                            ""}
                                                        on:input={(e) =>
                                                            handleFieldSemanticInput(
                                                                table.name,
                                                                col.name,
                                                                e,
                                                            )}
                                                    />
                                                </div>
                                            {/each}
                                        </div>
                                    {/if}
                                </details>
                            {/each}
                        </div>
                        <div
                            class="grid grid-cols-1 md:grid-cols-3 gap-3 bg-white rounded-lg border p-3"
                        >
                            <label
                                class="text-xs font-medium text-slate-600 md:col-span-1"
                                >入口表 <select
                                    class="mt-1 w-full px-3 py-2 border rounded-lg text-sm"
                                    bind:value={formEntryTable}
                                    ><option value="">不指定</option
                                    >{#each selectedTablesList as t}<option
                                            value={t.name}>{t.name}</option
                                        >{/each}</select
                                ></label
                            >
                            <div
                                class="md:col-span-2 text-xs text-slate-500 flex items-center"
                            >
                                建议订单查询选择 t_order，员工查询选择
                                t_profile。
                            </div>
                        </div>
                    </div>
                {/if}

                {#if activeText2SQLTab === "relations"}
                    <div class="bg-white rounded-lg border p-3">
                        <div class="flex items-center justify-between mb-2">
                            <label class="text-xs font-semibold text-slate-600"
                                >表关系</label
                            >
                            <button
                                class="text-xs text-primary-600"
                                on:click={addRelation}>+ 添加</button
                            >
                        </div>
                        <div
                            class="grid grid-cols-1 lg:grid-cols-[260px_1fr] gap-4"
                        >
                            <div
                                class="bg-slate-50 rounded-lg border p-3 text-xs"
                            >
                                <div class="font-medium text-slate-600 mb-2">
                                    关系树
                                </div>
                                {#if formEntryTable}
                                    {@const children = formRelations.filter(
                                        (r) =>
                                            r.from_table === formEntryTable &&
                                            r.to_table,
                                    )}
                                    <div class="space-y-1">
                                        <div
                                            class="font-mono font-semibold text-primary-700"
                                        >
                                            {formEntryTable}
                                        </div>
                                        {#each children as rel}
                                            <div
                                                class="ml-3 border-l-2 border-slate-200 pl-2 text-slate-600"
                                            >
                                                ├ {rel.to_table}
                                            </div>
                                        {/each}
                                    </div>
                                {:else}<div class="text-slate-400">
                                        设置入口表后显示关系树
                                    </div>{/if}
                            </div>
                            <div class="space-y-2 max-h-64 overflow-y-auto">
                                {#each formRelations as rel, i}
                                    <div
                                        class="grid grid-cols-[1fr_1fr_auto_1fr_1fr_1.5fr_auto] gap-1 items-center bg-slate-50 p-2 rounded-lg"
                                    >
                                        <select
                                            class="px-2 py-1 border rounded text-xs"
                                            bind:value={rel.from_table}
                                            on:change={() =>
                                                (rel.from_column = "")}
                                            ><option value="">源表</option
                                            >{#each selectedTablesList as t}<option
                                                    value={t.name}
                                                    >{t.name}</option
                                                >{/each}</select
                                        >
                                        <select
                                            class="px-2 py-1 border rounded text-xs"
                                            bind:value={rel.from_column}
                                            ><option value="">源字段</option
                                            >{#each boundaryColumns(rel.from_table) as c}<option
                                                    value={c.name}
                                                    >{c.name}</option
                                                >{/each}</select
                                        >
                                        <span class="text-slate-400">→</span>
                                        <select
                                            class="px-2 py-1 border rounded text-xs"
                                            bind:value={rel.to_table}
                                            on:change={() =>
                                                (rel.to_column = "")}
                                            ><option value="">目标表</option
                                            >{#each selectedTablesList as t}<option
                                                    value={t.name}
                                                    >{t.name}</option
                                                >{/each}</select
                                        >
                                        <select
                                            class="px-2 py-1 border rounded text-xs"
                                            bind:value={rel.to_column}
                                            ><option value="">目标字段</option
                                            >{#each boundaryColumns(rel.to_table) as c}<option
                                                    value={c.name}
                                                    >{c.name}</option
                                                >{/each}</select
                                        >
                                        <input
                                            class="px-2 py-1 border rounded text-xs"
                                            placeholder="说明"
                                            bind:value={rel.description}
                                        />
                                        <button
                                            class="text-red-400"
                                            on:click={() => removeRelation(i)}
                                            >×</button
                                        >
                                    </div>
                                {/each}
                            </div>
                        </div>
                    </div>
                {/if}

                {#if activeText2SQLTab === "rules"}
                    <div class="grid grid-cols-1 lg:grid-cols-2 gap-3">
                        <div class="bg-white rounded-lg border p-3">
                            <div class="flex items-center justify-between mb-2">
                                <label
                                    class="text-xs font-semibold text-slate-600"
                                    >指标规则</label
                                ><button
                                    class="text-xs text-primary-600"
                                    on:click={addMetricRule}>+ 添加</button
                                >
                            </div>
                            {#each formMetricRules as r, i}
                                <div
                                    class="grid grid-cols-[1fr_1fr_1fr_1fr_auto] gap-1 items-center mb-2"
                                >
                                    <input
                                        class="px-2 py-1 border rounded text-xs"
                                        placeholder="指标名"
                                        bind:value={r.metric}
                                    />
                                    <select
                                        class="px-2 py-1 border rounded text-xs"
                                        bind:value={r.entry_table}
                                        ><option value="">入口表</option
                                        >{#each selectedTablesList as t}<option
                                                value={t.name}>{t.name}</option
                                            >{/each}</select
                                    >
                                    <input
                                        class="px-2 py-1 border rounded text-xs"
                                        placeholder="聚合"
                                        bind:value={r.aggregation}
                                    />
                                    <select
                                        class="px-2 py-1 border rounded text-xs"
                                        bind:value={r.time_field}
                                        ><option value="">时间字段</option
                                        >{#each boundaryColumns(r.entry_table) as c}<option
                                                value={c.name}>{c.name}</option
                                            >{/each}</select
                                    >
                                    <button
                                        class="text-red-400"
                                        on:click={() => removeMetricRule(i)}
                                        >×</button
                                    >
                                </div>
                            {/each}
                        </div>
                        <div class="bg-white rounded-lg border p-3">
                            <div class="flex items-center justify-between mb-2">
                                <label
                                    class="text-xs font-semibold text-slate-600"
                                    >实体规则</label
                                ><button
                                    class="text-xs text-primary-600"
                                    on:click={addEntityRule}>+ 添加</button
                                >
                            </div>
                            {#each formEntityRules as r, i}
                                <div
                                    class="grid grid-cols-[1fr_1fr_1fr_1fr_auto] gap-1 items-center mb-2"
                                >
                                    <input
                                        class="px-2 py-1 border rounded text-xs"
                                        placeholder="实体"
                                        bind:value={r.entity}
                                    />
                                    <select
                                        class="px-2 py-1 border rounded text-xs"
                                        bind:value={r.table}
                                        ><option value="">实体表</option
                                        >{#each selectedTablesList as t}<option
                                                value={t.name}>{t.name}</option
                                            >{/each}</select
                                    >
                                    <select
                                        class="px-2 py-1 border rounded text-xs"
                                        bind:value={r.name_field}
                                        ><option value="">名称字段</option
                                        >{#each boundaryColumns(r.table) as c}<option
                                                value={c.name}>{c.name}</option
                                            >{/each}</select
                                    >
                                    <select
                                        class="px-2 py-1 border rounded text-xs"
                                        bind:value={r.join_field}
                                        ><option value="">关联字段</option
                                        >{#each boundaryColumns(r.table) as c}<option
                                                value={c.name}>{c.name}</option
                                            >{/each}</select
                                    >
                                    <button
                                        class="text-red-400"
                                        on:click={() => removeEntityRule(i)}
                                        >×</button
                                    >
                                </div>
                            {/each}
                        </div>
                    </div>
                {/if}

                {#if activeText2SQLTab === "boundary"}
                    <div class="bg-white rounded-lg border p-3">
                        <div class="flex items-center justify-between mb-2">
                            <label class="text-xs font-semibold text-slate-600"
                                >数据边界规则</label
                            ><button
                                class="text-xs text-primary-600"
                                on:click={addBoundaryRule}>+ 添加</button
                            >
                        </div>
                        {#each formBoundaryRules as rule, i}
                            <div
                                class="p-3 rounded-lg bg-white border border-slate-200 space-y-2 mb-2"
                            >
                                <div class="grid grid-cols-2 gap-2">
                                    <label class="text-[11px] text-slate-500"
                                        >适用用户组 <select
                                            multiple
                                            class="mt-1 w-full px-2 py-1 border rounded text-xs min-h-[72px]"
                                            bind:value={rule.groups}
                                            >{#each userGroups as g}<option
                                                    value={g.id}
                                                    >{g.group_name} ({g.group_code})</option
                                                >{/each}</select
                                        ></label
                                    >
                                    <label class="text-[11px] text-slate-500"
                                        >排除用户组 <select
                                            multiple
                                            class="mt-1 w-full px-2 py-1 border rounded text-xs min-h-[72px]"
                                            bind:value={rule.exclude_groups}
                                            >{#each userGroups as g}<option
                                                    value={g.id}
                                                    >{g.group_name} ({g.group_code})</option
                                                >{/each}</select
                                        ></label
                                    >
                                </div>
                                <div class="grid grid-cols-2 gap-2">
                                    <label class="text-[11px] text-slate-500"
                                        >适用用户 <select
                                            multiple
                                            class="mt-1 w-full px-2 py-1 border rounded text-xs min-h-[72px]"
                                            bind:value={rule.users}
                                            >{#each users as u}<option
                                                    value={u.id}
                                                    >{u.name} ({u.global_user_id})</option
                                                >{/each}</select
                                        ></label
                                    >
                                    <label class="text-[11px] text-slate-500"
                                        >排除用户 <select
                                            multiple
                                            class="mt-1 w-full px-2 py-1 border rounded text-xs min-h-[72px]"
                                            bind:value={rule.exclude_users}
                                            >{#each users as u}<option
                                                    value={u.id}
                                                    >{u.name} ({u.global_user_id})</option
                                                >{/each}</select
                                        ></label
                                    >
                                </div>
                                <div
                                    class="grid grid-cols-[1fr_1fr_90px_1fr_auto] gap-2 items-end"
                                >
                                    <label class="text-[11px] text-slate-500"
                                        >数据表 <select
                                            class="mt-1 w-full px-2 py-1 border rounded text-xs"
                                            bind:value={rule.table}
                                            on:change={() => (rule.field = "")}
                                            ><option value="">选择表</option
                                            >{#each selectedTablesList as t}<option
                                                    value={t.name}
                                                    >{t.name}</option
                                                >{/each}</select
                                        ></label
                                    >
                                    <label class="text-[11px] text-slate-500"
                                        >字段 <select
                                            class="mt-1 w-full px-2 py-1 border rounded text-xs"
                                            bind:value={rule.field}
                                            ><option value="">选择字段</option
                                            >{#each boundaryColumns(rule.table) as c}<option
                                                    value={c.name}
                                                    >{c.name}</option
                                                >{/each}</select
                                        ></label
                                    >
                                    <label class="text-[11px] text-slate-500"
                                        >操作符 <select
                                            class="mt-1 w-full px-2 py-1 border rounded text-xs"
                                            bind:value={rule.operator}
                                            ><option value="=">=</option><option
                                                value="!=">!=</option
                                            ><option value="IN">IN</option
                                            ><option value=">">&gt;</option
                                            ><option value="<">&lt;</option
                                            ></select
                                        ></label
                                    >
                                    <label class="text-[11px] text-slate-500"
                                        >当前用户字段 <select
                                            class="mt-1 w-full px-2 py-1 border rounded text-xs"
                                            bind:value={rule.user_field}
                                            ><option value="company_user_id"
                                                >公司用户ID</option
                                            ><option value="global_user_id"
                                                >全局用户ID</option
                                            ><option value="user_id"
                                                >系统用户ID</option
                                            ><option value="department"
                                                >部门</option
                                            ><option value="group_ids"
                                                >用户组ID</option
                                            ><option value="group_codes"
                                                >用户组Code</option
                                            ></select
                                        ></label
                                    >
                                    <button
                                        class="text-red-400 pb-1"
                                        on:click={() => removeBoundaryRule(i)}
                                        >×</button
                                    >
                                </div>
                            </div>
                        {/each}
                    </div>
                {/if}

                {#if activeText2SQLTab === "doc"}
                    <div class="space-y-3">
                        <div>
                            <label
                                class="block text-xs font-semibold text-slate-600 mb-1"
                                >业务口径</label
                            >
                            <textarea
                                class="w-full px-3 py-2 border border-slate-200 rounded-lg text-xs resize-none"
                                rows="3"
                                bind:value={formBusinessRules}
                                placeholder="如：销售额按 pay_amount 统计"
                            />
                        </div>
                        <div>
                            <label
                                class="block text-xs font-semibold text-slate-600 mb-1"
                                >MD 上下文</label
                            >
                            <textarea
                                class="w-full px-3 py-2 border border-slate-200 rounded-lg text-xs font-mono"
                                rows="6"
                                bind:value={formContextMD}
                                placeholder="点击生成 MD，或直接粘贴上下文文档"
                            />
                        </div>
                    </div>
                {/if}
            {/if}

            <div
                class="flex justify-end gap-3 pt-4 border-t border-slate-100 mt-4"
            >
                <button
                    class="px-4 py-2 bg-slate-100 rounded-lg text-sm"
                    on:click={closeModal}>取消</button
                >
                <button
                    class="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium"
                    on:click={handleSave}
                    disabled={saving}>{saving ? "保存中..." : "保存"}</button
                >
            </div>
        </div>
    </div>
{/if}
