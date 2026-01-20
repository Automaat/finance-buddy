<script lang="ts">
	import { onMount } from 'svelte';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import { Card, CardHeader, CardTitle, CardContent, Modal, formatPLN, formatPercent, calculateChange } from '@mskalski/home-ui';
	import { env } from '$env/dynamic/public';
	import { invalidateAll } from '$app/navigation';

	export let data;

	let chartContainer: HTMLDivElement;
	let pieChartContainer: HTMLDivElement;

	// Retirement limits modal
	let showLimitsModal = false;
	let limitsYear = data.currentYear;
	let limits = {
		IKE_Marcin: 0,
		IKE_Ewa: 0,
		IKZE_Marcin: 0,
		IKZE_Ewa: 0
	};

	function openLimitsModal() {
		// Load current limits from retirementStats
		if (data.retirementStats && data.retirementStats.length > 0) {
			data.retirementStats.forEach((stat: any) => {
				const key = `${stat.account_wrapper}_${stat.owner}` as keyof typeof limits;
				if (key in limits) {
					limits[key] = stat.limit_amount || 0;
				}
			});
		}
		showLimitsModal = true;
	}

	async function saveLimits() {
		const apiUrl = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';
		try {
			const responses = await Promise.all([
				fetch(`${apiUrl}/api/retirement/limits/${limitsYear}/IKE/Marcin`, {
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						year: limitsYear,
						account_wrapper: 'IKE',
						owner: 'Marcin',
						limit_amount: limits.IKE_Marcin,
						notes: ''
					})
				}),
				fetch(`${apiUrl}/api/retirement/limits/${limitsYear}/IKE/Ewa`, {
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						year: limitsYear,
						account_wrapper: 'IKE',
						owner: 'Ewa',
						limit_amount: limits.IKE_Ewa,
						notes: ''
					})
				}),
				fetch(`${apiUrl}/api/retirement/limits/${limitsYear}/IKZE/Marcin`, {
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						year: limitsYear,
						account_wrapper: 'IKZE',
						owner: 'Marcin',
						limit_amount: limits.IKZE_Marcin,
						notes: ''
					})
				}),
				fetch(`${apiUrl}/api/retirement/limits/${limitsYear}/IKZE/Ewa`, {
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						year: limitsYear,
						account_wrapper: 'IKZE',
						owner: 'Ewa',
						limit_amount: limits.IKZE_Ewa,
						notes: ''
					})
				})
			]);

			const failedResponses = responses.filter((r) => !r.ok);
			if (failedResponses.length > 0) {
				throw new Error(`${failedResponses.length} request(s) failed`);
			}

			showLimitsModal = false;
			await invalidateAll();
		} catch (err) {
			console.error('Failed to save limits:', err);
			alert('Nie udało się zapisać limitów. Spróbuj ponownie później.');
		}
	}

	onMount(() => {
		// Net Worth Line Chart
		const lineChart = echarts.init(chartContainer);

		const lineOption: EChartsOption = {
			title: {
				text: 'Wartość Netto w Czasie',
				left: 'center'
			},
			tooltip: {
				trigger: 'axis',
				formatter: (params: any) => {
					const date = new Date(params[0].value[0]).toLocaleDateString('pl-PL');
					const value = formatPLN(params[0].value[1]);
					return `${date}<br/>Wartość: ${value}`;
				}
			},
			xAxis: {
				type: 'time'
			},
			yAxis: {
				type: 'value',
				axisLabel: {
					formatter: (value: number) => formatPLN(value)
				}
			},
			series: [
				{
					data: data.net_worth_history.map((h: any) => [h.date, h.value]),
					type: 'line',
					smooth: true,
					areaStyle: {
						color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
							{ offset: 0, color: 'rgba(94, 129, 172, 0.5)' }, // Nord10 #5E81AC
							{ offset: 1, color: 'rgba(94, 129, 172, 0.1)' }
						])
					},
					lineStyle: {
						color: '#5E81AC', // Nord10
						width: 2
					}
				}
			],
			grid: {
				left: '80px',
				right: '40px'
			}
		};

		lineChart.setOption(lineOption);

		// Asset Allocation Pie Chart
		const pieChart = echarts.init(pieChartContainer);

		const pieOption: EChartsOption = {
			title: {
				text: 'Alokacja Aktywów',
				left: 'center'
			},
			tooltip: {
				trigger: 'item',
				formatter: '{b}: {c} PLN ({d}%)'
			},
			color: [
				'#5E81AC', // Nord10 - Frost blue
				'#88C0D0', // Nord8 - Frost cyan
				'#81A1C1', // Nord9 - Frost light blue
				'#8FBCBB', // Nord7 - Frost teal
				'#A3BE8C', // Nord14 - Aurora green
				'#EBCB8B', // Nord13 - Aurora yellow
				'#D08770', // Nord12 - Aurora orange
				'#B48EAD' // Nord15 - Aurora purple
			],
			series: [
				{
					type: 'pie',
					radius: ['40%', '70%'],
					data: data.allocation.map((a: any) => ({
						name: `${a.category}${a.owner ? ` (${a.owner})` : ''}`,
						value: a.value
					})),
					emphasis: {
						itemStyle: {
							shadowBlur: 10,
							shadowOffsetX: 0,
							shadowColor: 'rgba(0, 0, 0, 0.5)'
						}
					}
				}
			]
		};

		pieChart.setOption(pieOption);

		// Responsive resize
		const handleResize = () => {
			lineChart.resize();
			pieChart.resize();
		};

		window.addEventListener('resize', handleResize);

		// Cleanup
		return () => {
			window.removeEventListener('resize', handleResize);
		};
	});

	const change = calculateChange(
		data.current_net_worth,
		data.current_net_worth - data.change_vs_last_month
	);
