<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { browser } from '$app/environment';
	import { env } from '$env/dynamic/public';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';

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
	}

	interface MortgageVsInvestResponse {
		yearly_projections: MortgageVsInvestYearlyRow[];
		summary: MortgageVsInvestSummary;
	}

	// Form state
	let remainingPrincipal = 300000;
	let annualInterestRate = 6.5;
	let remainingMonths = 240;
	let totalMonthlyBudget = 3500;
	let expectedAnnualReturn = 7.0;
	let inflationRate = 3.0;
	let enableVariableRate = false;

	// Results
	let results: MortgageVsInvestResponse | null = null;
	let loading = false;
	let error = '';

	// Chart
	let chartContainer: HTMLDivElement;
	let chart: echarts.ECharts | null = null;

	async function runSimulation() {
		loading = true;
		error = '';

		if (chart) {
			chart.dispose();
			chart = null;
		}
		results = null;

		try {
			const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
			if (!apiUrl) throw new Error('API URL not configured');

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

		if (!chart) {
			chart = echarts.init(chartContainer);
		}

		const years = results.yearly_projections.map((r) => `Rok ${r.year}`);
		const nominalA = results.yearly_projections.map((r) => r.scenario_a_after_tax_portfolio);
		const nominalB = results.yearly_projections.map((r) => r.scenario_b_after_tax_portfolio);
		const realA = results.yearly_projections.map((r) => r.scenario_a_real_portfolio);
		const realB = results.yearly_projections.map((r) => r.scenario_b_real_portfolio);

		const option: EChartsOption = {
			title: { text: 'Porównanie strategii (po podatku Belki)' },
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

		chart.setOption(option);
	}

	onMount(() => {
		const handleResize = () => chart?.resize();
		if (browser) window.addEventListener('resize', handleResize);
		return () => {
			if (browser) window.removeEventListener('resize', handleResize);
			chart?.dispose();
		};
	});

	function formatCurrency(value: number): string {
		return value.toLocaleString('pl-PL', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
	}

	function winnerLabel(strategy: string): string {
		return strategy === 'inwestycja' ? '📈 Inwestycja wygrywa' : '🏠 Nadpłata wygrywa';
	}
</script>

<div class="mortgage-page">
	<h1>Hipoteka vs Inwestycja</h1>

	<div class="content">
		<div class="form-section">
			<h2>Parametry</h2>

			<div class="form-group">
				<label>
					Kwota pozostała do spłaty (PLN)
					<input type="number" bind:value={remainingPrincipal} min="1" step="10000" />
				</label>
				<label>
					Oprocentowanie (% rocznie)
					<input type="number" bind:value={annualInterestRate} min="0.1" max="30" step="0.1" />
				</label>
				<label>
					Pozostałe miesiące
					<input type="number" bind:value={remainingMonths} min="1" max="600" step="1" />
					<small>{Math.floor(remainingMonths / 12)} lat {remainingMonths % 12} mies.</small>
				</label>
				<label>
					Miesięczny budżet (PLN)
					<input type="number" bind:value={totalMonthlyBudget} min="0" step="100" />
					<small
						>Łączna kwota na ratę i inwestycje (A: wszystko na nadpłatę, B: reszta po racie
						inwestowana)</small
					>
				</label>
				<label>
					Oczekiwany zwrot z inwestycji (% rocznie)
					<input type="number" bind:value={expectedAnnualReturn} min="0.1" max="50" step="0.1" />
				</label>
				<label>
					Inflacja (% rocznie)
					<input type="number" bind:value={inflationRate} min="0" max="20" step="0.1" />
					<small>Do przeliczenia wartości realnej (siły nabywczej)</small>
				</label>

				<label class="checkbox-label">
					<input type="checkbox" bind:checked={enableVariableRate} />
					Zmienna stopa procentowa
					<small>Cykle 10-letnie: spadek do ~1%, wzrost do ~8%, powtarza się</small>
				</label>
			</div>

			<button class="primary-button" on:click={runSimulation} disabled={loading}>
				{loading ? 'Obliczanie...' : 'Oblicz'}
			</button>

			{#if error}
				<div class="error-message">{error}</div>
			{/if}
		</div>

		{#if results}
			<div class="results-section">
				<h2>Wyniki</h2>

				<div class="winner-banner" class:invest={results.summary.winning_strategy === 'inwestycja'}>
					{winnerLabel(results.summary.winning_strategy)}
					<span class="net-advantage">
						Przewaga: {formatCurrency(results.summary.net_advantage)} PLN
					</span>
				</div>

				<div class="summary-cards">
					<div class="summary-card">
						<div class="card-label">Rata bazowa</div>
						<div class="card-value">
							{formatCurrency(results.summary.regular_monthly_payment)} PLN
						</div>
					</div>
					<div class="summary-card">
						<div class="card-label">Zaoszczędzone odsetki (nadpłata)</div>
						<div class="card-value">{formatCurrency(results.summary.interest_saved)} PLN</div>
					</div>
					<div class="summary-card">
						<div class="card-label">Portfel inwestycyjny B (brutto)</div>
						<div class="card-value">
							{formatCurrency(results.summary.final_investment_portfolio)} PLN
						</div>
					</div>
					<div class="summary-card">
						<div class="card-label">Miesięcy wcześniej (A)</div>
						<div class="card-value">{results.summary.months_saved}</div>
						<div class="card-note">
							{Math.floor(results.summary.months_saved / 12)} lat {results.summary.months_saved %
								12} mies.
						</div>
					</div>
					<div class="summary-card">
						<div class="card-label">Odsetki razem (nadpłata A)</div>
						<div class="card-value">{formatCurrency(results.summary.total_interest_a)} PLN</div>
					</div>
					<div class="summary-card">
						<div class="card-label">Odsetki razem (inwestycja B)</div>
						<div class="card-value">{formatCurrency(results.summary.total_interest_b)} PLN</div>
					</div>
					<div class="summary-card belka">
						<div class="card-label">Podatek Belki A (19%)</div>
						<div class="card-value">{formatCurrency(results.summary.belka_tax_a)} PLN</div>
					</div>
					<div class="summary-card belka">
						<div class="card-label">Podatek Belki B (19%)</div>
						<div class="card-value">{formatCurrency(results.summary.belka_tax_b)} PLN</div>
					</div>
					<div class="summary-card real">
						<div class="card-label">Portfel A realny (dziś PLN)</div>
						<div class="card-value">
							{formatCurrency(results.summary.final_portfolio_a_real)} PLN
						</div>
					</div>
					<div class="summary-card real">
						<div class="card-label">Portfel B realny (dziś PLN)</div>
						<div class="card-value">
							{formatCurrency(results.summary.final_portfolio_b_real)} PLN
						</div>
					</div>
				</div>

				<div class="chart-container" bind:this={chartContainer}></div>

				<details class="projection-details">
					<summary>Projekcja roczna</summary>
					<div class="projection-table">
						<table>
							<thead>
								<tr>
									<th>Rok</th>
									<th>Stopa %</th>
									<th>Saldo A</th>
									<th>Saldo A (realne)</th>
									<th>Portfel A (brutto)</th>
									<th>Portfel A (po Belce)</th>
									<th>Portfel A (realny)</th>
									<th>Spłacone A</th>
									<th>Saldo B</th>
									<th>Saldo B (realne)</th>
									<th>Portfel B (brutto)</th>
									<th>Portfel B (po Belce)</th>
									<th>Portfel B (realny)</th>
									<th>Przewaga B (realna)</th>
								</tr>
							</thead>
							<tbody>
								{#each results.yearly_projections as row}
									<tr class:paid-off={row.scenario_a_paid_off}>
										<td>{row.year}</td>
										<td>{row.annual_rate.toFixed(2)}%</td>
										<td>{formatCurrency(row.scenario_a_mortgage_balance)}</td>
										<td>{formatCurrency(row.scenario_a_real_mortgage_balance)}</td>
										<td>{formatCurrency(row.scenario_a_investment_balance)}</td>
										<td>{formatCurrency(row.scenario_a_after_tax_portfolio)}</td>
										<td>{formatCurrency(row.scenario_a_real_portfolio)}</td>
										<td>{row.scenario_a_paid_off ? '✓' : '—'}</td>
										<td>{formatCurrency(row.scenario_b_mortgage_balance)}</td>
										<td>{formatCurrency(row.scenario_b_real_mortgage_balance)}</td>
										<td>{formatCurrency(row.scenario_b_investment_balance)}</td>
										<td>{formatCurrency(row.scenario_b_after_tax_portfolio)}</td>
										<td>{formatCurrency(row.scenario_b_real_portfolio)}</td>
										<td
											class:positive={row.net_advantage_invest >= 0}
											class:negative={row.net_advantage_invest < 0}
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

<style>
	.mortgage-page {
		padding: var(--size-4);
		max-width: 1400px;
		margin: 0 auto;
	}

	h1 {
		margin-bottom: var(--size-6);
		color: var(--color-text-1);
	}

	.content {
		display: grid;
		grid-template-columns: 400px 1fr;
		gap: var(--size-6);
		align-items: start;
	}

	.form-section,
	.results-section {
		background: var(--surface-2);
		padding: var(--size-5);
		border-radius: var(--radius-2);
	}

	h2 {
		margin-top: 0;
		margin-bottom: var(--size-4);
		color: var(--color-text-2);
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: var(--size-3);
		margin-bottom: var(--size-4);
	}

	label {
		display: flex;
		flex-direction: column;
		gap: var(--size-1);
		font-size: var(--font-size-1);
		color: var(--color-text-2);
	}

	input[type='number'] {
		padding: var(--size-2);
		border: 1px solid var(--surface-4);
		border-radius: var(--radius-2);
		background: var(--surface-1);
		color: var(--color-text-1);
		font-size: var(--font-size-1);
	}

	small {
		font-size: var(--font-size-0);
		color: var(--color-text-3);
	}

	.checkbox-label {
		flex-direction: row;
		align-items: center;
		gap: var(--size-2);
		font-weight: 600;
	}

	.checkbox-label input[type='checkbox'] {
		width: 1rem;
		height: 1rem;
		cursor: pointer;
	}

	.primary-button {
		width: 100%;
		padding: var(--size-3);
		background: var(--color-primary);
		color: white;
		border: none;
		border-radius: var(--radius-2);
		font-size: var(--font-size-2);
		font-weight: 600;
		cursor: pointer;
	}

	.primary-button:hover:not(:disabled) {
		background: var(--color-primary-hover);
	}

	.primary-button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.error-message {
		margin-top: var(--size-3);
		padding: var(--size-3);
		background: var(--color-error-bg);
		color: var(--color-error);
		border-radius: var(--radius-2);
	}

	.winner-banner {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: var(--size-4);
		border-radius: var(--radius-2);
		margin-bottom: var(--size-4);
		font-weight: 700;
		font-size: var(--font-size-3);
		background: hsl(210 14% 85%);
		color: var(--color-text-1);
	}

	.winner-banner.invest {
		background: hsl(92 30% 80%);
	}

	.net-advantage {
		font-size: var(--font-size-1);
		font-weight: 400;
	}

	.summary-cards {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
		gap: var(--size-3);
		margin-bottom: var(--size-5);
	}

	.summary-card {
		background: var(--surface-3);
		padding: var(--size-4);
		border-radius: var(--radius-2);
	}

	.card-label {
		font-size: var(--font-size-0);
		color: var(--color-text-3);
		margin-bottom: var(--size-2);
	}

	.card-value {
		font-size: var(--font-size-3);
		font-weight: 700;
		color: var(--color-text-1);
	}

	.card-note {
		font-size: var(--font-size-0);
		color: var(--color-text-3);
		margin-top: var(--size-1);
	}

	.summary-card.belka {
		border-left: 3px solid hsl(30 70% 55%);
	}

	.summary-card.real {
		border-left: 3px solid hsl(200 60% 55%);
	}

	.chart-container {
		width: 100%;
		height: 380px;
		margin-bottom: var(--size-5);
	}

	.projection-details {
		background: var(--surface-3);
		padding: var(--size-3);
		border-radius: var(--radius-2);
	}

	.projection-details summary {
		cursor: pointer;
		padding: var(--size-2);
		font-size: var(--font-size-2);
		font-weight: 600;
	}

	.projection-table {
		margin-top: var(--size-3);
		overflow-x: auto;
	}

	table {
		width: 100%;
		border-collapse: collapse;
		font-size: var(--font-size-0);
	}

	th,
	td {
		padding: var(--size-2);
		text-align: right;
		border-bottom: 1px solid var(--surface-4);
		white-space: nowrap;
	}

	th {
		background: var(--surface-4);
		font-weight: 600;
		color: var(--color-text-2);
	}

	th:first-child,
	td:first-child {
		text-align: left;
	}

	tr.paid-off td:first-child::after {
		content: ' ✓';
		color: green;
	}

	.positive {
		color: hsl(130 50% 35%);
	}

	.negative {
		color: hsl(0 60% 45%);
	}

	@media (max-width: 1100px) {
		.content {
			grid-template-columns: 1fr;
		}
	}
</style>
