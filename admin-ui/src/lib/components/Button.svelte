<script lang="ts">
	import { cn } from '$lib/utils/cn';

	export let variant: 'primary' | 'secondary' | 'danger' | 'ghost' = 'primary';
	export let size: 'sm' | 'md' | 'lg' = 'md';
	export let disabled = false;
	export let loading = false;
	export let type: 'button' | 'submit' = 'button';

	const base = 'inline-flex items-center justify-center gap-2 font-medium rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed';

	const variants = {
		primary: 'bg-primary-600 text-white hover:bg-primary-700 focus:ring-primary-500 shadow-sm',
		secondary: 'bg-slate-100 text-slate-700 hover:bg-slate-200 focus:ring-slate-400',
		danger: 'bg-red-600 text-white hover:bg-red-700 focus:ring-red-500 shadow-sm',
		ghost: 'bg-transparent text-slate-600 hover:bg-slate-100 focus:ring-slate-400',
	};

	const sizes = {
		sm: 'px-3 py-1.5 text-xs',
		md: 'px-4 py-2 text-sm',
		lg: 'px-6 py-3 text-base',
	};
</script>

<button {type} {disabled} class={cn(base, variants[variant], sizes[size])} on:click>
	{#if loading}
		<svg class="animate-spin h-4 w-4" viewBox="0 0 24 24">
			<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
			<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
		</svg>
	{/if}
	<slot />
</button>
