<script lang="ts">
	export let label: string;
	export let value: number;
	export let decimals: number = 0;
	export let suffix: string = '';
	export let color: 'blue' | 'green' | 'red' = 'blue';

	function formatValue(val: number, dec: number): string {
		return val.toLocaleString('pl-PL', {
			minimumFractionDigits: dec,
			maximumFractionDigits: dec
		});
	}
</script>

<div class="metric-card {color}">
	<div class="label">{label}</div>
	<div class="value">
		{formatValue(value, decimals)}{#if suffix}{suffix}{/if}
	</div>
</div>

<style>
	.metric-card {
		padding: var(--size-4);
		border-radius: var(--radius-2);
		background: var(--surface-2);
		border: 1px solid var(--surface-3);
	}

	.metric-card.blue {
		border-color: var(--blue-6);
	}

	.metric-card.green {
		border-color: var(--green-6);
	}

	.metric-card.red {
		border-color: var(--red-6);
	}

	.label {
		font-size: var(--font-size-0);
		color: var(--text-2);
		margin-bottom: var(--size-2);
	}

	.value {
		font-size: var(--font-size-4);
		font-weight: var(--font-weight-7);
		color: var(--text-1);
	}

	.metric-card.blue .value {
		color: var(--blue-6);
	}

	.metric-card.green .value {
		color: var(--green-6);
	}

	.metric-card.red .value {
		color: var(--red-6);
	}
</style>
