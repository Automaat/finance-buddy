<script lang="ts">
	import { onMount } from 'svelte';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import { formatPLN } from '$lib/utils/format';
	import { isMobile, isTablet } from '$lib/utils/viewport';
	import { chartPalette, chartAccent, chartAccentGradient } from '$lib/utils/theme';
	import { ownerName, type OwnerOption } from '$lib/types/owners';
	import { createChart } from '$lib/utils/charts/lifecycle';

	interface Props {
		netWorthHistory: { date: string; value: number }[];
		allocation: { category: string; owner_user_id: number | null; value: number }[];
		owners: OwnerOption[];
	}

	let { netWorthHistory, allocation, owners }: Props = $props();

	let chartContainer: HTMLDivElement | undefined;
	let pieChartContainer: HTMLDivElement | undefined;
	let lineChart: echarts.ECharts | undefined = $state(undefined);
	let pieChart: echarts.ECharts | undefined = $state(undefined);

	const gridConfig = $derived(
		$isMobile
			? { left: '40px', right: '20px' }
			: $isTablet
				? { left: '60px', right: '30px' }
				: { left: '80px', right: '40px' }
	);

	const titleFontSize = $derived($isMobile ? 14 : 16);

	const lineOption = $derived<EChartsOption>({
		title: {
			text: 'Wartość Netto w Czasie',
			left: 'center',
			textStyle: { fontSize: titleFontSize }
		},
		tooltip: {
			trigger: 'axis',
			formatter: (params: any) => {
				const date = new Date(params[0].value[0]).toLocaleDateString('pl-PL');
				const value = formatPLN(params[0].value[1]);
				return `${date}<br/>Wartość: ${value}`;
			}
		},
		xAxis: { type: 'time' },
		yAxis: {
			type: 'value',
			axisLabel: { formatter: (value: number) => formatPLN(value) }
		},
		series: [
			{
				data: netWorthHistory.map((h) => [h.date, h.value]),
				type: 'line',
				smooth: true,
				areaStyle: {
					color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
						{ offset: 0, color: chartAccentGradient[0] },
						{ offset: 1, color: chartAccentGradient[1] }
					])
				},
				lineStyle: { color: chartAccent, width: 2 }
			}
		],
		grid: gridConfig
	});

	const pieOption = $derived<EChartsOption>({
		title: {
			text: 'Alokacja Aktywów',
			left: 'center',
			textStyle: { fontSize: titleFontSize }
		},
		tooltip: { trigger: 'item', formatter: '{b}: {c} PLN ({d}%)' },
		color: [...chartPalette],
		series: [
			{
				type: 'pie',
				radius: ['40%', '70%'],
				data: allocation.map((a) => ({
					name: `${a.category} (${ownerName(owners, a.owner_user_id)})`,
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
	});

	onMount(() => {
		if (!chartContainer || !pieChartContainer) return;
		const lineHandle = createChart(chartContainer);
		const pieHandle = createChart(pieChartContainer);
		// The $effect blocks below run after mount and will apply the initial
		// options — no need to setOption here.
		lineChart = lineHandle.chart;
		pieChart = pieHandle.chart;

		return () => {
			lineHandle.dispose();
			pieHandle.dispose();
		};
	});

	$effect(() => {
		if (lineChart) lineChart.setOption(lineOption);
	});

	$effect(() => {
		if (pieChart) pieChart.setOption(pieOption);
	});
</script>

<div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
	<div class="card preset-filled-surface-100-900 p-4">
		<div bind:this={chartContainer} class="w-full h-[400px]"></div>
	</div>

	<div class="card preset-filled-surface-100-900 p-4">
		<div bind:this={pieChartContainer} class="w-full h-[400px]"></div>
	</div>
</div>
