<script lang="ts">
    import { toasts } from "$lib/stores/toast";

    const colors: Record<string, string> = {
        success: "bg-emerald-50 border-emerald-200 text-emerald-800",
        error: "bg-red-50 border-red-200 text-red-800",
        info: "bg-primary-50 border-primary-200 text-primary-800",
    };
</script>

<div class="fixed top-4 right-4 z-50 flex flex-col gap-2">
    {#each $toasts as t (t.id)}
        <div
            class="px-4 py-3 rounded-lg border shadow-lg text-sm font-medium {colors[
                t.type
            ] ?? colors.info}"
        >
            {t.message}
        </div>
    {/each}
</div>
