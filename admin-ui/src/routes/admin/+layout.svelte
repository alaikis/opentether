<script lang="ts">
    // (app) 路由组 - 认证守卫 + 默认 Admin 侧边栏
    import { auth, isAuthenticated } from "$lib/stores/auth";
    import { sidebarCollapsed } from "$lib/stores/ui";
    import { page } from "$app/stores";
    import { goto } from "$app/navigation";
    import { onMount } from "svelte";
    import Sidebar from "$lib/components/Sidebar.svelte";
    import Header from "$lib/components/Header.svelte";
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
        Settings,
    } from "lucide-svelte";

    const adminNav = [
        { href: "/admin", label: "仪表盘", icon: LayoutDashboard, exact: true },
        {
            href: "/admin/providers",
            label: "LLM 提供商",
            icon: Cpu,
            required: true,
        },
        { href: "/admin/users", label: "用户管理", icon: Users },
        { href: "/admin/groups", label: "用户组", icon: UserCog },
        { href: "/admin/datasources", label: "数据源", icon: Database },
        { href: "/admin/skills", label: "Skills", icon: Zap },
        { href: "/admin/tasks", label: "定时任务", icon: Calendar },
        { href: "/admin/im", label: "IM 配置", icon: MessageSquare },
        { href: "/admin/logs", label: "系统日志", icon: ScrollText },
        { href: "/admin/settings", label: "系统设置", icon: Settings },
    ];

    async function checkLLMConfigured(): Promise<boolean> {
        try {
            const { api } = await import("$lib/api/client");
            const data = await api.get<any>("/admin/providers");
            const enabledProviders = (data.providers || data || []).filter(
                (p: any) => p.enabled,
            );
            return enabledProviders.length > 0;
        } catch {
            return false;
        }
    }

    $: pageTitle =
        adminNav.find((i) =>
            i.exact
                ? $page.url.pathname === i.href
                : $page.url.pathname.startsWith(i.href),
        )?.label || "管理后台";

    onMount(async () => {
        auth.checkAndSync();
        // 等 store 更新，最多重试 5 次
        for (let i = 0; i < 5; i++) {
            await new Promise((resolve) => setTimeout(resolve, 100));
            if ($isAuthenticated) break;
            auth.checkAndSync();
        }

        if (!$isAuthenticated) {
            goto("/open/u/login");
            return;
        }

        const currentPath = $page.url.pathname;
        if (!currentPath.startsWith("/admin/providers")) {
            const llmConfigured = await checkLLMConfigured();
            if (!llmConfigured) {
                goto("/admin/providers?reason=llm_required");
            }
        }
    });
</script>

{#if !$page.url.pathname.startsWith("/admin/user/") && !$page.url.pathname.startsWith("/admin/docs") && $page.url.pathname !== "/admin/user"}
    <div class="flex min-h-screen">
        <Sidebar items={adminNav} />
        <div
            class="flex-1 flex flex-col transition-all duration-300"
            class:ml-16={$sidebarCollapsed}
            class:ml-64={!$sidebarCollapsed}
        >
            <Header title={pageTitle} />
            <main class="flex-1 p-6">
                <slot />
            </main>
        </div>
    </div>
{:else}
    <slot />
{/if}
