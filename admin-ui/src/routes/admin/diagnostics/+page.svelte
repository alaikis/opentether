<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import {
        Stethoscope,
        AlertTriangle,
        CheckCircle2,
        ExternalLink,
        RefreshCw,
        Loader2,
    } from "lucide-svelte";

    interface CheckItem {
        id: string;
        name: string;
        status: "ok" | "warn" | "error" | "loading";
        message: string;
        fixUrl?: string;
        fixLabel?: string;
    }

    let checks: CheckItem[] = [];
    let loading = true;
    let lastCheck = "";

    async function runDiagnostics() {
        loading = true;
        checks = [
            {
                id: "llm",
                name: "LLM 提供商",
                status: "loading",
                message: "检查中...",
            },
            {
                id: "datasource",
                name: "数据源配置",
                status: "loading",
                message: "检查中...",
            },
            {
                id: "skills",
                name: "Skills 配置",
                status: "loading",
                message: "检查中...",
            },
            {
                id: "sql_audit",
                name: "SQL 审计状态",
                status: "loading",
                message: "检查中...",
            },
        ];
        checks = checks;

        await checkLLM();
        await checkDS();
        await checkSkills();
        await checkSQLAudit();

        loading = false;
        lastCheck = new Date().toLocaleTimeString("zh-CN");
    }

    function update(
        id: string,
        status: "ok" | "warn" | "error",
        msg: string,
        fixUrl?: string,
        fixLabel?: string,
    ) {
        checks = checks.map((c) =>
            c.id === id ? { ...c, status, message: msg, fixUrl, fixLabel } : c,
        );
        checks = checks;
    }

    async function checkLLM() {
        try {
            const data = await api.get<any>("/admin/providers");
            const list = Array.isArray(data) ? data : data?.providers || [];
            const on = list.filter((p: any) => p.enabled);
            if (on.length > 0) {
                update("llm", "ok", `已配置 ${on.length} 个启用的 LLM 提供商`);
            } else {
                update(
                    "llm",
                    "warn",
                    "没有启用的 LLM 提供商，系统无法生成回答",
                    "/admin/providers",
                    "前往配置",
                );
            }
        } catch (e: any) {
            update("llm", "error", `检查失败: ${e.message || "未知错误"}`);
        }
    }

    async function checkDS() {
        try {
            const data = await api.get<any>("/admin/datasources");
            const list = Array.isArray(data) ? data : data?.datasources || [];
            const on = list.filter((d: any) => d.enabled);
            if (on.length === 0) {
                update(
                    "datasource",
                    "error",
                    "没有配置数据源，数据查询功能将报错",
                    "/admin/datasources",
                    "前往配置",
                );
                return;
            }
            const issues: string[] = [];
            for (const ds of on) {
                if (!ds.schema_info || ds.schema_info === "表结构分析中") {
                    issues.push(`${ds.name}: 表结构未获取`);
                }
            }
            if (issues.length > 0) {
                update(
                    "datasource",
                    "warn",
                    issues.join("; "),
                    "/admin/datasources",
                    "查看详情",
                );
            } else {
                update(
                    "datasource",
                    "ok",
                    `已配置 ${on.length} 个数据源，状态正常`,
                );
            }
        } catch (e: any) {
            update(
                "datasource",
                "error",
                `检查失败: ${e.message || "未知错误"}`,
            );
        }
    }

    async function checkSkills() {
        try {
            const data = await api.get<any>("/admin/skills");
            const list = Array.isArray(data) ? data : data?.skills || [];
            const on = list.filter((s: any) => s.enabled);
            if (on.length === 0) {
                update(
                    "skills",
                    "error",
                    "没有启用的 Skill",
                    "/admin/skills",
                    "前往配置",
                );
                return;
            }
            update("skills", "ok", `已启用 ${on.length} 个 Skill，配置正常`);
        } catch (e: any) {
            update("skills", "error", `检查失败: ${e.message || "未知错误"}`);
        }
    }

    async function checkSQLAudit() {
        try {
            const data = await api.get<any>("/admin/sql-audits?limit=5");
            const list = Array.isArray(data) ? data : [];
            const pending = list.filter((a: any) => a.status === "pending");
            if (pending.length > 0) {
                update(
                    "sql_audit",
                    "warn",
                    `有 ${pending.length} 条待审批`,
                    "/admin/sql-audits",
                    "前往审批",
                );
            } else {
                update("sql_audit", "ok", "无待审批项");
            }
        } catch (e: any) {
            update(
                "sql_audit",
                "error",
                `检查失败: ${e.message || "未知错误"}`,
            );
        }
    }

    onMount(() => {
        runDiagnostics();
    });
</script>

<svelte:head><title>诊断中心 - OpenTether</title></svelte:head>

