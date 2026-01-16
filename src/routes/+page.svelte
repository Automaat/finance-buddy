<script lang="ts">
	import { onMount } from 'svelte';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import Card from '$lib/components/Card.svelte';
	import CardHeader from '$lib/components/CardHeader.svelte';
	import CardTitle from '$lib/components/CardTitle.svelte';
	import CardContent from '$lib/components/CardContent.svelte';
	import { formatPLN, formatPercent } from '$lib/utils/format';
	import { calculateChange } from '$lib/utils/calculations';

	export let data;

	let chartContainer: HTMLDivElement;
	let pieChartContainer: HTMLDivElement;

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
</style>
