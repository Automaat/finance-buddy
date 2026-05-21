<script lang="ts">
	import { onMount, tick, untrack } from 'svelte';
	import { browser } from '$app/environment';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import { getApiUrlOrThrow } from '$lib/utils/api';

	interface Props {
		data: {
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
	}

	let { data }: Props = $props();

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
	let owner = $state(untrack(() => data.prefill.owner ?? data.personas[0]?.name ?? ''));
	let birthDate = $state(untrack(() => data.prefill.birth_date ?? ''));
	let gender = $state(untrack(() => data.prefill.gender ?? 'M'));
	let retirementAge = $state(untrack(() => data.prefill.retirement_age ?? 65));
	let currentGrossMonthly = $state(
		untrack(() => data.prefill.current_gross_monthly_salary ?? 10000)
	);
	let salaryGrowthRate = $state(3.0);
	let inflationRate = $state(3.0);
	let valorizationKonto = $state(5.0);
	let valorizationSubkonto = $state(4.0);
	let hasOfe = $state(false);
	let kapitalPoczatkowy = $state(0);
	let workStartYear = $state(untrack(() => data.prefill.work_start_year ?? 2015));

	// Results
	let results: ZusCalculatorResponse | null = $state(null);
	let loading = $state(false);
	let error = $state('');

	// Chart
	let chartContainer: HTMLDivElement | undefined = $state();
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
			const apiUrl = getApiUrlOrThrow();

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

<div class="space-y-4">
	<h1 class="h1">Kalkulator emerytury ZUS</h1>

	<div class="grid grid-cols-1 lg:grid-cols-[400px_1fr] gap-6 items-start">
		<div class="card preset-filled-surface-100-900 p-5 space-y-4">
			<h2 class="h3">Parametry</h2>

			<div class="flex flex-col gap-3">
				<label class="label">
					<span class="text-sm font-semibold">Osoba</span>
					<select bind:value={owner} class="select">
						{#each data.personas as persona}
							<option value={persona.name}>{persona.name}</option>
						{/each}
					</select>
				</label>

				<label class="label">
					<span class="text-sm font-semibold">Data urodzenia</span>
					<input type="date" bind:value={birthDate} class="input" />
				</label>

				<label class="label">
					<span class="text-sm font-semibold">Płeć</span>
					<select bind:value={gender} class="select">
						<option value="M">Mężczyzna</option>
						<option value="F">Kobieta</option>
					</select>
				</label>

				<label class="label">
					<span class="text-sm font-semibold">Wiek emerytalny</span>
					<input
						type="number"
						bind:value={retirementAge}
						min="55"
						max="70"
						step="1"
						class="input"
					/>
				</label>

				<label class="label">
					<span class="text-sm font-semibold">Aktualne wynagrodzenie brutto (PLN/mies.)</span>
					<input type="number" bind:value={currentGrossMonthly} min="0" step="500" class="input" />
				</label>

				<label class="label">
					<span class="text-sm font-semibold">Rok rozpoczęcia pracy</span>
					<input
						type="number"
						bind:value={workStartYear}
						min="1970"
						max="2030"
						step="1"
						class="input"
					/>
				</label>

				<label class="label">
					<span class="text-sm font-semibold">Kapitał początkowy (PLN)</span>
					<input type="number" bind:value={kapitalPoczatkowy} min="0" step="1000" class="input" />
					<span class="text-xs text-surface-600-400">Dla pracy przed 1999 r.</span>
				</label>
			</div>

			<h3 class="h4">Założenia</h3>
			<div class="flex flex-col gap-3">
				<label class="label">
					<span class="text-sm font-semibold">Wzrost wynagrodzeń (% rocznie)</span>
					<input
						type="number"
						bind:value={salaryGrowthRate}
						min="0"
						max="20"
						step="0.5"
						class="input"
					/>
				</label>

				<label class="label">
					<span class="text-sm font-semibold">Waloryzacja konto (%)</span>
					<input
						type="number"
						bind:value={valorizationKonto}
						min="0"
						max="20"
						step="0.5"
						class="input"
					/>
				</label>

				<label class="label">
					<span class="text-sm font-semibold">Waloryzacja subkonto (%)</span>
					<input
						type="number"
						bind:value={valorizationSubkonto}
						min="0"
						max="20"
						step="0.5"
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
						step="0.5"
						class="input"
					/>
				</label>

				<div class="space-y-1">
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="checkbox" bind:checked={hasOfe} class="checkbox" />
						<span class="text-sm font-semibold">Członek OFE</span>
					</label>
					<p class="text-xs text-surface-600-400 ml-6">
						Zmniejsza składkę na subkonto (4.38% zamiast 7.30%)
					</p>
				</div>
			</div>

			{#if data.prefill.salary_history.length > 0}
				<details class="card preset-tonal-surface p-3">
					<summary class="cursor-pointer text-sm font-semibold">
						Historia wynagrodzeń ({data.prefill.salary_history.length} lat)
					</summary>
					<div class="mt-2 flex flex-col divide-y divide-surface-200-800">
						{#each data.prefill.salary_history as entry}
							<div class="flex justify-between py-1 text-xs">
								<span>{entry.year}</span>
								<span>{formatCurrency(entry.annual_gross)} PLN/rok</span>
							</div>
						{/each}
					</div>
				</details>
			{/if}

			<button
				class="btn preset-filled-primary-500 w-full"
				onclick={runCalculation}
				disabled={loading}
			>
				{loading ? 'Obliczanie...' : 'Oblicz emeryturę'}
			</button>

			{#if error}
				<div class="card preset-filled-error-500 p-3 text-sm">{error}</div>
			{/if}
		</div>

		{#if results}
			<div class="card preset-filled-surface-100-900 p-5 space-y-4">
				<h2 class="h3">Wyniki</h2>

				<div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
					<div class="card preset-tonal-surface p-4 border-l-4 border-l-success-500">
						<div class="text-xs text-surface-600-400 mb-2">Emerytura brutto</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.monthly_pension_gross)} PLN
						</div>
						<div class="text-xs text-surface-600-400 mt-1">miesięcznie</div>
					</div>
					<div class="card preset-tonal-surface p-4 border-l-4 border-l-success-500">
						<div class="text-xs text-surface-600-400 mb-2">Emerytura netto</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.monthly_pension_net)} PLN
						</div>
						<div class="text-xs text-surface-600-400 mt-1">miesięcznie</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">Stopa zastąpienia</div>
						<div class="text-xl font-bold">{results.replacement_rate.toFixed(1)}%</div>
						<div class="text-xs text-surface-600-400 mt-1">
							emerytura / ostatnia pensja ({formatCurrency(results.last_gross_salary)} PLN)
						</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">Kapitał łączny</div>
						<div class="text-xl font-bold">{formatCurrency(results.total_capital)} PLN</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">Konto indywidualne</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.konto_at_retirement)} PLN
						</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">Subkonto</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.subkonto_at_retirement)} PLN
						</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">Kapitał początkowy (zwal.)</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.kapital_poczatkowy_valorized)} PLN
						</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">Dalsze trwanie życia</div>
						<div class="text-xl font-bold">{results.life_expectancy_months.toFixed(1)} mies.</div>
						<div class="text-xs text-surface-600-400 mt-1">
							{(results.life_expectancy_months / 12).toFixed(1)} lat
						</div>
					</div>
				</div>

				<div bind:this={chartContainer} class="w-full h-[280px] sm:h-[380px]"></div>

				<h3 class="h4">Analiza wrażliwości</h3>
				<div class="table-wrap">
					<table class="table table-hover text-xs">
						<thead>
							<tr>
								<th>Scenariusz</th>
								<th class="text-right">Wal. konto</th>
								<th class="text-right">Wal. subkonto</th>
								<th class="text-right">Emerytura brutto</th>
								<th class="text-right">Emerytura netto</th>
								<th class="text-right">Stopa zastąpienia</th>
							</tr>
						</thead>
						<tbody>
							{#each results.sensitivity as scenario}
								<tr>
									<td>{scenario.label}</td>
									<td class="text-right">{scenario.valorization_konto.toFixed(1)}%</td>
									<td class="text-right">{scenario.valorization_subkonto.toFixed(1)}%</td>
									<td class="text-right">{formatCurrency(scenario.monthly_pension_gross)} PLN</td>
									<td class="text-right">{formatCurrency(scenario.monthly_pension_net)} PLN</td>
									<td class="text-right">{scenario.replacement_rate.toFixed(1)}%</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<details class="card preset-tonal-surface p-3">
					<summary class="cursor-pointer text-sm font-semibold py-1">Projekcja roczna</summary>
					<div class="table-wrap mt-3">
						<table class="table table-hover text-xs">
							<thead>
								<tr>
									<th class="sticky left-0 z-10 bg-surface-100 dark:bg-surface-900">Rok</th>
									<th class="text-right">Wiek</th>
									<th class="text-right">Wynagrodzenie</th>
									<th class="text-right">Limit</th>
									<th class="text-right">Składka konto</th>
									<th class="text-right">Składka subkonto</th>
									<th class="text-right">Saldo konto</th>
									<th class="text-right">Saldo subkonto</th>
									<th class="text-right">Razem</th>
								</tr>
							</thead>
							<tbody>
								{#each results.yearly_projections as row}
									<tr class={row.salary_capped ? 'bg-warning-500/10' : ''}>
										<td class="sticky left-0 z-10 bg-surface-100 dark:bg-surface-900">{row.year}</td
										>
										<td class="text-right">{row.age}</td>
										<td class="text-right">{formatCurrency(row.annual_gross_salary)}</td>
										<td class="text-right">{row.salary_capped ? '30x' : '—'}</td>
										<td class="text-right">{formatCurrency(row.contribution_konto)}</td>
										<td class="text-right">{formatCurrency(row.contribution_subkonto)}</td>
										<td class="text-right">{formatCurrency(row.konto_balance)}</td>
										<td class="text-right">{formatCurrency(row.subkonto_balance)}</td>
										<td class="text-right">{formatCurrency(row.total_balance)}</td>
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
