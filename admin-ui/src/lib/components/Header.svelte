<script lang="ts">
    export let title = "仪表盘";
    import { BookOpen } from "lucide-svelte";
</script>

<header
    class="h-16 bg-white border-b border-slate-200 flex items-center justify-between px-6 sticky top-0 z-30"
>
    <div class="flex items-center gap-4">
        <h1 class="text-lg font-semibold text-slate-800">{title}</h1>
    </div>

    <div class="flex items-center gap-3">
        <!-- 文档入口 -->
        <a
            href="/admin/docs"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-slate-500 hover:text-primary-600 hover:bg-primary-50 transition-colors"
            title="文档中心"
        >
            <BookOpen class="w-4 h-4" />
            <span class="hidden lg:inline">文档</span>
        </a>

        <!-- Notification -->
        <button
            class="relative p-2 rounded-lg hover:bg-slate-100 transition-colors text-slate-500 hover:text-slate-700"
        >
            <svg
                class="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
            >
                <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
                />
            </svg>
            <span
                class="absolute top-1.5 right-1.5 w-2 h-2 rounded-full bg-red-500"
            ></span>
        </button>

        <!-- User -->
        <div class="flex items-center gap-3 pl-4 border-l border-slate-200">
            <div
                class="w-8 h-8 rounded-full bg-primary-100 flex items-center justify-center"
            >
                <span class="text-sm font-semibold text-primary-700">A</span>
            </div>
            <div class="hidden sm:block text-sm">
                <div class="font-medium text-slate-700">管理员</div>
                <div class="text-xs text-slate-400">admin@opentether</div>
            </div>
        </div>
    </div>
</header>
