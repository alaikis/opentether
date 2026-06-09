<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api/client";
    import {
        LayoutDashboard,
        Users,
        UserCog,
        Cpu,
        Database,
        Zap,
        Calendar,
        MessageSquare,
        ScrollText,
    } from "lucide-svelte";

    interface Stats {
        users: number;
        groups: number;
        providers: number;
        datasources: number;
        skills: number;
        tasks: number;
        imConfigs: number;
    }

    let stats: Stats = {
        users: 0,
        groups: 0,
        providers: 0,
        datasources: 0,
        skills: 0,
        tasks: 0,
        imConfigs: 0,
    };
    let loading = true;
    let error = "";

    const cards = [
        {
            href: "/admin/users",
            icon: Users,
            color: "bg-blue-50 text-blue-600",
            title: "用户管理",
            desc: "管理系统用户、权限和角色",
            key: "users" as keyof Stats,
        },
        {
            href: "/admin/groups",
            icon: UserCog,
            color: "bg-emerald-50 text-emerald-600",
            title: "用户组管理",
            desc: "管理用户组和数据权限",
            key: "groups" as keyof Stats,
        },
        {
            href: "/admin/providers",
            icon: Cpu,
            color: "bg-amber-50 text-amber-600",
            title: "LLM 提供商",
            desc: "配置 OpenAI、Azure、Anthropic",
            key: "providers" as keyof Stats,
        },
        {
            href: "/admin/datasources",
            icon: Database,
            color: "bg-indigo-50 text-indigo-600",
            title: "数据源管理",
            desc: "配置数据库、API 等数据源",
            key: "datasources" as keyof Stats,
        },
        {
            href: "/admin/skills",
            icon: Zap,
            color: "bg-pink-50 text-pink-600",
            title: "Skills 配置",
            desc: "管理 AI 技能和执行器",
            key: "skills" as keyof Stats,
        },
        {
            href: "/admin/tasks",
            icon: Calendar,
            color: "bg-cyan-50 text-cyan-600",
            title: "定时任务",
            desc: "配置和管理定时执行任务",
            key: "tasks" as keyof Stats,
        },
        {
            href: "/admin/im",
            icon: MessageSquare,
            color: "bg-purple-50 text-purple-600",
            title: "IM 配置",
            desc: "企业微信、飞书、钉钉集成",
            key: "imConfigs" as keyof Stats,
        },
        {
            href: "/admin/logs",
            icon: ScrollText,
            color: "bg-slate-100 text-slate-600",
            title: "系统日志",
            desc: "审计日志和请求日志",
            key: null,
        },
    ];

    async function loadStats(key: keyof Stats, endpoint: string) {
        try {
            const data = await api.get<any[]>(endpoint);
            stats[key] = Array.isArray(data)
                ? data.length
                : data?.length || data?.total || 0;
        } catch {
            stats[key] = 0;
        }
    }

    onMount(async () => {
        loading = true;
        error = "";
        try {
            await Promise.all([
                loadStats("users", "/admin/users"),
                loadStats("groups", "/admin/groups"),
                loadStats("providers", "/admin/providers"),
                loadStats("datasources", "/admin/datasources"),
                loadStats("skills", "/admin/skills"),
                loadStats("tasks", "/admin/tasks"),
                loadStats("imConfigs", "/admin/im/configs"),
            ]);
        } catch (e: any) {
            error = e.message || "加载统计数据失败";
        } finally {
            loading = false;
        }
    });
</script>

<svelte:head><title>仪表盘 - OpenTether</title></svelte:head>

<div class="mb-8">
    <h2 class="text-2xl font-bold text-slate-800">欢迎回来 👋</h2>
    <p class="text-slate-500 mt-1">OpenTether 企业级智能体管理平台</p>
</div>

{#if error}
    <div
        class="mb-6 p-4 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm"
    >
        {error}
    </div>
{/if}

<div
    class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5"
>
    {#each cards as card}
        <a href={card.href} class="card group relative">
            <div
                class="w-11 h-11 rounded-xl {card.color} flex items-center justify-center text-xl mb-4"
            >
                <svelte:component this={card.icon} size={22} />
            </div>
            <div class="flex items-center gap-2 mb-2">
                <h3 class="font-semibold text-slate-800">{card.title}</h3>
            </div>
            <p class="text-sm text-slate-500">{card.desc}</p>
            {#if card.key !== null}
                <div class="absolute top-4 right-4">
                    {#if loading}
                        <span
                            class="w-6 h-6 rounded-full bg-slate-100 animate-pulse"
                        />
                    {:else}
                        <span class="text-2xl font-bold text-slate-300"
                            >{stats[card.key]}</span
                        >
                    {/if}
                </div>
            {/if}
        </a>
    {/each}
</div>
