<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { browser } from '$app/environment';
	import { env } from '$env/dynamic/public';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';

	export let data: {
		prefill: {
			birth_date: string | null;
			retirement_age: number;
			gender: string;
			current_gross_monthly_salary: number | null;
			owner: string | null;
			salary_history: { year: number; annual_gross: number }[];
			work_start_year: number | null;
		};
		personas: { name: string }[];
	};

	interface ZusYearlyProjection {
		year: number;
		age: number;
		annual_gross_salary: number;
		salary_capped: boolean;
		contribution_konto: number;
		contribution_subkonto: number;
		konto_balance: number;
		subkonto_balance: number;
		total_balance: number;
	}

	interface ZusSensitivityScenario {
		label: string;
		valorization_konto: number;
		valorization_subkonto: number;
		monthly_pension_gross: number;
		monthly_pension_net: number;
		replacement_rate: number;
	}

	interface ZusCalculatorResponse {
		yearly_projections: ZusYearlyProjection[];
		life_expectancy_months: number;
		konto_at_retirement: number;
		subkonto_at_retirement: number;
		kapital_poczatkowy_valorized: number;
		total_capital: number;
		monthly_pension_gross: number;
		monthly_pension_net: number;
		replacement_rate: number;
		last_gross_salary: number;
		sensitivity: ZusSensitivityScenario[];
	}

	// Form state
	let owner = data.prefill.owner ?? data.personas[0]?.name ?? '';
	let birthDate = data.prefill.birth_date ?? '';
	let gender = data.prefill.gender ?? 'M';
	let retirementAge = data.prefill.retirement_age ?? 65;
	let currentGrossMonthly = data.prefill.current_gross_monthly_salary ?? 10000;
	let salaryGrowthRate = 3.0;
	let inflationRate = 3.0;
	let valorizationKonto = 5.0;
	let valorizationSubkonto = 4.0;
	let hasOfe = false;
	let kapitalPoczatkowy = 0;
	let workStartYear = data.prefill.work_start_year ?? 2015;

	// Results
	let results: ZusCalculatorResponse | null = null;
	let loading = false;
	let error = '';

	// Chart
	let chartContainer: HTMLDivElement;
	let chart: echarts.ECharts | null = null;

	async function runCalculation() {
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

			const response = await fetch(`${apiUrl}/api/zus/calculate`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					owner,
					birth_date: birthDate,
					gender,
					retirement_age: retirementAge,
					current_gross_monthly_salary: currentGrossMonthly,
					salary_growth_rate: salaryGrowthRate,
					inflation_rate: inflationRate,
					valorization_rate_konto: valorizationKonto,
					valorization_rate_subkonto: valorizationSubkonto,
					has_ofe: hasOfe,
					kapital_poczatkowy: kapitalPoczatkowy,
					work_start_year: workStartYear,
					salary_history: data.prefill.salary_history
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
			console.error('ZUS calculation failed:', err);
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

		const years = results.yearly_projections.map((r) => r.year.toString());
		const kontoData = results.yearly_projections.map((r) => r.konto_balance);
		const subkontoData = results.yearly_projections.map((r) => r.subkonto_balance);

		const option: EChartsOption = {
			title: { text: 'Wzrost kapitału ZUS' },
			tooltip: {
				trigger: 'axis',
				formatter: (params: any) => {
					let result = `<strong>Rok ${params[0].name}</strong><br/>`;
					params.forEach((p: any) => {
						result += `${p.seriesName}: ${formatCurrency(p.value)} PLN<br/>`;
					});
					return result;
				}
			},
			legend: { data: ['Konto', 'Subkonto'], bottom: 0 },
			grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true },
			xAxis: {
				type: 'category',
				data: years,
				axisLabel: {
					interval: Math.max(0, Math.floor(years.length / 10) - 1)
				}
			},
			yAxis: {
				type: 'value',
				name: 'Wartość (PLN)',
				axisLabel: { formatter: (v: number) => `${(v / 1000).toFixed(0)}k` }
			},
			series: [
				{
					name: 'Konto',
					type: 'bar',
					stack: 'total',
					data: kontoData,
					itemStyle: { color: '#5E81AC' }
				},
				{
					name: 'Subkonto',
					type: 'bar',
					stack: 'total',
					data: subkontoData,
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
</script>

<div class="zus-page">
	<h1>Kalkulator emerytury ZUS</h1>

	<div class="content">
		<div class="form-section">
			<h2>Parametry</h2>

			<div class="form-group">
				<label>
					Osoba
					<select bind:value={owner}>
						{#each data.personas as persona}
							<option value={persona.name}>{persona.name}</option>
						{/each}
					</select>
				</label>

				<label>
					Data urodzenia
					<input type="date" bind:value={birthDate} />
				</label>

				<label>
					Płeć
					<select bind:value={gender}>
						<option value="M">Mężczyzna</option>
						<option value="F">Kobieta</option>
					</select>
				</label>

				<label>
					Wiek emerytalny
					<input type="number" bind:value={retirementAge} min="55" max="70" step="1" />
				</label>

				<label>
					Aktualne wynagrodzenie brutto (PLN/mies.)
					<input type="number" bind:value={currentGrossMonthly} min="0" step="500" />
				</label>

				<label>
					Rok rozpoczęcia pracy
					<input type="number" bind:value={workStartYear} min="1970" max="2030" step="1" />
				</label>

				<label>
					Kapitał początkowy (PLN)
					<input type="number" bind:value={kapitalPoczatkowy} min="0" step="1000" />
					<small>Dla pracy przed 1999 r.</small>
				</label>
			</div>

			<h3>Założenia</h3>
			<div class="form-group">
				<label>
					Wzrost wynagrodzeń (% rocznie)
					<input type="number" bind:value={salaryGrowthRate} min="0" max="20" step="0.5" />
				</label>

				<label>
					Waloryzacja konto (%)
					<input type="number" bind:value={valorizationKonto} min="0" max="20" step="0.5" />
				</label>

				<label>
					Waloryzacja subkonto (%)
					<input type="number" bind:value={valorizationSubkonto} min="0" max="20" step="0.5" />
				</label>

				<label>
					Inflacja (% rocznie)
					<input type="number" bind:value={inflationRate} min="0" max="20" step="0.5" />
				</label>

				<label class="checkbox-label">
					<input type="checkbox" bind:checked={hasOfe} />
					Członek OFE
					<small>Zmniejsza składkę na subkonto (4.38% zamiast 7.30%)</small>
				</label>
			</div>

			{#if data.prefill.salary_history.length > 0}
				<details class="salary-history">
					<summary>Historia wynagrodzeń ({data.prefill.salary_history.length} lat)</summary>
					<div class="history-list">
						{#each data.prefill.salary_history as entry}
							<div class="history-entry">
								<span>{entry.year}</span>
								<span>{formatCurrency(entry.annual_gross)} PLN/rok</span>
							</div>
						{/each}
					</div>
				</details>
			{/if}

			<button class="primary-button" on:click={runCalculation} disabled={loading}>
				{loading ? 'Obliczanie...' : 'Oblicz emeryturę'}
			</button>

			{#if error}
				<div class="error-message">{error}</div>
			{/if}
		</div>

		{#if results}
			<div class="results-section">
				<h2>Wyniki</h2>

				<div class="summary-cards">
					<div class="summary-card highlight">
						<div class="card-label">Emerytura brutto</div>
						<div class="card-value">
							{formatCurrency(results.monthly_pension_gross)} PLN
						</div>
						<div class="card-note">miesięcznie</div>
					</div>
					<div class="summary-card highlight">
						<div class="card-label">Emerytura netto</div>
						<div class="card-value">
							{formatCurrency(results.monthly_pension_net)} PLN
						</div>
						<div class="card-note">miesięcznie</div>
					</div>
					<div class="summary-card">
						<div class="card-label">Stopa zastąpienia</div>
						<div class="card-value">{results.replacement_rate.toFixed(1)}%</div>
						<div class="card-note">
							emerytura / ostatnia pensja ({formatCurrency(results.last_gross_salary)} PLN)
						</div>
					</div>
					<div class="summary-card">
						<div class="card-label">Kapitał łączny</div>
						<div class="card-value">{formatCurrency(results.total_capital)} PLN</div>
					</div>
					<div class="summary-card">
						<div class="card-label">Konto indywidualne</div>
						<div class="card-value">
							{formatCurrency(results.konto_at_retirement)} PLN
						</div>
					</div>
					<div class="summary-card">
						<div class="card-label">Subkonto</div>
						<div class="card-value">
							{formatCurrency(results.subkonto_at_retirement)} PLN
						</div>
					</div>
					<div class="summary-card">
						<div class="card-label">Kapitał początkowy (zwal.)</div>
						<div class="card-value">
							{formatCurrency(results.kapital_poczatkowy_valorized)} PLN
						</div>
					</div>
					<div class="summary-card">
						<div class="card-label">Dalsze trwanie życia</div>
						<div class="card-value">{results.life_expectancy_months.toFixed(1)} mies.</div>
						<div class="card-note">
							{(results.life_expectancy_months / 12).toFixed(1)} lat
						</div>
					</div>
				</div>

				<div class="chart-container" bind:this={chartContainer}></div>

				<h3>Analiza wrażliwości</h3>
				<div class="sensitivity-table">
					<table>
						<thead>
							<tr>
								<th>Scenariusz</th>
								<th>Wal. konto</th>
								<th>Wal. subkonto</th>
								<th>Emerytura brutto</th>
								<th>Emerytura netto</th>
								<th>Stopa zastąpienia</th>
							</tr>
						</thead>
						<tbody>
							{#each results.sensitivity as scenario}
								<tr>
									<td>{scenario.label}</td>
									<td>{scenario.valorization_konto.toFixed(1)}%</td>
									<td>{scenario.valorization_subkonto.toFixed(1)}%</td>
									<td>{formatCurrency(scenario.monthly_pension_gross)} PLN</td>
									<td>{formatCurrency(scenario.monthly_pension_net)} PLN</td>
									<td>{scenario.replacement_rate.toFixed(1)}%</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<details class="projection-details">
					<summary>Projekcja roczna</summary>
					<div class="projection-table">
						<table>
							<thead>
								<tr>
									<th>Rok</th>
									<th>Wiek</th>
									<th>Wynagrodzenie</th>
									<th>Limit</th>
									<th>Składka konto</th>
									<th>Składka subkonto</th>
									<th>Saldo konto</th>
									<th>Saldo subkonto</th>
									<th>Razem</th>
								</tr>
							</thead>
							<tbody>
								{#each results.yearly_projections as row}
									<tr class:capped={row.salary_capped}>
										<td>{row.year}</td>
										<td>{row.age}</td>
										<td>{formatCurrency(row.annual_gross_salary)}</td>
										<td>{row.salary_capped ? '30x' : '—'}</td>
										<td>{formatCurrency(row.contribution_konto)}</td>
										<td>{formatCurrency(row.contribution_subkonto)}</td>
										<td>{formatCurrency(row.konto_balance)}</td>
										<td>{formatCurrency(row.subkonto_balance)}</td>
										<td>{formatCurrency(row.total_balance)}</td>
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
	.zus-page {
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

	h3 {
		margin-top: var(--size-4);
		margin-bottom: var(--size-3);
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

	input[type='number'],
	input[type='date'],
	select {
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

	.salary-history {
		margin-bottom: var(--size-4);
		background: var(--surface-3);
		padding: var(--size-3);
		border-radius: var(--radius-2);
	}

	.salary-history summary {
		cursor: pointer;
		font-size: var(--font-size-1);
		font-weight: 600;
	}

	.history-list {
		margin-top: var(--size-2);
	}

	.history-entry {
		display: flex;
		justify-content: space-between;
		padding: var(--size-1) 0;
		font-size: var(--font-size-0);
		border-bottom: 1px solid var(--surface-4);
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

	.summary-card.highlight {
		border-left: 3px solid hsl(140 60% 45%);
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

	.chart-container {
		width: 100%;
		height: 380px;
		margin-bottom: var(--size-5);
	}

	.sensitivity-table,
	.projection-table {
		overflow-x: auto;
		margin-bottom: var(--size-4);
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

	tr.capped {
		background: hsl(40 80% 95%);
	}

	@media (max-width: 640px) {
		table {
			font-size: var(--font-size-0);
		}

		th,
		td {
			padding: var(--size-1);
			white-space: normal;
			word-break: break-word;
		}

		th:first-child,
		td:first-child {
			position: sticky;
			left: 0;
			background: var(--surface-2);
			z-index: 1;
		}

		th:first-child {
			background: var(--surface-4);
		}
	}

	@media (max-width: 1024px) {
		.content {
			grid-template-columns: 1fr;
		}
	}

	@media (max-width: 640px) {
		.zus-page {
			padding: var(--size-3);
		}

		.form-section,
		.results-section {
			padding: var(--size-4);
		}

		.chart-container {
			height: 280px;
		}

		.summary-cards {
			grid-template-columns: 1fr 1fr;
			gap: var(--size-2);
		}

		.card-value {
			font-size: var(--font-size-2);
		}

		.primary-button {
			min-height: var(--tap-target-min);
		}

		h1 {
			font-size: var(--font-size-4);
		}
	}
</style>
