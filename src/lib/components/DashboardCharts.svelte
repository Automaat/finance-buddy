<script lang="ts">
	import { onMount } from 'svelte';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import { formatPLN } from '$lib/utils/format';
	import { isMobile, isTablet } from '$lib/utils/viewport';
	import { chartPalette, chartAccent, chartAccentGradient } from '$lib/utils/theme';

	interface Props {
		netWorthHistory: { date: string; value: number }[];
		allocation: { category: string; owner: string | null; value: number }[];
	}

	let { netWorthHistory, allocation }: Props = $props();

	let chartContainer: HTMLDivElement | undefined;
	let pieChartContainer: HTMLDivElement | undefined;
	let lineChart: echarts.ECharts | undefined;
	let pieChart: echarts.ECharts | undefined;

	const gridConfig = $derived(
		$isMobile
			? { left: '40px', right: '20px' }
			: $isTablet
				? { left: '60px', right: '30px' }
				: { left: '80px', right: '40px' }
	);

	const titleFontSize = $derived($isMobile ? 14 : 16);

	onMount(() => {
		if (!chartContainer || !pieChartContainer) return;

		lineChart = echarts.init(chartContainer);

		const lineOption: EChartsOption = {
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
		};

		lineChart.setOption(lineOption);

		pieChart = echarts.init(pieChartContainer);

		const pieOption: EChartsOption = {
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

		const handleResize = () => {
			lineChart?.resize();
			pieChart?.resize();
		};

		window.addEventListener('resize', handleResize);

		return () => {
			window.removeEventListener('resize', handleResize);
			lineChart?.dispose();
			pieChart?.dispose();
		};
	});

	$effect(() => {
		if (lineChart && gridConfig) {
			lineChart.setOption({
				grid: gridConfig,
				title: { textStyle: { fontSize: titleFontSize } }
			});
		}
	});

	$effect(() => {
		if (pieChart && titleFontSize) {
			pieChart.setOption({
				title: { textStyle: { fontSize: titleFontSize } }
			});
		}
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
