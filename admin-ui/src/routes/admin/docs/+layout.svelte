<script lang="ts">
    import Sidebar from "$lib/components/Sidebar.svelte";
    import Header from "$lib/components/Header.svelte";
    import { sidebarCollapsed } from "$lib/stores/ui";
    import { BookOpen, Code2, Users, Bot } from "lucide-svelte";

    const docsNav = [
        { href: "/admin/docs", label: "文档首页", icon: BookOpen, exact: true },
        { href: "/admin/docs/dev", label: "开发者文档", icon: Code2 },
        { href: "/admin/docs/user", label: "用户指南", icon: Users },
        { href: "/admin/docs/agent", label: "智能体指南", icon: Bot },
    ];
</script>

<div class="flex min-h-screen">
    <Sidebar items={docsNav} />
    <div
        class="flex-1 flex flex-col transition-all duration-300"
        class:ml-16={$sidebarCollapsed}
        class:ml-64={!$sidebarCollapsed}
    >
        <Header title="文档中心" />
        <main class="flex-1 p-6 max-w-4xl mx-auto w-full">
            <slot />
        </main>
    </div>
</div>
