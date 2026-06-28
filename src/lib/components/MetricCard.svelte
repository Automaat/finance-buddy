<script lang="ts">
	import type { ComponentType, Snippet, SvelteComponent } from 'svelte';

	interface Props {
		label: string;
		labelHeadingLevel?: 2 | 3 | 4;
		value?: number | null;
		valueText?: string | null;
		decimals?: number;
		suffix?: string;
		signed?: boolean;
		color?: 'green' | 'blue' | 'red' | 'yellow' | 'neutral';
		icon?: ComponentType<SvelteComponent<{ size?: string | number }>>;
		size?: 'md' | 'lg';
		children?: Snippet;
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
		labelHeadingLevel,
		value,
		valueText,
		decimals = 0,
		suffix = '',
		signed = false,
		color = 'neutral',
		icon: Icon,
		size = 'md',
		children,
		tooltip,
		secondary,
		emptyHint,
		emptyHref
	}: Props = $props();

	type LabelTag = 'div' | 'h2' | 'h3' | 'h4';
	const labelTag: LabelTag = $derived(
		labelHeadingLevel === 2
			? 'h2'
			: labelHeadingLevel === 3
				? 'h3'
				: labelHeadingLevel === 4
					? 'h4'
					: 'div'
	);

	const formatter = $derived(
		new Intl.NumberFormat('pl-PL', {
			minimumFractionDigits: decimals,
			maximumFractionDigits: decimals
		})
	);

	const hasValueText = $derived(valueText != null && valueText.trim() !== '');
	const hasNumericValue = $derived(value != null && !Number.isNaN(value));
	const isEmpty = $derived(!hasValueText && !hasNumericValue);
	const roundsToZero = $derived(
		hasNumericValue && value !== 0 && Math.round(Math.abs(value as number) * 10 ** decimals) === 0
	);
	const displayValue = $derived.by(() => {
		if (hasValueText) return valueText ?? '';
		if (isEmpty) return '—';
		const numeric = value as number;
		if (!signed) return formatter.format(numeric) + suffix;
		if (numeric === 0 || roundsToZero) return formatter.format(0) + suffix;
		const sign = numeric > 0 ? '+' : '−';
		return sign + formatter.format(Math.abs(numeric)) + suffix;
	});
	const showHint = $derived(isEmpty && emptyHint != null);
	const valueSizeClass = $derived(size === 'lg' ? 'text-3xl' : 'text-2xl');

	const colorClass = $derived(
		{
			green: 'text-success-600-400',
			red: 'text-error-600-400',
			blue: 'text-primary-600-400',
			yellow: 'text-warning-600-400',
			neutral: 'text-surface-950-50'
		}[color]
	);
	const signedValueClass = $derived(
		signed && hasNumericValue && !roundsToZero && value !== 0
			? (value as number) > 0
				? 'text-success-600-400'
				: 'text-error-600-400'
			: 'text-surface-950-50'
	);
	const valueClass = $derived(signed ? signedValueClass : colorClass);
</script>

<div class="card preset-filled-surface-100-900 p-4 space-y-1">
	{#if tooltip}
		<svelte:element
			this={labelTag}
			class="metric-card-label text-sm opacity-75 cursor-help underline decoration-dotted flex items-center gap-2"
			title={tooltip}
		>
			{#if Icon}
				<Icon size={16} />
			{/if}
			{label}
		</svelte:element>
	{:else}
		<svelte:element
			this={labelTag}
			class="metric-card-label text-sm opacity-75 flex items-center gap-2"
		>
			{#if Icon}
				<Icon size={16} />
			{/if}
			{label}
		</svelte:element>
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
		<div class="{valueSizeClass} font-bold {valueClass}">{displayValue}</div>
		{#if secondary}
			<div class="text-xs text-surface-600-400">{secondary}</div>
		{/if}
		{@render children?.()}
	{/if}
</div>

<style>
	.metric-card-label {
		margin: 0;
		font-weight: inherit;
	}
</style>
