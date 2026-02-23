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
		government_subsidies?: number;
		monthly_salary?: number;
		return_rate?: number;
	}

	interface AccountSimulation {
		account_name: string;
		starting_balance: number;
		total_contributions: number;
		total_returns: number;
		total_tax_savings: number;
		total_subsidies?: number;
		final_balance: number;
		yearly_projections: YearlyProjection[];
	}

	interface SimulationSummary {
		total_final_balance: number;
		total_contributions: number;
		total_returns: number;
		total_tax_savings: number;
		total_subsidies?: number;
		estimated_monthly_income: number;
		estimated_monthly_income_today: number;
		years_until_retirement: number;
	}

	interface SimulationResponse {
		simulations: AccountSimulation[];
		summary: SimulationSummary;
	}

	interface IkeIkzeConfig {
		enabled: boolean;
		wrapper: string;
		owner: string;
		balance: number;
		autoFill: boolean;
		monthly: number;
		taxRate: number;
	}

	interface PpkConfig {
		enabled: boolean;
		owner: string;
		balance: number;
		salary: number;
		employeeRate: number;
		employerRate: number;
		belowThreshold: boolean;
		includeSubsidies: boolean;
	}

	interface BrokerageConfig {
		enabled: boolean;
		owner: string;
		balance: number;
		monthly: number;
	}

	const SALARY_THRESHOLD_2026 = 5767;

	export let data: PageData;

	$: personas = (data.personas || []) as Array<{ name: string }>;

	let currentAge = data.current_age;
	let retirementAge = data.retirement_age;

	// Dynamic IKE/IKZE accounts - one per persona per wrapper
	let ikeIkzeAccounts: IkeIkzeConfig[] = [];
	let ppkAccounts: PpkConfig[] = [];
	let brokerageAccounts: BrokerageConfig[] = [];

	function initAccounts() {
		ikeIkzeAccounts = [];
		ppkAccounts = [];
		brokerageAccounts = [];

		for (const persona of personas) {
			const ownerLower = persona.name.toLowerCase();

			for (const wrapper of ['IKE', 'IKZE']) {
				const balanceKey = `${wrapper.toLowerCase()}_${ownerLower}`;
				ikeIkzeAccounts.push({
					enabled: true,
					wrapper,
					owner: persona.name,
					balance: data.balances?.[balanceKey] ?? 0,
					autoFill: false,
					monthly: 0,
					taxRate: wrapper === 'IKZE' ? 17.0 : 0
				});
			}

			ppkAccounts.push({
				enabled: false,
				owner: persona.name,
				balance: data.ppk_balances?.[ownerLower] ?? 0,
				salary: data.monthly_salaries?.[ownerLower] ?? 10000,
				employeeRate: data.ppk_rates?.[ownerLower]?.employee ?? 2.0,
				employerRate: data.ppk_rates?.[ownerLower]?.employer ?? 1.5,
				belowThreshold: false,
				includeSubsidies: true
			});

			brokerageAccounts.push({
				enabled: false,
				owner: persona.name,
				balance: 0,
				monthly: 0
			});
		}
	}

	$: if (personas.length > 0 && ikeIkzeAccounts.length === 0) {
		initAccounts();
	}

	// Assumptions
	let annualReturnRate = 7.0;
	let limitGrowthRate = 5.0;
	let expectedSalaryGrowth = 3.0;
	let inflationRate = 3.0;

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
			// Validate PPK inputs
			for (const ppk of ppkAccounts) {
				if (!ppk.enabled) continue;
				if (ppk.salary > SALARY_THRESHOLD_2026 && ppk.belowThreshold) {
					error = `PPK ${ppk.owner}: Wynagrodzenie przekracza próg (${SALARY_THRESHOLD_2026} PLN)`;
					loading = false;
					return;
				}
				if (ppk.employeeRate < 0.5 || ppk.employeeRate > 4.0) {
					error = `PPK ${ppk.owner}: Składka pracownika musi być w zakresie 0.5-4%`;
					loading = false;
					return;
				}
				if (ppk.employerRate < 1.5 || ppk.employerRate > 4.0) {
					error = `PPK ${ppk.owner}: Składka pracodawcy musi być w zakresie 1.5-4%`;
					loading = false;
					return;
				}
			}

			const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
			if (!apiUrl) throw new Error('API URL not configured');

			const requestBody = {
				current_age: currentAge,
				retirement_age: retirementAge,
				ike_ikze_accounts: ikeIkzeAccounts.map((a) => ({
					enabled: a.enabled,
					wrapper: a.wrapper,
					owner: a.owner,
					balance: a.balance,
					auto_fill_limit: a.autoFill,
					monthly_contribution: a.monthly,
					tax_rate: a.taxRate
				})),
				ppk_accounts: ppkAccounts
					.filter((p) => p.enabled)
					.map((p) => ({
						owner: p.owner,
						enabled: true,
						starting_balance: p.balance,
						monthly_gross_salary: p.salary,
						employee_rate: p.employeeRate,
						employer_rate: p.employerRate,
						salary_below_threshold: p.belowThreshold,
						include_welcome_bonus: p.includeSubsidies,
						include_annual_subsidy: p.includeSubsidies
					})),
				brokerage_accounts: brokerageAccounts.map((b) => ({
					enabled: b.enabled,
					owner: b.owner,
					balance: b.balance,
					monthly_contribution: b.monthly
				})),
				annual_return_rate: annualReturnRate,
				limit_growth_rate: limitGrowthRate,
				expected_salary_growth: expectedSalaryGrowth,
				inflation_rate: inflationRate
			};

			const response = await fetch(`${apiUrl}/api/simulations/retirement`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(requestBody)
			});

			if (!response.ok) {
				throw new Error(`Simulation failed: ${response.statusText}`);
			}

			const responseData = await response.json();
			results = responseData;
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

		if (results.simulations.length === 0) {
			chart.clear();
			return;
		}

		const years = results.simulations[0].yearly_projections.map((p) => p.year);
		const series = results.simulations.map((sim, idx) => {
			const colors = ['#5E81AC', '#81A1C1', '#88C0D0', '#8FBCBB', '#B48EAD', '#A3BE8C'];
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
		const handleResize = () => chart?.resize();

		if (browser) {
			window.addEventListener('resize', handleResize);
		}

		return () => {
			if (browser) {
				window.removeEventListener('resize', handleResize);
			}
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
				{#each ikeIkzeAccounts as account, i}
					<div class="account-card">
						<label class="checkbox-label">
							<input type="checkbox" bind:checked={ikeIkzeAccounts[i].enabled} />
							{account.wrapper} ({account.owner})
						</label>
						{#if account.enabled}
							<label>
								Saldo obecne (PLN)
								<input type="number" bind:value={ikeIkzeAccounts[i].balance} min="0" step="100" />
							</label>
							<label class="checkbox-label">
								<input type="checkbox" bind:checked={ikeIkzeAccounts[i].autoFill} />
								Auto-wypełnienie limitu
							</label>
							{#if !account.autoFill}
								<label>
									Wpłata miesięczna (PLN)
									<input type="number" bind:value={ikeIkzeAccounts[i].monthly} min="0" step="100" />
								</label>
							{/if}
							{#if account.wrapper === 'IKZE'}
								<label>
									Stawka podatkowa (%)
									<input
										type="number"
										bind:value={ikeIkzeAccounts[i].taxRate}
										min="0"
										max="50"
										step="1"
									/>
								</label>
							{/if}
						{/if}
					</div>
				{/each}

				{#each ppkAccounts as ppk, i}
					<div class="account-card">
						<label class="checkbox-label">
							<input type="checkbox" bind:checked={ppkAccounts[i].enabled} />
							PPK ({ppk.owner})
						</label>
						{#if ppk.enabled}
							<label>
								Obecna wartość (PLN)
								<input type="number" bind:value={ppkAccounts[i].balance} min="0" step="1000" />
							</label>
							<label>
								Miesięczne wynagrodzenie brutto (PLN)
								<input type="number" bind:value={ppkAccounts[i].salary} min="1000" step="500" />
							</label>
							<div class="contribution-rates">
								<label>
									Składka pracownika (%)
									<input
										type="number"
										bind:value={ppkAccounts[i].employeeRate}
										min="0.5"
										max="4"
										step="0.5"
									/>
									<small>Zakres: 0.5-4% (podstawa: 2%)</small>
								</label>
								<label>
									Składka pracodawcy (%)
									<input
										type="number"
										bind:value={ppkAccounts[i].employerRate}
										min="1.5"
										max="4"
										step="0.5"
									/>
									<small>Zakres: 1.5-4% (podstawa: 1.5%)</small>
								</label>
							</div>
							<label class="checkbox-label">
								<input type="checkbox" bind:checked={ppkAccounts[i].belowThreshold} />
								Wynagrodzenie poniżej progu ({SALARY_THRESHOLD_2026} PLN)
								<small>Dotyczy dopłaty rocznej 240 PLN</small>
							</label>
							<label class="checkbox-label">
								<input type="checkbox" bind:checked={ppkAccounts[i].includeSubsidies} />
								Uwzględnij dopłaty państwa (250 PLN + 240 PLN/rok)
							</label>
							<div class="contribution-estimate">
								<small>
									Szacowana miesięczna składka:
									{formatCurrency((ppk.salary * (ppk.employeeRate + ppk.employerRate)) / 100)}
									PLN (pracownik: {formatCurrency((ppk.salary * ppk.employeeRate) / 100)} PLN, pracodawca:
									{formatCurrency((ppk.salary * ppk.employerRate) / 100)} PLN)
								</small>
							</div>
						{/if}
					</div>
				{/each}

				{#each brokerageAccounts as brokerage, i}
					<div class="account-card">
						<label class="checkbox-label">
							<input type="checkbox" bind:checked={brokerageAccounts[i].enabled} />
							Rachunek maklerski ({brokerage.owner})
						</label>
						{#if brokerage.enabled}
							<label>
								Obecna wartość (PLN)
								<input
									type="number"
									bind:value={brokerageAccounts[i].balance}
									min="0"
									step="1000"
								/>
							</label>
							<label>
								Wpłata miesięczna (PLN)
								<input type="number" bind:value={brokerageAccounts[i].monthly} min="0" step="100" />
							</label>
							<div class="card-note">
								<small
									>Rachunki maklerskie są opodatkowane 19% podatkiem Belki od zysków kapitałowych</small
								>
							</div>
						{/if}
					</div>
				{/each}
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
				<label>
					Przewidywany wzrost wynagrodzeń (%)
					<input type="number" bind:value={expectedSalaryGrowth} min="0" max="10" step="0.5" />
					<small>Roczny wzrost płacy brutto (typowo 3-5%)</small>
				</label>
				<label>
					Inflacja (%)
					<input type="number" bind:value={inflationRate} min="0" max="20" step="0.1" />
					<small>Roczna inflacja do przeliczenia dochodu na dzisiejsze pieniądze</small>
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
						<div class="card-value">
							{formatCurrency(results.summary.total_final_balance)} PLN
						</div>
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
						<div class="card-value">
							{formatCurrency(results.summary.total_contributions)} PLN
						</div>
					</div>
					<div class="summary-card">
						<div class="card-label">Zyski z inwestycji</div>
						<div class="card-value">
							{formatCurrency(results.summary.total_returns)} PLN
						</div>
					</div>
					{#if results.summary.total_tax_savings > 0}
						<div class="summary-card">
							<div class="card-label">Oszczędności podatkowe (IKZE)</div>
							<div class="card-value">
								{formatCurrency(results.summary.total_tax_savings)} PLN
							</div>
						</div>
					{/if}
					{#if results.summary.total_subsidies && results.summary.total_subsidies > 0}
						<div class="summary-card">
							<div class="card-label">Dopłaty państwa (PPK)</div>
							<div class="card-value">
								{formatCurrency(results.summary.total_subsidies)} PLN
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
										{#if !simulation.account_name.startsWith('PPK')}
											<th>Wykorzystanie limitu</th>
										{/if}
										<th>Saldo</th>
										<th>Suma wpłat</th>
										<th>Zyski</th>
										{#if simulation.account_name.includes('IKZE')}
											<th>Ulga podatkowa</th>
										{/if}
										{#if simulation.account_name.startsWith('PPK')}
											<th>Dopłaty państwa</th>
											<th>Roczne wynagrodzenie</th>
											<th>Stopa zwrotu</th>
										{/if}
									</tr>
								</thead>
								<tbody>
									{#each simulation.yearly_projections as projection}
										<tr>
											<td>{projection.year}</td>
											<td>{projection.age}</td>
											<td>{formatCurrency(projection.annual_contribution)}</td>
											{#if !simulation.account_name.startsWith('PPK')}
												<td>{projection.limit_utilized_pct.toFixed(1)}%</td>
											{/if}
											<td>{formatCurrency(projection.balance_end_of_year)}</td>
											<td>{formatCurrency(projection.cumulative_contributions)}</td>
											<td>{formatCurrency(projection.cumulative_returns)}</td>
											{#if simulation.account_name.includes('IKZE')}
												<td>{formatCurrency(projection.tax_savings)}</td>
											{/if}
											{#if simulation.account_name.startsWith('PPK')}
												<td>{formatCurrency(projection.government_subsidies || 0)}</td>
												<td>
													{formatCurrency((projection.monthly_salary || 0) * 12)}
												</td>
												<td>{(projection.return_rate || 0).toFixed(1)}%</td>
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
