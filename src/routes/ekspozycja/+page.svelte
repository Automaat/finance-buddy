<script lang="ts">
	import { untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { Globe } from 'lucide-svelte';
	import MetricCard from '$lib/components/MetricCard.svelte';
	import { formatPLN, formatDate } from '$lib/utils/format';
	import { chartPalette } from '$lib/utils/theme';
	import { createChart, type ChartHandle } from '$lib/utils/charts/lifecycle';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	const report = $derived(data.report);

	// Local form state seeded from the URL-driven load; Apply pushes it back to
	// the query string so the drift band is shareable and survives reloads.
	let targetInput = $state<number | null>(untrack(() => data.targetPLNPct));
	let toleranceInput = $state<number>(untrack(() => data.tolerance));

	$effect(() => {
		targetInput = data.targetPLNPct;
		toleranceInput = data.tolerance;
	});

	let container: HTMLDivElement | undefined = $state();
	let handle: ChartHandle | null = null;

	$effect(() => {
		if (!container || report.currencies.length === 0) {
			handle?.dispose();
			handle = null;
			return;
		}
		if (!handle) handle = createChart(container);
		const pieData = report.currencies.map((c, i) => ({
			name: c.currency,
			value: c.value_pln,
			itemStyle: { color: chartPalette[i % chartPalette.length] }
		}));
		handle.chart.setOption({
			tooltip: {
				trigger: 'item',
				formatter: (p: { name: string; value: number; percent: number }) =>
					`${p.name}<br/>${formatPLN(p.value)} (${p.percent.toFixed(1)}%)`
			},
			legend: { bottom: 0 },
			series: [
				{
					type: 'pie',
					radius: ['45%', '70%'],
					avoidLabelOverlap: false,
					label: { show: true, formatter: '{b}: {d}%' },
					data: pieData
				}
			]
		});
		return () => {
			handle?.dispose();
			handle = null;
		};
	});

	function applyTarget(event: SubmitEvent) {
		event.preventDefault();
		const params = new URLSearchParams($page.url.searchParams);
		if (targetInput != null && Number.isFinite(targetInput)) {
			params.set('target_pln_pct', String(targetInput));
			params.set('tolerance', String(toleranceInput ?? 5));
		} else {
			params.delete('target_pln_pct');
			params.delete('tolerance');
		}
		const qs = params.toString();
		void goto(qs ? `/ekspozycja?${qs}` : '/ekspozycja', { keepFocus: true });
	}

	function clearTarget() {
		targetInput = null;
		void goto('/ekspozycja', { keepFocus: true });
	}
</script>

<svelte:head>
	<title>Ekspozycja walutowa - Finance Buddy</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center gap-2">
		<Globe size={28} class="text-primary-500" />
		<h1 class="h1">Ekspozycja walutowa</h1>
	</div>
	<p class="text-surface-700-300">
		Udział PLN vs walut obcych w portfelu inwestycyjnym (bez mieszkania i ROR). Ustaw docelowy
		udział PLN, aby zobaczyć dryft względem celu.
	</p>

	<form
		class="card preset-filled-surface-100-900 p-4 flex flex-wrap items-end gap-3"
		onsubmit={applyTarget}
	>
		<label class="label">
			<span class="text-xs font-semibold">Cel PLN (%)</span>
			<input
				type="number"
				min="0"
				max="100"
				step="1"
				class="input w-28"
				bind:value={targetInput}
				placeholder="—"
			/>
		</label>
		<label class="label">
			<span class="text-xs font-semibold">Tolerancja (pp)</span>
			<input
				type="number"
				min="0"
				max="50"
				step="1"
				class="input w-24"
				bind:value={toleranceInput}
			/>
		</label>
		<button type="submit" class="btn preset-filled-primary-500">Zastosuj</button>
		{#if report.drift}
			<button type="button" class="btn preset-tonal-surface" onclick={clearTarget}>Wyczyść</button>
		{/if}
	</form>

	{#if report.currencies.length === 0}
		<div
			class="card preset-filled-surface-100-900 p-8 text-center text-surface-700-300 flex flex-col items-center gap-2"
		>
			<Globe size={32} class="opacity-60" />
			<p class="font-semibold">Brak danych o ekspozycji walutowej</p>
			<p class="text-sm">Dodaj pierwszy snapshot lub aktywne konto, aby zobaczyć podział walut.</p>
		</div>
	{:else}
		<div class="grid grid-cols-2 md:grid-cols-4 gap-4">
			<MetricCard label="Udział PLN" valueText={`${report.pln_pct.toFixed(1)}%`} />
			<MetricCard label="Waluty obce" valueText={`${report.foreign_pct.toFixed(1)}%`} />
			<MetricCard label="Wartość portfela" valueText={formatPLN(report.total_pln)} />
			<MetricCard label="Snapshot" valueText={formatDate(report.snapshot_date)} />
		</div>

		{#if report.drift}
			{@const within = report.drift.within_tolerance}
			<MetricCard
				label={`Dryft względem celu ${report.drift.target_pln_pct}% PLN`}
				valueText={`${report.drift.drift_pln_pct >= 0 ? '+' : ''}${report.drift.drift_pln_pct.toFixed(1)} pp`}
				color={within ? 'green' : 'yellow'}
				size="lg"
			>
				<div class="text-sm">
					Aktualnie {report.drift.actual_pln_pct.toFixed(1)}% ·
					{within
						? `w tolerancji ±${report.drift.tolerance_pct} pp`
						: `poza pasmem ±${report.drift.tolerance_pct} pp`}
				</div>
			</MetricCard>
		{/if}

		<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
			<div class="card preset-filled-surface-100-900 p-4">
				<div
					bind:this={container}
					role="img"
					aria-label="Wykres kołowy udziału walut w portfelu"
					class="w-full h-[320px]"
				></div>
			</div>
			<div class="card preset-filled-surface-100-900 p-4 overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr class="text-left opacity-75 border-b border-surface-300-700">
							<th class="py-2 pr-3">Waluta</th>
							<th class="py-2 px-3 text-right">Wartość (PLN)</th>
							<th class="py-2 pl-3">Udział</th>
						</tr>
					</thead>
					<tbody>
						{#each report.currencies as bucket, i (bucket.currency)}
							<tr class="border-b border-surface-200-800 last:border-0">
								<td class="py-2 pr-3 font-semibold">{bucket.currency}</td>
								<td class="py-2 px-3 text-right">{formatPLN(bucket.value_pln)}</td>
								<td class="py-2 pl-3">
									<div class="flex items-center gap-2">
										<div class="flex-1 h-2 rounded-full bg-surface-200-800 overflow-hidden">
											<div
												class="h-full rounded-full"
												style="width: {Math.min(bucket.percent, 100)}%; background: {chartPalette[
													i % chartPalette.length
												]}"
											></div>
										</div>
										<span class="w-12 text-right tabular-nums">{bucket.percent.toFixed(1)}%</span>
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
</div>
