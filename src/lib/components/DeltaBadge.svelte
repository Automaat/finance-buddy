<script lang="ts">
	import { TrendingUp, TrendingDown, Minus } from 'lucide-svelte';
	import { formatSignedPLN, formatSignedPercent } from '$lib/utils/format';

	interface Props {
		label: string;
		absolute: number | null;
		percentage: number | null;
		formulaTitle: string;
	}

	let { label, absolute, percentage, formulaTitle }: Props = $props();

	const isPositive = $derived(absolute != null && absolute > 0);
	const isNegative = $derived(absolute != null && absolute < 0);
	const colorClass = $derived(
		absolute == null
			? 'text-surface-700-300'
			: isPositive
				? 'text-success-600-400'
				: isNegative
					? 'text-error-600-400'
					: 'text-surface-700-300'
	);
</script>

<span
	class="inline-flex items-center gap-1 text-xs font-semibold {colorClass}"
	title={formulaTitle}
	aria-label="Δ {label}"
	data-testid="delta-badge-{label.toLowerCase()}"
>
	<span class="opacity-75">Δ {label}</span>
	{#if absolute == null}
		<span>—</span>
	{:else}
		{#if isPositive}
			<TrendingUp size={12} aria-hidden="true" />
		{:else if isNegative}
			<TrendingDown size={12} aria-hidden="true" />
		{:else}
			<Minus size={12} aria-hidden="true" />
		{/if}
		<span>{formatSignedPLN(absolute)}</span>
		{#if percentage != null}
			<span class="opacity-75">({formatSignedPercent(percentage)})</span>
		{/if}
	{/if}
</span>
