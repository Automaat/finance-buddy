<script lang="ts">
	import { onMount } from 'svelte';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
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
							{ offset: 0, color: 'rgba(59, 130, 246, 0.5)' },
							{ offset: 1, color: 'rgba(59, 130, 246, 0.1)' }
						])
					},
					lineStyle: {
						color: 'rgb(59, 130, 246)',
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
		window.addEventListener('resize', () => {
			lineChart.resize();
			pieChart.resize();
		});
	});

	const change = calculateChange(
		data.current_net_worth,
		data.current_net_worth - data.change_vs_last_month
	);
</script>

<svelte:head>
	<title>Dashboard | Finansowa Forteca</title>
</svelte:head>

<div class="space-y-8">
	<div>
		<h1 class="text-4xl font-bold mb-2">Dashboard</h1>
		<p class="text-muted-foreground">Twoja sytuacja finansowa w jednym miejscu</p>
	</div>

	<!-- KPI Cards -->
	<div class="grid grid-cols-1 md:grid-cols-3 gap-6">
		<Card>
			<CardHeader>
				<CardTitle class="text-sm font-medium">Wartość Netto</CardTitle>
			</CardHeader>
			<CardContent>
				<div class="text-3xl font-bold">{formatPLN(data.current_net_worth)}</div>
				<p
					class="text-sm mt-2"
					class:text-green-600={data.change_vs_last_month >= 0}
					class:text-red-600={data.change_vs_last_month < 0}
				>
					{data.change_vs_last_month >= 0 ? '↑' : '↓'}
					{formatPLN(Math.abs(data.change_vs_last_month))}
					({formatPercent(Math.abs(change.percent))})
					<span class="text-muted-foreground">vs poprzedni miesiąc</span>
				</p>
			</CardContent>
		</Card>

		<Card>
			<CardHeader>
				<CardTitle class="text-sm font-medium">Aktywa</CardTitle>
			</CardHeader>
			<CardContent>
				<div class="text-3xl font-bold text-green-600">{formatPLN(data.total_assets)}</div>
				<p class="text-sm text-muted-foreground mt-2">Suma wszystkich aktywów</p>
			</CardContent>
		</Card>

		<Card>
			<CardHeader>
				<CardTitle class="text-sm font-medium">Zobowiązania</CardTitle>
			</CardHeader>
			<CardContent>
				<div class="text-3xl font-bold text-red-600">{formatPLN(data.total_liabilities)}</div>
				<p class="text-sm text-muted-foreground mt-2">Suma wszystkich zobowiązań</p>
			</CardContent>
		</Card>
	</div>

	<!-- Charts -->
	<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
		<Card>
			<CardContent class="p-6">
				<div bind:this={chartContainer} class="w-full h-96"></div>
			</CardContent>
		</Card>

		<Card>
			<CardContent class="p-6">
				<div bind:this={pieChartContainer} class="w-full h-96"></div>
			</CardContent>
		</Card>
	</div>
</div>