</script>

<svelte:head>
	<title>Dashboard | Finansowa Forteca</title>
</svelte:head>

<div class="dashboard">
	<div class="page-header">
		<h1>Dashboard</h1>
		<p class="subtitle">Twoja sytuacja finansowa w jednym miejscu</p>
	</div>

	<!-- KPI Cards -->
	<div class="kpi-grid">
		<Card>
			<CardHeader>
				<CardTitle>Wartość Netto</CardTitle>
			</CardHeader>
			<CardContent>
				<div class="kpi-value">{formatPLN(data.current_net_worth)}</div>
				<p
					class="kpi-change"
					class:positive={data.change_vs_last_month >= 0}
					class:negative={data.change_vs_last_month < 0}
				>
					{data.change_vs_last_month >= 0 ? '↑' : '↓'}
					{formatPLN(Math.abs(data.change_vs_last_month))}
					({formatPercent(Math.abs(change.percent))})
					<span class="muted">vs poprzedni miesiąc</span>
				</p>
			</CardContent>
		</Card>

		<Card>
			<CardHeader>
				<CardTitle>Aktywa</CardTitle>
			</CardHeader>
			<CardContent>
				<div class="kpi-value positive">{formatPLN(data.total_assets)}</div>
				<p class="kpi-subtitle">Suma wszystkich aktywów</p>
			</CardContent>
		</Card>

		<Card>
			<CardHeader>
				<CardTitle>Zobowiązania</CardTitle>
			</CardHeader>
			<CardContent>
				<div class="kpi-value negative">{formatPLN(data.total_liabilities)}</div>
				<p class="kpi-subtitle">Suma wszystkich zobowiązań</p>
			</CardContent>
		</Card>
	</div>

	<!-- Retirement Limits Widget -->
	{#if data.retirementStats && data.retirementStats.length > 0}
		<Card>
			<CardHeader>
				<div class="retirement-header">
					<CardTitle>💰 Limity Emerytalne {data.currentYear}</CardTitle>
					<button class="settings-btn" on:click={openLimitsModal}>⚙️</button>
				</div>
			</CardHeader>
			<CardContent>
				<div class="retirement-grid">
					{#each data.retirementStats as stat}
						<div class="retirement-stat">
							<div class="stat-header">
								<h4>{stat.account_wrapper} ({stat.owner})</h4>
								<span class="stat-amount">
									{formatPLN(stat.total_contributed)} / {formatPLN(stat.limit_amount)}
								</span>
							</div>

							<div
								class="progress-bar"
								class:danger={stat.percentage_used < 50}
								class:warning={stat.percentage_used >= 50 && stat.percentage_used < 100}
								class:success={stat.percentage_used >= 100}
							>
								<div
									class="progress-fill"
									style="width: {Math.min(stat.percentage_used, 100)}%"
								></div>
							</div>

							<div class="stat-details">
								<span class="remaining">Pozostało: {formatPLN(stat.remaining)}</span>
								<span
									class="percentage"
									class:danger={stat.percentage_used < 50}
									class:warning={stat.percentage_used >= 50 && stat.percentage_used < 100}
									class:success={stat.percentage_used >= 100}
								>
									{stat.percentage_used}%
								</span>
							</div>

							{#if stat.percentage_used >= 100}
								<div class="success-banner">✅ Limit osiągnięty</div>
							{:else if stat.percentage_used >= 50}
								<div class="warning-banner">⚠️ Zbliżasz się do limitu</div>
							{/if}
						</div>
					{/each}
				</div>
			</CardContent>
		</Card>
	{/if}

	<!-- Charts -->
	<div class="charts-grid">
		<Card>
			<CardContent>
				<div bind:this={chartContainer} class="chart-container"></div>
			</CardContent>
		</Card>

		<Card>
			<CardContent>
				<div bind:this={pieChartContainer} class="chart-container"></div>
			</CardContent>
		</Card>
	</div>
</div>

<!-- Limits Configuration Modal -->
<Modal
	open={showLimitsModal}
	title="Konfiguracja Limitów Emerytalnych"
	onCancel={() => (showLimitsModal = false)}
	showActions={false}
>
	<form on:submit|preventDefault={saveLimits}>
		<div class="limits-form">
			<label>
				Rok
				<select bind:value={limitsYear}>
					{#each [data.currentYear, data.currentYear + 1, data.currentYear + 2] as year}
						<option value={year}>{year}</option>
					{/each}
				</select>
			</label>

			<div class="limit-inputs-grid">
				<label>
					IKE Marcin (PLN)
					<input type="number" bind:value={limits.IKE_Marcin} step="0.01" required />
				</label>

				<label>
					IKE Ewa (PLN)
					<input type="number" bind:value={limits.IKE_Ewa} step="0.01" required />
				</label>

				<label>
					IKZE Marcin (PLN)
					<input type="number" bind:value={limits.IKZE_Marcin} step="0.01" required />
				</label>

				<label>
					IKZE Ewa (PLN)
					<input type="number" bind:value={limits.IKZE_Ewa} step="0.01" required />
				</label>
			</div>
		</div>

		<div class="form-actions">
			<button type="button" class="btn btn-secondary" on:click={() => (showLimitsModal = false)}>
				Anuluj
			</button>
			<button type="submit" class="btn btn-primary">Zapisz</button>
		</div>
	</form>
</Modal>

<style>
	.dashboard {
		display: flex;
		flex-direction: column;
		gap: var(--size-8);
	}

	.page-header h1 {
		font-size: var(--font-size-6);
		font-weight: var(--font-weight-8);
		margin: 0 0 var(--size-2) 0;
	}

	.subtitle {
		color: var(--color-text-muted);
		font-size: var(--font-size-2);
	}

	.kpi-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
		gap: var(--size-6);
	}

	.kpi-value {
		font-size: var(--font-size-6);
		font-weight: var(--font-weight-7);
		margin-bottom: var(--size-2);
	}

	.kpi-value.positive {
		color: var(--color-success);
	}

	.kpi-value.negative {
		color: var(--color-error);
	}

	.kpi-change {
		font-size: var(--font-size-1);
		margin: var(--size-2) 0 0 0;
	}

	.kpi-change.positive {
		color: var(--color-success);
	}

	.kpi-change.negative {
		color: var(--color-error);
	}

	.kpi-subtitle {
		font-size: var(--font-size-1);
		color: var(--color-text-muted);
		margin: var(--size-2) 0 0 0;
	}

	.muted {
		color: var(--color-text-muted);
	}

	.charts-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(500px, 1fr));
		gap: var(--size-6);
	}

	.chart-container {
		width: 100%;
		height: 400px;
	}

	@media (max-width: 768px) {
		.charts-grid {
			grid-template-columns: 1fr;
		}
	}

	/* Retirement Widget */
	.retirement-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		width: 100%;
	}

	.settings-btn {
		background: none;
		border: none;
		font-size: var(--font-size-3);
		cursor: pointer;
		padding: var(--size-1);
		transition: transform 0.2s;
	}

	.settings-btn:hover {
		transform: scale(1.1);
	}

	.retirement-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
		gap: var(--size-6);
	}

	.retirement-stat {
		display: flex;
		flex-direction: column;
		gap: var(--size-3);
	}

	.stat-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.stat-header h4 {
		font-size: var(--font-size-3);
		font-weight: var(--font-weight-7);
		margin: 0;
	}

	.stat-amount {
		font-size: var(--font-size-1);
		color: var(--color-text-muted);
	}

	.progress-bar {
		height: 12px;
		background: var(--color-bg-muted);
		border-radius: var(--radius-2);
		overflow: hidden;
	}

	.progress-bar.danger {
		background: rgba(191, 97, 106, 0.2);
	}

	.progress-bar.warning {
		background: rgba(235, 203, 139, 0.2);
	}

	.progress-bar.success {
		background: rgba(163, 190, 140, 0.2);
	}

	.progress-fill {
		height: 100%;
		background: var(--color-success);
		transition: width 0.3s;
	}

	.progress-bar.danger .progress-fill {
		background: #bf616a;
	}

	.progress-bar.warning .progress-fill {
		background: #ebcb8b;
	}

	.progress-bar.success .progress-fill {
		background: var(--color-success);
	}

	.stat-details {
		display: flex;
		justify-content: space-between;
		font-size: var(--font-size-1);
	}

	.remaining {
		color: var(--color-text-muted);
	}

	.percentage {
		font-weight: var(--font-weight-6);
		color: var(--color-success);
	}

	.percentage.danger {
		color: #bf616a;
	}

	.percentage.warning {
		color: #ebcb8b;
	}

	.percentage.success {
		color: var(--color-success);
	}

	.warning-banner {
		background: rgba(208, 135, 112, 0.2);
		color: var(--color-error);
		padding: var(--size-2);
		border-radius: var(--radius-2);
		text-align: center;
		font-size: var(--font-size-0);
		font-weight: var(--font-weight-6);
	}

	.success-banner {
		background: rgba(163, 190, 140, 0.2);
		color: var(--color-success);
		padding: var(--size-2);
		border-radius: var(--radius-2);
		text-align: center;
		font-size: var(--font-size-0);
		font-weight: var(--font-weight-6);
	}

	.limits-form {
		display: flex;
		flex-direction: column;
		gap: var(--size-4);
	}

	.limit-inputs-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: var(--size-4);
	}

	.limits-form label {
		display: flex;
		flex-direction: column;
		gap: var(--size-2);
		font-size: var(--font-size-1);
		font-weight: var(--font-weight-6);
	}

	.limits-form input,
	.limits-form select {
		padding: var(--size-2);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-2);
		font-size: var(--font-size-1);
		background: var(--color-bg);
		color: var(--color-text);
	}

	.form-actions {
		display: flex;
		justify-content: flex-end;
		gap: var(--size-3);
		margin-top: var(--size-6);
		padding-top: var(--size-4);
		border-top: 1px solid var(--color-border);
	}

	.btn {
		padding: var(--size-2) var(--size-4);
		border-radius: var(--radius-2);
		font-size: var(--font-size-1);
		font-weight: var(--font-weight-6);
		cursor: pointer;
		transition: all 0.2s;
	}

	.btn-primary {
		background: var(--color-primary);
		color: var(--nord6);
		border: none;
	}

	.btn-primary:hover {
		background: var(--nord9);
	}

	.btn-secondary {
		background: transparent;
		color: var(--color-text);
		border: 1px solid var(--color-border);
	}

	.btn-secondary:hover {
		background: var(--color-accent);
	}
</style>
