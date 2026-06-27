<script lang="ts">
	import { onMount } from 'svelte';
	import { resolveApiUrl } from '$lib/api';
	import { formatPLN } from '$lib/utils/format';
	import { createChart, type ChartHandle } from '$lib/utils/charts/lifecycle';
	import { applyMobileChartTweaks } from '$lib/utils/charts/responsive';
	import { isMobile } from '$lib/utils/viewport';
	import { chartPalette } from '$lib/utils/theme';
	import { Globe } from 'lucide-svelte';

	interface CurrencyBucket {
		currency: string;
		value_pln: number;
		percent: number;
	}

	interface DriftBand {
		target_pln_pct: number;
		actual_pln_pct: number;
		drift_pln_pct: number;
		within_tolerance: boolean;
		tolerance_pct: number;
	}

	interface Report {
		currencies: CurrencyBucket[];
		total_pln: number;
		pln_pct: number;
		foreign_pct: number;
		snapshot_date: string;
		drift?: DriftBand;
	}

	let targetPLNPct = $state<number | null>(null);
	let tolerance = $state(5);
	let report = $state<Report | null>(null);
	let loading = $state(true);
	let error = $state('');
	let container: HTMLDivElement | undefined = $state();
	let handle: ChartHandle | null = null;
	// Cancels an in-flight request when a new one starts or the component is
	// destroyed, so a late response can't write state onto an unmounted widget.
	let inFlight: AbortController | null = null;

	async function load() {
		inFlight?.abort();
		const controller = new AbortController();
		inFlight = controller;
		loading = true;
		error = '';
		try {
			const apiUrl = resolveApiUrl();
			const params = new URLSearchParams();
			if (targetPLNPct != null && !Number.isNaN(targetPLNPct)) {
				params.set('target_pln_pct', String(targetPLNPct));
				params.set('tolerance', String(tolerance));
			}
			const qs = params.toString();
			const url = `${apiUrl}/api/exposure/currency${qs ? `?${qs}` : ''}`;
			const res = await fetch(url, { signal: controller.signal });
			if (!res.ok) throw new Error(`Pobranie ekspozycji nieudane: ${res.statusText}`);
			report = await res.json();
		} catch (err) {
			if (controller.signal.aborted) return;
			if (err instanceof Error) error = err.message;
		} finally {
			if (!controller.signal.aborted) loading = false;
		}
	}

	onMount(() => {
		void load();
		return () => inFlight?.abort();
	});

	$effect(() => {
		if (!container) {
			handle?.dispose();
			handle = null;
			return;
		}
		if (!report) return;
		if (!handle) handle = createChart(container);
		const data = report.currencies.map((c, i) => ({
			name: c.currency,
			value: c.value_pln,
			itemStyle: { color: chartPalette[i % chartPalette.length] }
		}));
		handle.chart.setOption(
			applyMobileChartTweaks(
				{
					tooltip: {
						trigger: 'item',
						formatter: (params) => {
							const p = Array.isArray(params) ? params[0] : params;
							const pct = (p as { percent?: number }).percent ?? 0;
							return `${p.name}<br/>${formatPLN(p.value as number)} (${pct.toFixed(1)}%)`;
						}
					},
					legend: { bottom: 0 },
					series: [
						{
							type: 'pie',
							radius: ['45%', '70%'],
							avoidLabelOverlap: false,
							label: { show: true, formatter: '{b}: {d}%' },
							data
						}
					]
				},
				$isMobile
			)
		);
	});

	function applyTarget() {
		void load();
	}

	function clearTarget() {
		targetPLNPct = null;
		void load();
	}
</script>

<section class="card preset-filled-surface-100-900 p-5 space-y-3">
	<header class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
		<div>
			<h2 class="h4 flex items-center gap-2">
				<Globe size={20} class="text-primary-500" />
				Ekspozycja walutowa
			</h2>
			<p class="text-sm text-surface-700-300">
				Udział PLN vs walut obcych w portfelu inwestycyjnym (bez mieszkania i ROR).
			</p>
		</div>
		<div class="flex flex-wrap items-end gap-2">
			<label class="label">
				<span class="text-xs font-semibold">Cel PLN (%)</span>
				<input
					type="number"
					min="0"
					max="100"
					step="1"
					class="input w-24"
					bind:value={targetPLNPct}
					placeholder="—"
				/>
			</label>
			<label class="label">
				<span class="text-xs font-semibold">Tol. (pp)</span>
				<input type="number" min="0" max="50" step="1" class="input w-20" bind:value={tolerance} />
			</label>
			<button type="button" class="btn btn-sm preset-filled-primary-500" onclick={applyTarget}>
				Zastosuj
			</button>
			{#if report?.drift}
				<button type="button" class="btn btn-sm preset-tonal-surface" onclick={clearTarget}>
					Wyczyść
				</button>
			{/if}
		</div>
	</header>

	{#if loading}
		<p class="text-sm text-surface-700-300">Ładowanie…</p>
	{:else if error}
		<div class="card preset-tonal-error p-3 text-sm">{error}</div>
	{:else if !report || report.currencies.length === 0}
		<p class="text-sm text-surface-700-300">
			Brak snapshotu lub aktywnych kont — dodaj pierwszy snapshot, aby zobaczyć ekspozycję.
		</p>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
			<div bind:this={container} class="md:col-span-2 w-full h-[280px]"></div>
			<div class="space-y-3 text-sm">
				<div class="card preset-tonal-surface p-3">
					<div class="text-xs text-surface-600-400">PLN</div>
					<div class="text-2xl font-bold">{report.pln_pct.toFixed(1)}%</div>
				</div>
				<div class="card preset-tonal-surface p-3">
					<div class="text-xs text-surface-600-400">Waluty obce</div>
					<div class="text-2xl font-bold">{report.foreign_pct.toFixed(1)}%</div>
				</div>
				{#if report.drift}
					{@const within = report.drift.within_tolerance}
					<div class="card {within ? 'preset-tonal-success' : 'preset-tonal-warning'} p-3">
						<div class="text-xs">Drift vs cel {report.drift.target_pln_pct}%</div>
						<div class="text-xl font-bold">
							{report.drift.drift_pln_pct >= 0 ? '+' : ''}{report.drift.drift_pln_pct.toFixed(1)} pp
						</div>
						<div class="text-xs">
							{within
								? `W tolerancji ±${report.drift.tolerance_pct}pp`
								: `Poza pasmem ±${report.drift.tolerance_pct}pp`}
						</div>
					</div>
				{/if}
			</div>
		</div>
		<table class="table text-sm">
			<thead>
				<tr>
					<th>Waluta</th>
					<th class="text-right">Wartość (PLN)</th>
					<th class="text-right">Udział</th>
				</tr>
			</thead>
			<tbody>
				{#each report.currencies as bucket (bucket.currency)}
					<tr>
						<td class="font-semibold">{bucket.currency}</td>
						<td class="text-right">{formatPLN(bucket.value_pln)}</td>
						<td class="text-right">{bucket.percent.toFixed(1)}%</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</section>
