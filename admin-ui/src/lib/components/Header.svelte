<script lang="ts">
    export let title = "仪表盘";
    import { BookOpen, LogOut } from "lucide-svelte";
    import { auth } from "$lib/stores/auth";
    import { onMount } from "svelte";

    let userName = "管理员";
    let userEmail = "";

    onMount(() => {
        const user = auth.getUser();
        if (user?.name) userName = user.name;
        if (user?.email) userEmail = user.email;
    });

    function handleLogout() {
        auth.logout();
        window.location.href = "/open/u/login";
    }
</script>

<header
    class="h-16 bg-white border-b border-slate-200 flex items-center justify-between px-6 sticky top-0 z-30"
>
    <div class="flex items-center gap-4">
        <h1 class="text-lg font-semibold text-slate-800">{title}</h1>
    </div>

    <div class="flex items-center gap-3">
        <a
            href="/admin/docs"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-slate-500 hover:text-primary-600 hover:bg-primary-50 transition-colors"
        >
            <BookOpen class="w-4 h-4" />
            <span class="hidden lg:inline">文档</span>
        </a>

        <div class="flex items-center gap-3 pl-4 border-l border-slate-200">
            <div
                class="w-8 h-8 rounded-full bg-primary-100 flex items-center justify-center"
            >
                <span class="text-sm font-semibold text-primary-700"
                    >{userName[0] || "A"}</span
                >
            </div>
            <div class="hidden sm:block text-sm">
                <div class="font-medium text-slate-700">{userName}</div>
                <div class="text-xs text-slate-400">
                    {userEmail || "admin@opentether"}
                </div>
            </div>
        </div>

        <button
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-slate-400 hover:text-red-600 hover:bg-red-50 transition-colors border-l border-slate-200 pl-4"
            on:click={handleLogout}
        >
            <LogOut class="w-4 h-4" />
            <span class="hidden sm:inline">退出</span>
        </button>
    </div>
</header>