<div class="space-y-6">
    <div class="card">
        <div class="flex items-center justify-between mb-6">
            <div>
                <h2 class="text-xl font-bold text-slate-800">系统诊断中心</h2>
                <p class="text-sm text-slate-500 mt-1">
                    检查系统关键配置，快速定位问题
                </p>
            </div>
            <button
                class="flex items-center gap-1.5 px-3 py-1.5 text-sm border border-slate-200 rounded-lg hover:bg-slate-50 disabled:opacity-50"
                on:click={runDiagnostics}
                disabled={loading}
            >
                {#if loading}<Loader2 size={14} class="animate-spin" />{/if}
                {loading ? "检测中" : "重新检测"}
            </button>
        </div>
        {#if lastCheck}<p class="text-xs text-slate-400 mb-4">
                上次检测: {lastCheck}
            </p>{/if}
        <div class="space-y-3">
            {#each checks as check}
                <div
                    class="flex items-start gap-3 p-4 rounded-xl border {check.status ===
                    'ok'
                        ? 'border-emerald-100 bg-emerald-50/50'
                        : check.status === 'warn'
                          ? 'border-amber-100 bg-amber-50/50'
                          : check.status === 'error'
                            ? 'border-red-100 bg-red-50/50'
                            : 'border-slate-100 bg-slate-50/50'}"
                >
                    <div class="mt-0.5 shrink-0">
                        {#if check.status === "loading"}<Loader2
                                size={18}
                                class="text-slate-400 animate-spin"
                            />
                        {:else if check.status === "ok"}<CheckCircle2
                                size={18}
                                class="text-emerald-500"
                            />
                        {:else}<AlertTriangle
                                size={18}
                                class={check.status === "warn"
                                    ? "text-amber-500"
                                    : "text-red-500"}
                            />{/if}
                    </div>
                    <div class="flex-1 min-w-0">
                        <div class="flex items-center gap-2 mb-0.5">
                            <span class="text-sm font-semibold text-slate-800"
                                >{check.name}</span
                            >
                            <span
                                class="px-1.5 py-0.5 rounded text-[11px] font-medium {check.status ===
                                'ok'
                                    ? 'bg-emerald-100 text-emerald-700'
                                    : check.status === 'warn'
                                      ? 'bg-amber-100 text-amber-700'
                                      : check.status === 'error'
                                        ? 'bg-red-100 text-red-700'
                                        : 'bg-slate-100 text-slate-500'}"
                            >
                                {check.status === "ok"
                                    ? "正常"
                                    : check.status === "warn"
                                      ? "需关注"
                                      : check.status === "error"
                                        ? "异常"
                                        : "检查中"}
                            </span>
                        </div>
                        <p
                            class="text-sm {check.status === 'ok'
                                ? 'text-slate-500'
                                : check.status === 'warn'
                                  ? 'text-amber-700'
                                  : 'text-red-700'}"
                        >
                            {check.message}
                        </p>
                        {#if check.fixUrl && check.status !== "ok"}
                            <a
                                href={check.fixUrl}
                                class="inline-flex items-center gap-1 mt-1.5 text-xs font-medium text-primary-600 hover:text-primary-700"
                                >{check.fixLabel || "去修复"}
                                <ExternalLink size={11} /></a
                            >
                        {/if}
                    </div>
                </div>
            {/each}
        </div>
    </div>

    <div class="card">
        <h3 class="text-lg font-bold text-slate-800 mb-4">常见问题排查指南</h3>
        <div class="space-y-3">
            <div class="p-4 rounded-lg border border-slate-200 bg-slate-50">
                <h4 class="text-sm font-semibold text-slate-800 mb-1">
                    无法根据当前数据源结构生成查询
                </h4>
                <p class="text-xs text-slate-600">
                    此错误说明 LLM 在当前数据源的表中找不到能回答问题的字段。
                </p>
                <ul
                    class="mt-2 text-xs text-slate-500 space-y-1 list-disc list-inside"
                >
                    <li>确认数据源已正确连接并启用了正确的业务数据库</li>
                    <li>在「数据源」页面点击「分析」获取最新表结构</li>
                    <li>
                        确认数据源包含业务表（如 orders、sales、products 等）
                    </li>
                    <li>在「Skills」页面为 text2sql Skill 关联正确的数据源</li>
                </ul>
            </div>
            <div class="p-4 rounded-lg border border-slate-200 bg-slate-50">
                <h4 class="text-sm font-semibold text-slate-800 mb-1">
                    SQL 必须以 SELECT 开头
                </h4>
                <p class="text-xs text-slate-600">
                    LLM 返回的内容不是合法的 SELECT 查询语句。
                </p>
                <ul
                    class="mt-2 text-xs text-slate-500 space-y-1 list-disc list-inside"
                >
                    <li>
                        LLM 提供商配置的模型可能不支持 SQL 生成，建议更换模型
                    </li>
                    <li>表结构信息格式异常，建议重新分析数据源</li>
                    <li>Skill 的 prompt_template 可能干扰了 LLM 输出</li>
                </ul>
            </div>
            <div class="p-4 rounded-lg border border-slate-200 bg-slate-50">
                <h4 class="text-sm font-semibold text-slate-800 mb-1">
                    Skills keywords 多层转义
                </h4>
                <p class="text-xs text-slate-600">
                    在 Skills 编辑页保存后 keywords 出现大量反斜杠转义。
                </p>
                <ul
                    class="mt-2 text-xs text-slate-500 space-y-1 list-disc list-inside"
                >
                    <li>此为前端解析 bug，已在最新版本中修复</li>
                    <li>
                        打开有问题的 Skill 直接保存（无需修改内容）即可自动修复
                    </li>
                </ul>
            </div>
        </div>
    </div>
</div>
