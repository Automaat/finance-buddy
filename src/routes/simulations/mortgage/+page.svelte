<script lang="ts">
	import { onMount, tick, untrack } from 'svelte';
	import { resolveApiUrl } from '$lib/api';
	import { applyMobileChartTweaks } from '$lib/utils/charts/responsive';
	import { isMobile } from '$lib/utils/viewport';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import { createChart, type ChartHandle } from '$lib/utils/charts/lifecycle';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}
	let { data }: Props = $props();

	interface MortgageVsInvestYearlyRow {
		year: number;
		annual_rate: number;
		scenario_a_mortgage_balance: number;
		scenario_a_real_mortgage_balance: number;
		scenario_a_cumulative_interest: number;
		scenario_a_investment_balance: number;
		scenario_a_after_tax_portfolio: number;
		scenario_a_real_portfolio: number;
		scenario_a_paid_off: boolean;
		scenario_b_mortgage_balance: number;
		scenario_b_real_mortgage_balance: number;
		scenario_b_investment_balance: number;
		scenario_b_after_tax_portfolio: number;
		scenario_b_real_portfolio: number;
		scenario_b_cumulative_interest: number;
		net_advantage_invest: number;
	}

	interface MortgageVsInvestSummary {
		regular_monthly_payment: number;
		total_interest_a: number;
		total_interest_b: number;
		interest_saved: number;
		final_investment_portfolio: number;
		belka_tax_a: number;
		belka_tax_b: number;
		final_portfolio_a_real: number;
		final_portfolio_b_real: number;
		months_saved: number;
		winning_strategy: string;
		net_advantage: number;
		break_even_gross_return: number;
	}

	interface MortgageVsInvestResponse {
		yearly_projections: MortgageVsInvestYearlyRow[];
		summary: MortgageVsInvestSummary;
	}

	// Form state. Defaults pull from app_config where mappings exist:
	// expectedAnnualReturn ← expected_return_rate, totalMonthlyBudget ←
	// monthly_mortgage_payment (when set). Mortgage-specific fields (balance,
	// rate, term) have no config equivalent yet — keep numeric placeholders.
	let remainingPrincipal = $state(300000);
	let annualInterestRate = $state(6.5);
	let remainingMonths = $state(240);
	let totalMonthlyBudget = $state(
		untrack(() => {
			// monthly_mortgage_payment === 0 is a legitimate "no mortgage" signal,
			// but it makes the simulator output meaningless — keep the 3500
			// placeholder until the user types something.
			const m = data?.defaults?.monthlyMortgagePLN;
			return m != null && m > 0 ? m : 3500;
		})
	);
	let expectedAnnualReturn = $state(untrack(() => data?.defaults?.annualReturnPct ?? 7.0));
	let inflationRate = $state(3.0);
	let enableVariableRate = $state(false);

	// Results
	let results: MortgageVsInvestResponse | null = $state(null);
	let loading = $state(false);
	let error = $state('');

	// Chart
	let chartContainer: HTMLDivElement | undefined = $state();
	let chart: echarts.ECharts | null = null;
	let chartHandle: ChartHandle | null = null;

	async function runSimulation() {
		loading = true;
		error = '';

		if (chartHandle) {
			chartHandle.dispose();
			chartHandle = null;
			chart = null;
		}
		results = null;

		try {
			const apiUrl = resolveApiUrl();

			const response = await fetch(`${apiUrl}/api/simulations/mortgage-vs-invest`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					remaining_principal: remainingPrincipal,
					annual_interest_rate: annualInterestRate,
					remaining_months: remainingMonths,
					total_monthly_budget: totalMonthlyBudget,
					expected_annual_return: expectedAnnualReturn,
					inflation_rate: inflationRate,
					enable_variable_rate: enableVariableRate
				})
			});

			if (!response.ok) {
				const detail = await response.json().catch(() => ({ detail: response.statusText }));
				throw new Error(detail.detail ?? response.statusText);
			}

			results = await response.json();
			await tick();
			renderChart();
		} catch (err) {
			console.error('Simulation failed:', err);
			if (err instanceof Error) {
				error = err.message;
			}
		} finally {
			loading = false;
		}
	}

	function renderChart() {
		if (!results || !chartContainer) return;

		if (!chartHandle) {
			chartHandle = createChart(chartContainer);
			chart = chartHandle.chart;
		}

		const years = results.yearly_projections.map((r) => `Rok ${r.year}`);
		const nominalA = results.yearly_projections.map((r) => r.scenario_a_after_tax_portfolio);
		const nominalB = results.yearly_projections.map((r) => r.scenario_b_after_tax_portfolio);
		const realA = results.yearly_projections.map((r) => r.scenario_a_real_portfolio);
		const realB = results.yearly_projections.map((r) => r.scenario_b_real_portfolio);

		const option: EChartsOption = {
			title: { text: 'Porównanie strategii (Belka wbudowana w stopę zwrotu)' },
			tooltip: {
				trigger: 'axis',
				formatter: (params: any) => {
					let result = `<strong>${params[0].name}</strong><br/>`;
					params.forEach((p: any) => {
						result += `${p.seriesName}: ${p.value.toLocaleString('pl-PL', { maximumFractionDigits: 0 })} PLN<br/>`;
					});
					return result;
				}
			},
			legend: {
				data: [
					'Portfel A nominalne',
					'Portfel B nominalne',
					'Portfel A realne',
					'Portfel B realne'
				],
				bottom: 0
			},
			grid: { left: '3%', right: '4%', bottom: '20%', containLabel: true },
			xAxis: { type: 'category', data: years },
			yAxis: {
				type: 'value',
				name: 'Wartość (PLN)',
				axisLabel: { formatter: (v: number) => `${(v / 1000).toFixed(0)}k` }
			},
			series: [
				{
					name: 'Portfel A nominalne',
					type: 'line',
					data: nominalA,
					smooth: true,
					itemStyle: { color: '#5E81AC' }
				},
				{
					name: 'Portfel B nominalne',
					type: 'line',
					data: nominalB,
					smooth: true,
					itemStyle: { color: '#A3BE8C' }
				},
				{
					name: 'Portfel A realne',
					type: 'line',
					data: realA,
					smooth: true,
					lineStyle: { type: 'dashed' },
					itemStyle: { color: '#5E81AC' }
				},
				{
					name: 'Portfel B realne',
					type: 'line',
					data: realB,
					smooth: true,
					lineStyle: { type: 'dashed' },
					itemStyle: { color: '#A3BE8C' }
				}
			]
		};

		chart?.setOption(applyMobileChartTweaks(option, $isMobile));
	}

	onMount(() => {
		return () => {
			chartHandle?.dispose();
			chartHandle = null;
			chart = null;
		};
	});

	function formatCurrency(value: number): string {
		return value.toLocaleString('pl-PL', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
	}

	function winnerLabel(strategy: string): string {
		return strategy === 'inwestycja' ? '📈 Inwestycja wygrywa' : '🏠 Nadpłata wygrywa';
	}
</script>

<div class="space-y-4">
	<h1 class="h1">Hipoteka vs Inwestycja</h1>

	<div class="grid grid-cols-1 lg:grid-cols-[400px_1fr] gap-6 items-start">
		<div class="card preset-filled-surface-100-900 p-5 space-y-4">
			<h2 class="h3">Parametry</h2>

			<div class="flex flex-col gap-3">
				<label class="label">
					<span class="text-sm font-semibold">Kwota pozostała do spłaty (PLN)</span>
					<input type="number" bind:value={remainingPrincipal} min="1" step="10000" class="input" />
				</label>
				<label class="label">
					<span class="text-sm font-semibold">Oprocentowanie (% rocznie)</span>
					<input
						type="number"
						bind:value={annualInterestRate}
						min="0.1"
						max="30"
						step="0.1"
						class="input"
					/>
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
					<span class="text-xs text-surface-600-400"
						>{Math.floor(remainingMonths / 12)} lat {remainingMonths % 12} mies.</span
					>
				</label>
				<label class="label">
					<span class="text-sm font-semibold">Miesięczny budżet (PLN)</span>
					<input type="number" bind:value={totalMonthlyBudget} min="0" step="100" class="input" />
					<span class="text-xs text-surface-600-400"
						>Łączna kwota na ratę i inwestycje (A: wszystko na nadpłatę, B: reszta po racie
						inwestowana)</span
					>
				</label>
				<label class="label">
					<span class="text-sm font-semibold">Oczekiwany zwrot z inwestycji (% rocznie)</span>
					<input
						type="number"
						bind:value={expectedAnnualReturn}
						min="0.1"
						max="50"
						step="0.1"
						class="input"
					/>
				</label>
				<label class="label">
					<span class="text-sm font-semibold">Inflacja (% rocznie)</span>
					<input
						type="number"
						bind:value={inflationRate}
						min="0"
						max="20"
						step="0.1"
						class="input"
					/>
					<span class="text-xs text-surface-600-400"
						>Do przeliczenia wartości realnej (siły nabywczej)</span
					>
				</label>

				<div class="space-y-1">
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="checkbox" bind:checked={enableVariableRate} class="checkbox" />
						<span class="text-sm font-semibold">Zmienna stopa procentowa</span>
					</label>
					<p class="text-xs text-surface-600-400 ml-6">
						Cykle 10-letnie: spadek do ~1%, wzrost do ~8%, powtarza się
					</p>
				</div>
			</div>

			<button
				class="btn preset-filled-primary-500 w-full"
				onclick={runSimulation}
				disabled={loading}
			>
				{loading ? 'Obliczanie...' : 'Oblicz'}
			</button>

			{#if error}
				<div class="card preset-filled-error-500 p-3 text-sm">{error}</div>
			{/if}
		</div>

		{#if results}
			<div class="card preset-filled-surface-100-900 p-5 space-y-4">
				<h2 class="h3">Wyniki</h2>

				<div
					class="flex justify-between items-center p-4 rounded-container font-bold text-lg {results
						.summary.winning_strategy === 'inwestycja'
						? 'bg-success-500/10'
						: 'bg-surface-200-800'}"
				>
					{winnerLabel(results.summary.winning_strategy)}
					<span class="text-sm font-normal">
						Przewaga: {formatCurrency(results.summary.net_advantage)} PLN
					</span>
				</div>

				<div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">Rata bazowa</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.summary.regular_monthly_payment)} PLN
						</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">Zaoszczędzone odsetki (nadpłata)</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.summary.interest_saved)} PLN
						</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">Portfel inwestycyjny B (brutto)</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.summary.final_investment_portfolio)} PLN
						</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">Miesięcy wcześniej (A)</div>
						<div class="text-xl font-bold">{results.summary.months_saved}</div>
						<div class="text-xs text-surface-600-400 mt-1">
							{Math.floor(results.summary.months_saved / 12)} lat {results.summary.months_saved %
								12} mies.
						</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">Odsetki razem (nadpłata A)</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.summary.total_interest_a)} PLN
						</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">Odsetki razem (inwestycja B)</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.summary.total_interest_b)} PLN
						</div>
					</div>
					<div
						class="card preset-tonal-surface p-4 border-l-4 {expectedAnnualReturn >=
						results.summary.break_even_gross_return
							? 'border-l-success-500'
							: 'border-l-error-500'}"
					>
						<div class="text-xs text-surface-600-400 mb-2">Próg rentowności (brutto)</div>
						<div class="text-xl font-bold">
							{results.summary.break_even_gross_return.toFixed(2)}%
						</div>
						<div
							class="text-xs mt-1 {expectedAnnualReturn >= results.summary.break_even_gross_return
								? 'text-success-500'
								: 'text-error-500'}"
						>
							min. stopa przed podatkiem Belki · twój cel: {expectedAnnualReturn.toFixed(1)}%
							{expectedAnnualReturn >= results.summary.break_even_gross_return ? '✓' : '✗'}
						</div>
					</div>
					<div class="card preset-tonal-surface p-4 border-l-4 border-l-warning-500">
						<div class="text-xs text-surface-600-400 mb-2">Podatek Belki (19%)</div>
						<div class="text-xl font-bold">wbudowany</div>
						<div class="text-xs text-surface-600-400 mt-1">
							odliczany co miesiąc od stopy zwrotu
						</div>
					</div>
					<div class="card preset-tonal-surface p-4 border-l-4 border-l-primary-500">
						<div class="text-xs text-surface-600-400 mb-2">Portfel A realny (dziś PLN)</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.summary.final_portfolio_a_real)} PLN
						</div>
					</div>
					<div class="card preset-tonal-surface p-4 border-l-4 border-l-primary-500">
						<div class="text-xs text-surface-600-400 mb-2">Portfel B realny (dziś PLN)</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.summary.final_portfolio_b_real)} PLN
						</div>
					</div>
				</div>

				<div bind:this={chartContainer} class="w-full h-[280px] sm:h-[380px]"></div>

				<details class="card preset-tonal-surface p-3">
					<summary class="cursor-pointer text-sm font-semibold py-1">Projekcja roczna</summary>
					<div class="table-wrap mt-3">
						<table class="table table-hover text-xs">
							<thead>
								<tr>
									<th class="sticky left-0 z-10 bg-surface-100 dark:bg-surface-900">Rok</th>
									<th class="text-right">Stopa %</th>
									<th class="text-right">Saldo A</th>
									<th class="text-right">Saldo A (realne)</th>
									<th class="text-right">Portfel A (brutto)</th>
									<th class="text-right">Portfel A (po Belce)</th>
									<th class="text-right">Portfel A (realny)</th>
									<th class="text-right">Spłacone A</th>
									<th class="text-right">Saldo B</th>
									<th class="text-right">Saldo B (realne)</th>
									<th class="text-right">Portfel B (brutto)</th>
									<th class="text-right">Portfel B (po Belce)</th>
									<th class="text-right">Portfel B (realny)</th>
									<th class="text-right">Przewaga B (realna)</th>
								</tr>
							</thead>
							<tbody>
								{#each results.yearly_projections as row}
									<tr>
										<td class="sticky left-0 z-10 bg-surface-100 dark:bg-surface-900"
											>{row.year}{#if row.scenario_a_paid_off}
												✓{/if}</td
										>
										<td class="text-right">{row.annual_rate.toFixed(2)}%</td>
										<td class="text-right">{formatCurrency(row.scenario_a_mortgage_balance)}</td>
										<td class="text-right"
											>{formatCurrency(row.scenario_a_real_mortgage_balance)}</td
										>
										<td class="text-right">{formatCurrency(row.scenario_a_investment_balance)}</td>
										<td class="text-right">{formatCurrency(row.scenario_a_after_tax_portfolio)}</td>
										<td class="text-right">{formatCurrency(row.scenario_a_real_portfolio)}</td>
										<td class="text-right">{row.scenario_a_paid_off ? '✓' : '—'}</td>
										<td class="text-right">{formatCurrency(row.scenario_b_mortgage_balance)}</td>
										<td class="text-right"
											>{formatCurrency(row.scenario_b_real_mortgage_balance)}</td
										>
										<td class="text-right">{formatCurrency(row.scenario_b_investment_balance)}</td>
										<td class="text-right">{formatCurrency(row.scenario_b_after_tax_portfolio)}</td>
										<td class="text-right">{formatCurrency(row.scenario_b_real_portfolio)}</td>
										<td
											class="text-right {row.net_advantage_invest >= 0
												? 'text-success-500'
												: 'text-error-500'}"
										>
											{formatCurrency(row.net_advantage_invest)}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</details>
			</div>
		{/if}
	</div>
</div>
