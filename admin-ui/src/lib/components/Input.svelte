<script lang="ts">
    import { cn } from "$lib/utils/cn";

    export let label = "";
    export let type = "text";
    export let placeholder = "";
    export let value = "";
    export let error = "";
    export let required = false;
    export let disabled = false;

    function handleInput(e: Event) {
        value = (e.target as HTMLInputElement).value;
    }
</script>

<div class="space-y-1.5">
    {#if label}
        <label
            for={label.replace(/\s/g, "-").toLowerCase()}
            class="block text-sm font-medium text-slate-700"
        >
            {label}
            {#if required}<span class="text-red-500 ml-0.5">*</span>{/if}
        </label>
    {/if}
    <input
        {type}
        id={label.replace(/\s/g, "-").toLowerCase()}
        {placeholder}
        {required}
        {disabled}
        on:input={handleInput}
        on:blur
        class={cn(
            "w-full px-3 py-2 border rounded-lg text-sm transition-all duration-200",
            "focus:outline-none focus:ring-2 focus:ring-primary-200 focus:border-primary-400",
            "disabled:bg-slate-50 disabled:text-slate-400",
            error
                ? "border-red-300 focus:ring-red-200 focus:border-red-400"
                : "border-slate-200",
        )}
    />
    {#if error}
        <p class="text-xs text-red-500">{error}</p>
    {/if}
</div>
