<script lang="ts">
	interface Props {
		label: string;
		value: number | null | undefined;
		decimals?: number;
		suffix?: string;
		color?: 'green' | 'blue' | 'red' | 'yellow' | 'neutral';
	}

	let { label, value, decimals = 0, suffix = '', color = 'neutral' }: Props = $props();

	const formatter = $derived(
		new Intl.NumberFormat('pl-PL', {
			minimumFractionDigits: decimals,
			maximumFractionDigits: decimals
		})
	);

	const displayValue = $derived(
		value == null || Number.isNaN(value) ? '—' : formatter.format(value) + suffix
	);

	const valueClass = $derived(
		{
			green: 'text-success-600-400',
			red: 'text-error-600-400',
			blue: 'text-primary-600-400',
			yellow: 'text-warning-600-400',
			neutral: 'text-surface-950-50'
		}[color]
	);
</script>

<div class="card preset-filled-surface-100-900 p-4 space-y-1">
	<div class="text-sm opacity-75">{label}</div>
	<div class="text-2xl font-bold {valueClass}">{displayValue}</div>
</div>
