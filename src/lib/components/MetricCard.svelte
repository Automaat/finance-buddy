<script lang="ts">
	interface Props {
		label: string;
		value: number | null | undefined;
		decimals?: number;
		suffix?: string;
		signed?: boolean;
		// Optional explanation shown as a native tooltip on the label (and a
		// subtle dotted underline as the affordance), for cryptic metrics.
		tooltip?: string;
		// Optional muted line beneath the value (e.g. a mortgage's months/years).
		secondary?: string;
		// When value is null/undefined, render this hint + optional link instead
		// of a bare em-dash, so a missing metric points the user at how to fill it.
		emptyHint?: string;
		emptyHref?: string;
	}

	let {
		label,
		value,
		decimals = 0,
		suffix = '',
		signed = false,
		tooltip,
		secondary,
		emptyHint,
		emptyHref
	}: Props = $props();

	const formatter = $derived(
		new Intl.NumberFormat('pl-PL', {
			minimumFractionDigits: decimals,
			maximumFractionDigits: decimals
		})
	);

	const isEmpty = $derived(value == null || Number.isNaN(value));
	// True when a non-zero value rounds to zero at the current decimal
	// precision, preventing misleading "−0,00" display and error coloring.
	const roundsToZero = $derived(
		!isEmpty && value !== 0 && Math.round(Math.abs(value as number) * 10 ** decimals) === 0
	);
	const displayValue = $derived.by(() => {
		if (isEmpty) return '—';
		if (!signed) return formatter.format(value as number) + suffix;
		if (value === 0 || roundsToZero) return formatter.format(0) + suffix;
		const sign = (value as number) > 0 ? '+' : '−';
		return sign + formatter.format(Math.abs(value as number)) + suffix;
	});
	const showHint = $derived(isEmpty && emptyHint != null);

	const valueClass = $derived(
		signed && !isEmpty && value !== 0 && !roundsToZero
			? (value as number) > 0
				? 'text-success-600-400'
				: 'text-error-600-400'
			: 'text-surface-950-50'
	);
</script>

<div class="card preset-filled-surface-100-900 p-4 space-y-1">
	{#if tooltip}
		<div class="text-sm opacity-75 cursor-help underline decoration-dotted" title={tooltip}>
			{label}
		</div>
	{:else}
		<div class="text-sm opacity-75">{label}</div>
	{/if}

	{#if showHint}
		<div class="text-sm text-surface-600-400">
			{#if emptyHref}
				<a href={emptyHref} class="underline">{emptyHint}</a>
			{:else}
				{emptyHint}
			{/if}
		</div>
	{:else}
		<div class="text-2xl font-bold {valueClass}">{displayValue}</div>
		{#if secondary}
			<div class="text-xs text-surface-600-400">{secondary}</div>
		{/if}
	{/if}
</div>
