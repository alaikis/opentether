<script lang="ts">
    import { page } from "$app/stores";
    import type { ComponentType } from "svelte";

    export let items: {
        href: string;
        label: string;
        icon: ComponentType;
        exact?: boolean;
    }[] = [];
    export let collapsed = false;

    function isActive(item: { href: string; exact?: boolean }): boolean {
        const current = $page.url.pathname;
        if (item.exact) return current === item.href;
        return current.startsWith(item.href);
    }
</script>

<aside
    class="fixed left-0 top-0 z-40 h-screen bg-gradient-to-b from-primary-950 to-primary-900 text-white transition-all duration-300 flex flex-col {collapsed
        ? 'w-16'
        : 'w-64'}"
>
    <!-- Brand -->
    <div
        class="flex items-center gap-3 h-16 px-4 border-b border-white/10 shrink-0"
    >
        <div
            class="w-8 h-8 rounded-lg bg-primary-500 flex items-center justify-center text-sm font-bold shrink-0"
        >
            OT
        </div>
        {#if !collapsed}
            <span class="font-semibold text-base truncate">OpenTether</span>
        {/if}
    </div>

    <!-- Nav -->
    <nav class="flex-1 overflow-y-auto py-3 px-2 space-y-0.5">
        {#each items as item}
            <a
                href={item.href}
                class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-all duration-200
					{isActive(item)
                    ? 'bg-white/15 text-white font-medium'
                    : 'text-white/60 hover:text-white hover:bg-white/10'}"
            >
                <svelte:component this={item.icon} class="w-5 h-5 shrink-0" />
                {#if !collapsed}
                    <span class="truncate">{item.label}</span>
                {/if}
            </a>
        {/each}
    </nav>

    <!-- Footer -->
    <div class="shrink-0 border-t border-white/10 p-3">
        <button
            on:click={() => (collapsed = !collapsed)}
            class="w-full flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-white/50 hover:text-white hover:bg-white/10 transition-all text-xs"
        >
            <span
                class="i-lucide-chevron-left text-lg {collapsed
                    ? 'rotate-180'
                    : ''}">◀</span
            >
            {#if !collapsed}<span>收起菜单</span>{/if}
        </button>
    </div>
</aside>
