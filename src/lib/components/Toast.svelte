<script lang="ts">
	import { toast, type Toast } from '$lib/stores/toast.svelte';
	import { CheckCircle2, AlertCircle, Info, X } from 'lucide-svelte';

	const items = $derived(toast.items);

	function iconFor(kind: Toast['kind']) {
		if (kind === 'success') return CheckCircle2;
		if (kind === 'error') return AlertCircle;
		return Info;
	}

	function classFor(kind: Toast['kind']): string {
		if (kind === 'success') return 'preset-filled-success-500';
		if (kind === 'error') return 'preset-filled-error-500';
		return 'preset-filled-primary-500';
	}
</script>

<div
	class="pointer-events-none fixed z-50 flex flex-col gap-2 p-4
		top-0 right-0 left-0 items-center
		md:top-auto md:left-auto md:bottom-0 md:right-0 md:items-end"
	role="region"
	aria-label="Powiadomienia"
>
	{#each items as item (item.id)}
		<div
			class="pointer-events-auto flex items-start gap-3 rounded-container px-4 py-3 shadow-lg w-full max-w-sm {classFor(
				item.kind
			)}"
			role={item.kind === 'error' ? 'alert' : 'status'}
		>
			<svelte:component this={iconFor(item.kind)} size={18} class="mt-0.5 shrink-0" />
			<span class="flex-1 text-sm">{item.message}</span>
			<button
				type="button"
				class="btn-icon btn-icon-sm shrink-0"
				aria-label="Zamknij powiadomienie"
				onclick={() => toast.dismiss(item.id)}
			>
				<X size={16} />
			</button>
		</div>
	{/each}
</div>
