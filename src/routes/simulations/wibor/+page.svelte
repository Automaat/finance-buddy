<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { resolveApiUrl } from '$lib/api';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import { createChart, type ChartHandle } from '$lib/utils/charts/lifecycle';
	import { Info } from 'lucide-svelte';

	interface ScenarioRow {
		delta_pp: number;
		annual_rate: number;
		monthly_payment: number;
		total_interest: number;
		term_months: number;
		rate_floored: boolean;
		yearly_balances: number[];
	}

	interface WiborResponse {
		base_payment: number;
		scenarios: ScenarioRow[];
	}

	let remainingPrincipal = $state(300000);
	let baseAnnualRate = $state(7.5);
	let remainingMonths = $state(240);
	let basePayment = $state(0);

	let results: WiborResponse | null = $state(null);
	let loading = $state(false);
	let error = $state('');
	let chartContainer: HTMLDivElement | undefined = $state();
	let chartHandle: ChartHandle | null = null;
	let chart: echarts.ECharts | null = null;

	async function runSimulation(): Promise<void> {
		loading = true;
		error = '';
		results = null;
		try {
			const apiUrl = resolveApiUrl();
			const response = await fetch(`${apiUrl}/api/simulations/wibor`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					remaining_principal: remainingPrincipal,
					base_annual_rate: baseAnnualRate,
					remaining_months: remainingMonths,
					base_payment: basePayment > 0 ? basePayment : null
				})
			});
			if (!response.ok) {
				const detail = await response.json().catch(() => ({ detail: response.statusText }));
				throw new Error(detail.detail ?? response.statusText);
			}
			results = (await response.json()) as WiborResponse;
			await tick();
			renderChart();
		} catch (err) {
			console.error('WIBOR scenarios failed:', err);
			if (err instanceof Error) error = err.message;
		} finally {
			loading = false;
		}
	}

	const SCENARIO_COLORS = ['#A3BE8C', '#5E81AC', '#EBCB8B', '#D08770', '#BF616A'];

	function renderChart(): void {
		if (!results || !chartContainer) return;
		if (!chartHandle) {
			chartHandle = createChart(chartContainer);
			chart = chartHandle.chart;
		}
		const maxYears = Math.max(...results.scenarios.map((s) => s.yearly_balances.length));
		const xAxis = Array.from({ length: maxYears }, (_, i) => `Rok ${i + 1}`);
		const series = results.scenarios.map((s, idx) => ({
			name: formatDelta(s.delta_pp),
			type: 'line' as const,
			data: s.yearly_balances,
			smooth: true,
			showSymbol: false,
			emphasis: { focus: 'series' as const },
			itemStyle: { color: SCENARIO_COLORS[idx % SCENARIO_COLORS.length] },
			lineStyle: { width: s.delta_pp === 0 ? 3 : 2 }
		}));
		const option: EChartsOption = {
			title: { text: 'Saldo kredytu w czasie (amortyzacja)' },
			tooltip: {
				trigger: 'axis',
				formatter: (params: unknown) => {
					const list = params as Array<{ name: string; seriesName: string; value: number }>;
					let out = `<strong>${list[0]?.name ?? ''}</strong><br/>`;
					for (const p of list) {
						const v = p.value ?? 0;
						out += `${p.seriesName}: ${v.toLocaleString('pl-PL', { maximumFractionDigits: 0 })} PLN<br/>`;
					}
					return out;
				}
			},
			legend: { data: series.map((s) => s.name), bottom: 0 },
			grid: { left: '3%', right: '4%', bottom: '20%', containLabel: true },
			xAxis: { type: 'category', data: xAxis },
			yAxis: {
				type: 'value',
				name: 'Saldo (PLN)',
				axisLabel: { formatter: (v: number) => `${(v / 1000).toFixed(0)}k` }
			},
			series
		};
		chart?.setOption(option);
	}

	function formatCurrency(value: number): string {
		return value.toLocaleString('pl-PL', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
	}

	function formatDelta(delta: number): string {
		if (delta === 0) return 'obecna';
		const sign = delta > 0 ? '+' : '';
		return `${sign}${delta} pp`;
	}

	function formatMonths(m: number): string {
		const years = Math.floor(m / 12);
		const months = m % 12;
		if (years === 0) return `${months} mies.`;
		if (months === 0) return `${years} lat`;
		return `${years} lat ${months} mies.`;
	}

	onMount(() => {
		return () => {
			chartHandle?.dispose();
			chartHandle = null;
			chart = null;
		};
	});
</script>

<svelte:head>
	<title>WIBOR — Symulacje | Finansowa Forteca</title>
</svelte:head>

<div class="space-y-4">
	<div class="flex items-center gap-2">
		<h1 class="h1">Szoki WIBOR</h1>
		<details class="relative inline-block">
			<summary class="cursor-pointer text-surface-600-400" aria-label="Co to jest WIBOR">
				<Info size={20} />
			</summary>
			<div
				class="absolute left-0 z-20 mt-2 w-80 sm:w-96 card preset-filled-surface-100-900 p-3 text-sm shadow-xl"
			>
				<p>
					<strong>WIBOR</strong> (Warsaw Interbank Offered Rate) to średnia stawka, po jakiej banki pożyczają
					sobie pieniądze. Większość polskich hipotek ma oprocentowanie zmienne: rata = WIBOR (3M lub
					6M) + marża banku. Gdy WIBOR rośnie, rata kredytu rośnie wraz z nim.
				</p>
				<p class="mt-2">
					W 2022 WIBOR 3M wzrósł z ~0.2% do ~7.6% w ciągu roku — rata wzrosła o około 70%. Ta
					symulacja pokazuje, jak rata i koszt odsetek zmieniają się przy szokach +/-1pp, +2pp,
					+3pp.
				</p>
			</div>
		</details>
	</div>

	<div class="grid grid-cols-1 lg:grid-cols-[400px_1fr] gap-6 items-start">
		<div class="card preset-filled-surface-100-900 p-5 space-y-4">
			<h2 class="h3">Parametry kredytu</h2>
			<div class="flex flex-col gap-3">
				<label class="label">
					<span class="text-sm font-semibold">Pozostały kapitał (PLN)</span>
					<input type="number" bind:value={remainingPrincipal} min="1" step="10000" class="input" />
				</label>
				<label class="label">
					<span class="text-sm font-semibold">Obecne oprocentowanie (% rocznie)</span>
					<input
						type="number"
						bind:value={baseAnnualRate}
						min="0.1"
						max="30"
						step="0.1"
						class="input"
					/>
					<span class="text-xs text-surface-600-400">WIBOR + marża banku</span>
				</label>
				<label class="label">
					<span class="text-sm font-semibold">Pozostałe miesiące</span>
					<input
						type="number"
						bind:value={remainingMonths}
						min="1"
						max="600"
						step="1"
						class="input"
					/>
					<span class="text-xs text-surface-600-400">
						{Math.floor(remainingMonths / 12)} lat {remainingMonths % 12} mies.
					</span>
				</label>
				<label class="label">
					<span class="text-sm font-semibold">Obecna rata (PLN, opcjonalnie)</span>
					<input type="number" bind:value={basePayment} min="0" step="50" class="input" />
					<span class="text-xs text-surface-600-400">
						Zostaw 0 aby pominąć. Jeśli podasz, policzymy też, ile lat zajmie spłata przy tej racie
						po szoku.
					</span>
				</label>
			</div>

			<button
				class="btn preset-filled-primary-500 w-full"
				onclick={runSimulation}
				disabled={loading}
			>
				{loading ? 'Obliczanie...' : 'Policz scenariusze'}
			</button>

			{#if error}
				<div class="card preset-filled-error-500 p-3 text-sm">{error}</div>
			{/if}
		</div>

		{#if results}
			<div class="card preset-filled-surface-100-900 p-5 space-y-4">
				<h2 class="h3">Wyniki</h2>

				<div class="table-wrap">
					<table class="table table-hover text-sm">
						<thead>
							<tr>
								<th>Scenariusz</th>
								<th class="text-right">Oprocentowanie</th>
								<th class="text-right">Rata miesięczna</th>
								<th class="text-right">Δ rata</th>
								<th class="text-right">Łączne odsetki</th>
								{#if basePayment > 0}
									<th class="text-right">Spłata przy obecnej racie</th>
								{/if}
							</tr>
						</thead>
						<tbody>
							{#each results.scenarios as row (row.delta_pp)}
								<tr class:font-semibold={row.delta_pp === 0}>
									<td>
										{formatDelta(row.delta_pp)}
										{#if row.rate_floored}<span
												class="ml-1 text-xs text-warning-500"
												title="Szok obniżyłby stopę poniżej zera — przycięto do 0.01%">⚠</span
											>{/if}
									</td>
									<td class="text-right">{row.annual_rate.toFixed(2)}%</td>
									<td class="text-right">{formatCurrency(row.monthly_payment)} PLN</td>
									<td
										class="text-right {row.monthly_payment > results.base_payment
											? 'text-error-500'
											: row.monthly_payment < results.base_payment
												? 'text-success-500'
												: ''}"
									>
										{#if results.base_payment > 0 && row.delta_pp !== 0}
											{row.monthly_payment > results.base_payment ? '+' : ''}{formatCurrency(
												row.monthly_payment - results.base_payment
											)} PLN
										{:else}
											—
										{/if}
									</td>
									<td class="text-right">{formatCurrency(row.total_interest)} PLN</td>
									{#if basePayment > 0}
										<td class="text-right">
											{row.term_months >= 1200 ? '> 100 lat' : formatMonths(row.term_months)}
										</td>
									{/if}
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<div bind:this={chartContainer} class="w-full h-[280px] sm:h-[380px]"></div>

				<p class="text-xs text-surface-600-400">
					Symulacja zakłada amortyzację w pozostałym okresie po szoku. Bank zwykle aktualizuje ratę
					po zmianie WIBOR — kolumna <em>Spłata przy obecnej racie</em> pokazuje, ile lat zajmie spłata,
					gdybyś zachował obecną wysokość raty.
				</p>
			</div>
		{/if}
	</div>
</div>
