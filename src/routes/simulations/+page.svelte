<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { env } from '$env/dynamic/public';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import type { PageData } from './$types';

	interface YearlyProjection {
		year: number;
		age: number;
		annual_contribution: number;
		balance_end_of_year: number;
		cumulative_contributions: number;
		cumulative_returns: number;
		annual_limit: number;
		limit_utilized_pct: number;
		tax_savings: number;
	}

	interface AccountSimulation {
		account_name: string;
		starting_balance: number;
		total_contributions: number;
		total_returns: number;
		total_tax_savings: number;
		final_balance: number;
		yearly_projections: YearlyProjection[];
	}

	interface SimulationSummary {
		total_final_balance: number;
		total_contributions: number;
		total_returns: number;
		total_tax_savings: number;
		estimated_monthly_income: number;
		estimated_monthly_income_today: number;
		years_until_retirement: number;
	}

	interface SimulationResponse {
		simulations: AccountSimulation[];
		summary: SimulationSummary;
	}

	export let data: PageData;

	// Form state
	let currentAge = data.current_age;
	let retirementAge = data.retirement_age;

	// Account selection
	let simulateIkeMarcin = true;
	let simulateIkeEwa = true;
	let simulateIkzeMarcin = true;
	let simulateIkzeEwa = true;

	// Balances
	let ikeMarcinBalance = data.balances.ike_marcin;
	let ikeEwaBalance = data.balances.ike_ewa;
	let ikzeMarcinBalance = data.balances.ikze_marcin;
	let ikzeEwaBalance = data.balances.ikze_ewa;

	// Contribution strategies
	let ikeMarcinAutoFill = false;
	let ikeMarcinMonthly = 0;
	let ikeEwaAutoFill = false;
	let ikeEwaMonthly = 0;
	let ikzeMarcinAutoFill = false;
	let ikzeMarcinMonthly = 0;
	let ikzeEwaAutoFill = false;
	let ikzeEwaMonthly = 0;

	// Tax rates
	let marcinTaxRate = 17.0;
	let ewaTaxRate = 17.0;

	// Assumptions
	let annualReturnRate = 7.0;
	let limitGrowthRate = 5.0;

	// Results
	let results: SimulationResponse | null = null;
	let loading = false;
	let error = '';

	// Chart
	let chartContainer: HTMLDivElement;
	let chart: echarts.ECharts | null = null;

	async function runSimulation() {
		loading = true;
		error = '';
		results = null;

		try {
			const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
			if (!apiUrl) throw new Error('API URL not configured');

			const response = await fetch(`${apiUrl}/api/simulations/retirement`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					current_age: currentAge,
					retirement_age: retirementAge,
					simulate_ike_marcin: simulateIkeMarcin,
					simulate_ike_ewa: simulateIkeEwa,
					simulate_ikze_marcin: simulateIkzeMarcin,
					simulate_ikze_ewa: simulateIkzeEwa,
					ike_marcin_balance: ikeMarcinBalance,
					ike_ewa_balance: ikeEwaBalance,
					ikze_marcin_balance: ikzeMarcinBalance,
					ikze_ewa_balance: ikzeEwaBalance,
					ike_marcin_auto_fill: ikeMarcinAutoFill,
					ike_marcin_monthly: ikeMarcinMonthly,
					ike_ewa_auto_fill: ikeEwaAutoFill,
					ike_ewa_monthly: ikeEwaMonthly,
					ikze_marcin_auto_fill: ikzeMarcinAutoFill,
					ikze_marcin_monthly: ikzeMarcinMonthly,
					ikze_ewa_auto_fill: ikzeEwaAutoFill,
					ikze_ewa_monthly: ikzeEwaMonthly,
					marcin_tax_rate: marcinTaxRate,
					ewa_tax_rate: ewaTaxRate,
					annual_return_rate: annualReturnRate,
					limit_growth_rate: limitGrowthRate
				})
			});

			if (!response.ok) {
				throw new Error(`Simulation failed: ${response.statusText}`);
			}

			const data = await response.json();
			results = data;
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

		const years = results.simulations[0]?.yearly_projections.map((p) => p.year) || [];
		const series = results.simulations.map((sim, idx) => {
			const colors = ['#5E81AC', '#81A1C1', '#88C0D0', '#8FBCBB'];
			return {
				name: sim.account_name,
				type: 'line' as const,
				data: sim.yearly_projections.map((p) => p.balance_end_of_year),
				smooth: true,
				itemStyle: { color: colors[idx % colors.length] }
			};
		});

		const option: EChartsOption = {
			title: { text: 'Projekcja wartości kont emerytalnych' },
			tooltip: {
				trigger: 'axis',
				formatter: (params: any) => {
					let result = `<strong>Rok ${params[0].name}</strong><br/>`;
					params.forEach((param: any) => {
						result += `${param.seriesName}: ${param.value.toLocaleString('pl-PL')} PLN<br/>`;
					});
					return result;
				}
			},
			legend: {
				data: results.simulations.map((s) => s.account_name),
				bottom: 0
			},
			grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true },
			xAxis: { type: 'category', data: years, name: 'Rok' },
			yAxis: {
				type: 'value',
				name: 'Wartość (PLN)',
				axisLabel: {
					formatter: (value: number) => `${(value / 1000).toFixed(0)}k`
				}
			},
			series
		};

		chart.setOption(option);
	}

	onMount(() => {
		if (browser) {
			window.addEventListener('resize', () => chart?.resize());
		}

		return () => {
			chart?.dispose();
		};
	});

	function formatCurrency(value: number): string {
		return value.toLocaleString('pl-PL', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
	}
</script>

<div class="simulations-page">
	<h1>Symulacje Emerytalne</h1>

	<div class="content">
		<div class="form-section">
			<h2>Parametry symulacji</h2>

			<div class="form-group">
				<label>
					Obecny wiek
					<input type="number" bind:value={currentAge} min="18" max="100" />
				</label>
				<label>
					Wiek emerytalny
					<input type="number" bind:value={retirementAge} min="18" max="100" />
				</label>
			</div>

			<h3>Konta do symulacji</h3>
			<div class="accounts-grid">
				<div class="account-card">
					<label class="checkbox-label">
						<input type="checkbox" bind:checked={simulateIkeMarcin} />
						IKE (Marcin)
					</label>
					{#if simulateIkeMarcin}
						<label>
							Saldo obecne (PLN)
							<input type="number" bind:value={ikeMarcinBalance} min="0" step="100" />
						</label>
						<label class="checkbox-label">
							<input type="checkbox" bind:checked={ikeMarcinAutoFill} />
							Auto-wypełnienie limitu
						</label>
						{#if !ikeMarcinAutoFill}
							<label>
								Wpłata miesięczna (PLN)
								<input type="number" bind:value={ikeMarcinMonthly} min="0" step="100" />
							</label>
						{/if}
					{/if}
				</div>

				<div class="account-card">
					<label class="checkbox-label">
						<input type="checkbox" bind:checked={simulateIkeEwa} />
						IKE (Ewa)
					</label>
					{#if simulateIkeEwa}
						<label>
							Saldo obecne (PLN)
							<input type="number" bind:value={ikeEwaBalance} min="0" step="100" />
						</label>
						<label class="checkbox-label">
							<input type="checkbox" bind:checked={ikeEwaAutoFill} />
							Auto-wypełnienie limitu
						</label>
						{#if !ikeEwaAutoFill}
							<label>
								Wpłata miesięczna (PLN)
								<input type="number" bind:value={ikeEwaMonthly} min="0" step="100" />
							</label>
						{/if}
					{/if}
				</div>

				<div class="account-card">
					<label class="checkbox-label">
						<input type="checkbox" bind:checked={simulateIkzeMarcin} />
						IKZE (Marcin)
					</label>
					{#if simulateIkzeMarcin}
						<label>
							Saldo obecne (PLN)
							<input type="number" bind:value={ikzeMarcinBalance} min="0" step="100" />
						</label>
						<label class="checkbox-label">
							<input type="checkbox" bind:checked={ikzeMarcinAutoFill} />
							Auto-wypełnienie limitu
						</label>
						{#if !ikzeMarcinAutoFill}
							<label>
								Wpłata miesięczna (PLN)
								<input type="number" bind:value={ikzeMarcinMonthly} min="0" step="100" />
							</label>
						{/if}
						<label>
							Stawka podatkowa (%)
							<input type="number" bind:value={marcinTaxRate} min="0" max="50" step="1" />
						</label>
					{/if}
				</div>

				<div class="account-card">
					<label class="checkbox-label">
						<input type="checkbox" bind:checked={simulateIkzeEwa} />
						IKZE (Ewa)
					</label>
					{#if simulateIkzeEwa}
						<label>
							Saldo obecne (PLN)
							<input type="number" bind:value={ikzeEwaBalance} min="0" step="100" />
						</label>
						<label class="checkbox-label">
							<input type="checkbox" bind:checked={ikzeEwaAutoFill} />
							Auto-wypełnienie limitu
						</label>
						{#if !ikzeEwaAutoFill}
							<label>
								Wpłata miesięczna (PLN)
								<input type="number" bind:value={ikzeEwaMonthly} min="0" step="100" />
							</label>
						{/if}
						<label>
							Stawka podatkowa (%)
							<input type="number" bind:value={ewaTaxRate} min="0" max="50" step="1" />
						</label>
					{/if}
				</div>
			</div>

			<h3>Założenia</h3>
			<div class="form-group">
				<label>
					Roczna stopa zwrotu (%)
					<input type="number" bind:value={annualReturnRate} min="-50" max="50" step="0.1" />
				</label>
				<label>
					Wzrost limitów wpłat (%)
					<input type="number" bind:value={limitGrowthRate} min="0" max="20" step="0.1" />
				</label>
			</div>

			<button class="primary-button" on:click={runSimulation} disabled={loading}>
				{loading ? 'Obliczanie...' : 'Uruchom symulację'}
			</button>

			{#if error}
				<div class="error-message">{error}</div>
			{/if}
		</div>

		{#if results}
			<div class="results-section">
				<h2>Wyniki symulacji</h2>

				<div class="summary-cards">
					<div class="summary-card">
						<div class="card-label">Końcowy kapitał</div>
						<div class="card-value">{formatCurrency(results.summary.total_final_balance)} PLN</div>
					</div>
					<div class="summary-card">
						<div class="card-label">Miesięczny dochód (4% rule)</div>
						<div class="card-value">
							{formatCurrency(results.summary.estimated_monthly_income)} PLN
						</div>
					</div>
					<div class="summary-card">
						<div class="card-label">Miesięczny dochód (w dzisiejszych pieniądzach)</div>
						<div class="card-value">
							{formatCurrency(results.summary.estimated_monthly_income_today)} PLN
						</div>
						<div class="card-note">przy 3% inflacji rocznie</div>
					</div>
					<div class="summary-card">
						<div class="card-label">Suma wpłat</div>
						<div class="card-value">{formatCurrency(results.summary.total_contributions)} PLN</div>
					</div>
					<div class="summary-card">
						<div class="card-label">Zyski z inwestycji</div>
						<div class="card-value">{formatCurrency(results.summary.total_returns)} PLN</div>
					</div>
					{#if results.summary.total_tax_savings > 0}
						<div class="summary-card">
							<div class="card-label">Oszczędności podatkowe (IKZE)</div>
							<div class="card-value">
								{formatCurrency(results.summary.total_tax_savings)} PLN
							</div>
						</div>
					{/if}
				</div>

				<div class="chart-container" bind:this={chartContainer}></div>

				<h3>Szczegóły projekcji</h3>
				{#each results.simulations as simulation}
					<details class="account-details">
						<summary>
							<strong>{simulation.account_name}</strong> - Końcowa wartość: {formatCurrency(
								simulation.final_balance
							)} PLN
						</summary>
						<div class="projection-table">
							<table>
								<thead>
									<tr>
										<th>Rok</th>
										<th>Wiek</th>
										<th>Roczna wpłata</th>
										<th>Wykorzystanie limitu</th>
										<th>Saldo</th>
										<th>Suma wpłat</th>
										<th>Zyski</th>
										{#if simulation.account_name.includes('IKZE')}
											<th>Ulga podatkowa</th>
										{/if}
									</tr>
								</thead>
								<tbody>
									{#each simulation.yearly_projections as projection}
										<tr>
											<td>{projection.year}</td>
											<td>{projection.age}</td>
											<td>{formatCurrency(projection.annual_contribution)}</td>
											<td>{projection.limit_utilized_pct.toFixed(1)}%</td>
											<td>{formatCurrency(projection.balance_end_of_year)}</td>
											<td>{formatCurrency(projection.cumulative_contributions)}</td>
											<td>{formatCurrency(projection.cumulative_returns)}</td>
											{#if simulation.account_name.includes('IKZE')}
												<td>{formatCurrency(projection.tax_savings)}</td>
											{/if}
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</details>
				{/each}
			</div>
		{/if}
	</div>
</div>

<style>
	.simulations-page {
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
		grid-template-columns: 1fr 1fr;
		gap: var(--size-6);
	}

	.form-section,
	.results-section {
		background: var(--surface-2);
		padding: var(--size-5);
		border-radius: var(--radius-2);
	}

	h2,
	h3 {
		margin-top: var(--size-5);
		margin-bottom: var(--size-3);
		color: var(--color-text-2);
	}

	h2 {
		margin-top: 0;
	}

	.form-group {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--size-3);
		margin-bottom: var(--size-4);
	}

	.accounts-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--size-3);
		margin-bottom: var(--size-4);
	}

	.account-card {
		background: var(--surface-3);
		padding: var(--size-4);
		border-radius: var(--radius-2);
		display: flex;
		flex-direction: column;
		gap: var(--size-2);
	}

	label {
		display: flex;
		flex-direction: column;
		gap: var(--size-1);
		font-size: var(--font-size-1);
		color: var(--color-text-2);
	}

	.checkbox-label {
		flex-direction: row;
		align-items: center;
		font-weight: 600;
	}

	.checkbox-label input[type='checkbox'] {
		margin-right: var(--size-2);
	}

	input[type='number'] {
		padding: var(--size-2);
		border: 1px solid var(--surface-4);
		border-radius: var(--radius-2);
		background: var(--surface-1);
		color: var(--color-text-1);
		font-size: var(--font-size-1);
	}

	input[type='checkbox'] {
		width: var(--size-4);
		height: var(--size-4);
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
		margin-top: var(--size-4);
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

	.summary-cards {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
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
		font-size: var(--font-size-4);
		font-weight: 700;
		color: var(--color-text-1);
	}

	.card-note {
		font-size: var(--font-size-0);
		color: var(--color-text-3);
		margin-top: var(--size-1);
	}

	.chart-container {
		width: 100%;
		height: 400px;
		margin-bottom: var(--size-5);
	}

	.account-details {
		background: var(--surface-3);
		padding: var(--size-3);
		border-radius: var(--radius-2);
		margin-bottom: var(--size-3);
	}

	.account-details summary {
		cursor: pointer;
		padding: var(--size-2);
		font-size: var(--font-size-2);
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

	@media (max-width: 1200px) {
		.content {
			grid-template-columns: 1fr;
		}
	}

	@media (max-width: 768px) {
		.accounts-grid {
			grid-template-columns: 1fr;
		}

		.form-group {
			grid-template-columns: 1fr;
		}
	}
</style>
