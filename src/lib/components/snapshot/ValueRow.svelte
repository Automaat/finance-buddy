<script lang="ts">
	interface Props {
		inputId: string;
		label: string;
		meta?: string;
		value: number;
		onRemove: () => void;
	}

	let { inputId, label, meta = '', value = $bindable(), onRemove }: Props = $props();
</script>

<div class="flex gap-2 items-end mb-4">
	<div class="flex flex-col gap-1 flex-1">
		<label for={inputId} class="text-sm font-semibold">
			{label}
			{#if meta}
				<span class="text-surface-600-400 font-normal">{meta}</span>
			{/if}
		</label>
		<input id={inputId} type="number" step="0.01" bind:value placeholder="0.00" class="input" />
	</div>
	<button
		type="button"
		class="btn-icon btn-icon-sm preset-tonal-error shrink-0"
		onclick={onRemove}
		title="Usuń pole"
	>
		×
	</button>
</div>
