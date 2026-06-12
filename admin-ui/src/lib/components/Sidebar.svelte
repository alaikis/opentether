<script lang="ts">
    import { page } from "$app/stores";
    import { sidebarCollapsed } from "$lib/stores/ui";
    import type { ComponentType } from "svelte";
    import { ChevronDown } from "lucide-svelte";

    export let items: {
        href: string;
        label: string;
        icon: ComponentType;
        exact?: boolean;
        group?: string;
    }[] = [];

    type NavItem = {
        href: string;
        label: string;
        icon: ComponentType;
        exact?: boolean;
    };
    type NavGroup = { id: string; label: string; items: NavItem[] };

    $: groups = (() => {
        const result: NavGroup[] = [];
        const ungrouped: NavItem[] = [];
        const seen = new Set<string>();
        for (const it of items) {
            if (it.group) {
                if (!seen.has(it.group)) {
                    seen.add(it.group);
                    result.push({ id: it.group, label: it.group, items: [] });
                }
                result
                    .find((g) => g.id === it.group)!
                    .items.push({
                        href: it.href,
                        label: it.label,
                        icon: it.icon,
                        exact: it.exact,
                    });
            } else {
                ungrouped.push({
                    href: it.href,
                    label: it.label,
                    icon: it.icon,
                    exact: it.exact,
                });
            }
        }
        return { groups: result, ungrouped };
    })();

    let openGroup = "";
    let initialRoute = "";
    $: if (initialRoute === "" && $page.url.pathname)
        initialRoute = $page.url.pathname;
    $: if (initialRoute && initialRoute === $page.url.pathname) {
        const need = groups.groups.find((g) =>
            g.items.some((it) => active(it)),
        );
        if (need) openGroup = need.id;
    }

    function active(item: { href: string; exact?: boolean }): boolean {
        const cur = $page.url.pathname;
        return item.exact ? cur === item.href : cur.startsWith(item.href);
    }
    function toggle(id: string) {
        openGroup = openGroup === id ? "" : id;
    }

    let popupGroup = "";
    let popupTimer: ReturnType<typeof setTimeout> | null = null;
    function showPopup(id: string) {
        if (popupTimer) clearTimeout(popupTimer);
        popupGroup = id;
    }
    function hidePopup() {
        popupTimer = setTimeout(() => (popupGroup = ""), 200);
    }
    function keepPopup() {
        if (popupTimer) clearTimeout(popupTimer);
    }
</script>

<aside
    class="fixed left-0 top-0 z-40 h-screen flex flex-col bg-gradient-to-b from-blue-950 via-indigo-950 to-slate-900 text-white transition-all duration-300 {$sidebarCollapsed
        ? 'w-16'
        : 'w-60'}"
>
    <!-- Logo -->
    <div
        class="flex items-center gap-3 h-16 px-4 border-b border-blue-800/40 shrink-0"
    >
        <div
            class="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-400 to-blue-600 flex items-center justify-center text-white text-sm font-bold shrink-0 shadow-lg shadow-blue-500/20"
        >
            OT
        </div>
        {#if !$sidebarCollapsed}
            <span class="font-semibold text-base text-blue-50 truncate"
                >OpenTether</span
            >
        {/if}
    </div>

    <!-- Nav -->
    <nav class="flex-1 overflow-y-auto py-3 px-2 space-y-0.5">
        {#each groups.groups as group}
            {#if !$sidebarCollapsed}
                <button
                    class="flex items-center gap-2 w-full px-2.5 py-2 rounded-lg text-[12px] font-semibold text-blue-300/50 hover:text-blue-200 hover:bg-blue-800/30 transition-colors"
                    on:click={() => toggle(group.id)}
                >
                    <span class="flex-1 text-left uppercase tracking-wider"
                        >{group.label}</span
                    >
                    <ChevronDown
                        class="w-3 h-3 transition-transform duration-200 {openGroup ===
                        group.id
                            ? 'rotate-180'
                            : ''}"
                    />
                </button>
                <div
                    class="overflow-hidden transition-all duration-200 {openGroup ===
                    group.id
                        ? 'max-h-80 opacity-100'
                        : 'max-h-0 opacity-0'}"
                >
                    <div class="space-y-0.5 pt-0.5 pb-1">
                        {#each group.items as item}
                            <a
                                href={item.href}
                                class="flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-sm transition-colors {active(
                                    item,
                                )
                                    ? 'bg-blue-500/20 text-blue-300 font-medium shadow-sm shadow-blue-500/5'
                                    : 'text-blue-200/50 hover:text-blue-100 hover:bg-blue-800/30'}"
                            >
                                <svelte:component
                                    this={item.icon}
                                    class="w-4 h-4 shrink-0"
                                />
                                <span class="truncate">{item.label}</span>
                            </a>
                        {/each}
                    </div>
                </div>
            {:else}
                <div
                    class="relative"
                    on:mouseenter={() => showPopup(group.id)}
                    on:mouseleave={hidePopup}
                >
                    <div class="mx-3 my-3 border-t border-blue-800/30"></div>
                    {#if popupGroup === group.id}
                        <div
                            class="absolute left-14 -top-2 z-50 bg-white rounded-xl shadow-xl border border-slate-200 py-2 min-w-[170px]"
                            on:mouseenter={keepPopup}
                            on:mouseleave={hidePopup}
                        >
                            <div
                                class="px-3 py-1.5 text-[11px] font-semibold text-slate-400 uppercase tracking-wider"
                            >
                                {group.label}
                            </div>
                            {#each group.items as item}
                                <a
                                    href={item.href}
                                    class="flex items-center gap-2.5 px-3 py-2 text-sm transition-colors {active(
                                        item,
                                    )
                                        ? 'bg-blue-50 text-blue-600 font-medium'
                                        : 'text-slate-600 hover:text-slate-900 hover:bg-slate-50'}"
                                >
                                    <svelte:component
                                        this={item.icon}
                                        class="w-4 h-4 shrink-0"
                                    />
                                    <span class="truncate">{item.label}</span>
                                </a>
                            {/each}
                        </div>
                    {/if}
                </div>
            {/if}
        {/each}

        {#each groups.ungrouped as item}
            <a
                href={item.href}
                title={$sidebarCollapsed ? item.label : undefined}
                class="flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-sm transition-colors {active(
                    item,
                )
                    ? 'bg-blue-500/20 text-blue-300 font-medium'
                    : 'text-blue-200/50 hover:text-blue-100 hover:bg-blue-800/30'}"
            >
                <svelte:component this={item.icon} class="w-5 h-5 shrink-0" />
                {#if !$sidebarCollapsed}<span class="truncate"
                        >{item.label}</span
                    >{/if}
            </a>
        {/each}
    </nav>

    <!-- 折叠按钮 -->
    <div class="shrink-0 border-t border-blue-800/40 p-3">
        <button
            on:click={() => sidebarCollapsed.toggle()}
            class="w-full flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-blue-300/40 hover:text-blue-200 hover:bg-blue-800/30 transition-colors text-xs"
        >
            <span
                class="text-sm transition-transform duration-300 {$sidebarCollapsed
                    ? 'rotate-180'
                    : ''}">◀</span
            >
            {#if !$sidebarCollapsed}<span>收起菜单</span>{/if}
        </button>
    </div>
</aside>
