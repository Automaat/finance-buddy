<script lang="ts">
	import type { ComponentType, Snippet, SvelteComponent } from 'svelte';

	interface Props {
		label: string;
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

	const formatter = $derived(
		new Intl.NumberFormat('pl-PL', {
			minimumFractionDigits: decimals,
			maximumFractionDigits: decimals
		})
	);

	const hasValueText = $derived(valueText != null && valueText.trim() !== '');
	const isEmpty = $derived(!hasValueText && (value == null || Number.isNaN(value)));
	// True when a non-zero value rounds to zero at the current decimal
	// precision, preventing misleading "−0,00" display and error coloring.
	const roundsToZero = $derived(
		!hasValueText &&
			!isEmpty &&
			value !== 0 &&
			Math.round(Math.abs(value as number) * 10 ** decimals) === 0
	);
	const displayValue = $derived.by(() => {
		if (hasValueText) return valueText;
		if (isEmpty) return '—';
		if (!signed) return formatter.format(value as number) + suffix;
		if (value === 0 || roundsToZero) return formatter.format(0) + suffix;
		const sign = (value as number) > 0 ? '+' : '−';
		return sign + formatter.format(Math.abs(value as number)) + suffix;
	});
	const showHint = $derived(isEmpty && emptyHint != null);
	const valueSizeClass = $derived(size === 'lg' ? 'text-3xl' : 'text-2xl');

	const valueClass = $derived(
		color !== 'neutral'
			? {
					green: 'text-success-600-400',
					red: 'text-error-600-400',
					blue: 'text-primary-600-400',
					yellow: 'text-warning-600-400',
					neutral: 'text-surface-950-50'
				}[color]
			: signed && !hasValueText && !isEmpty && value !== 0 && !roundsToZero
				? (value as number) > 0
					? 'text-success-600-400'
					: 'text-error-600-400'
				: 'text-surface-950-50'
	);
</script>

<div class="card preset-filled-surface-100-900 p-4 space-y-1">
	{#if tooltip}
		<div
			class="text-sm opacity-75 cursor-help underline decoration-dotted flex items-center gap-2"
			title={tooltip}
		>
			{#if Icon}
				<Icon size={16} />
			{/if}
			{label}
		</div>
	{:else}
		<div class="text-sm opacity-75 flex items-center gap-2">
			{#if Icon}
				<Icon size={16} />
			{/if}
			{label}
		</div>
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
