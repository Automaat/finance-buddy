<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import * as echarts from 'echarts';
	import type { ECElementEvent, EChartsOption } from 'echarts';
	import { formatPLN, formatNumber, formatDate } from '$lib/utils/format';
	import { isMobile, isTablet } from '$lib/utils/viewport';
	import { chartPalette, chartValue, chartValueGradient } from '$lib/utils/theme';
	import { ownerName, type OwnerOption } from '$lib/types/owners';
	import { createChart } from '$lib/utils/charts/lifecycle';
	import { buildWaterfallOption, buildWaterfallSteps } from '$lib/utils/charts/waterfall';
	import { topNWithOther } from '$lib/utils/allocation';

	interface NetWorthHistoryPoint {
		date: string;
		value: number;
		assets?: number;
		liabilities?: number;
		snapshot_id?: number;
	}

	interface Props {
		netWorthHistory: NetWorthHistoryPoint[];
		allocation: { category: string; owner_user_id: number | null; value: number }[];
		owners: OwnerOption[];
	}

	let { netWorthHistory, allocation, owners }: Props = $props();

	let chartContainer: HTMLDivElement | undefined;
	let pieChartContainer: HTMLDivElement | undefined;
	// Read inside an $effect, so must be $state.
	let waterfallContainer: HTMLDivElement | undefined = $state();
	let lineChart: echarts.ECharts | undefined = $state(undefined);
	let pieChart: echarts.ECharts | undefined = $state(undefined);
	let waterfallChart: echarts.ECharts | undefined = $state(undefined);

	const waterfallSteps = $derived.by(() => {
		const usable = netWorthHistory.filter(
			(h) => typeof h.assets === 'number' && typeof h.liabilities === 'number'
		);
		return buildWaterfallSteps(
			usable.map((h) => ({
				date: h.date,
				snapshotId: h.snapshot_id ?? 0,
				value: h.value,
				assets: h.assets!,
				liabilities: h.liabilities!
			}))
		);
	});

	const waterfallMaxMonths = $derived($isMobile ? 6 : 12);
	const waterfallSliced = $derived(waterfallSteps.slice(-waterfallMaxMonths));

	const gridConfig = $derived(
		$isMobile
			? { left: '40px', right: '20px' }
			: $isTablet
				? { left: '60px', right: '30px' }
				: { left: '80px', right: '40px' }
	);

	const titleFontSize = $derived($isMobile ? 14 : 16);
	const allocationSlices = $derived(topNWithOther(allocation, 6));

	const lineOption = $derived<EChartsOption>({
		title: {
			text: 'Wartość Netto w Czasie',
			left: 'center',
			textStyle: { fontSize: titleFontSize }
		},
		tooltip: {
			trigger: 'axis',
			formatter: (params: any) => {
				const date = formatDate(params[0].value[0]);
				const value = formatPLN(params[0].value[1]);
				return `${date}<br/>Wartość: ${value}`;
			}
		},
		xAxis: {
			type: 'time',
			// hideOverlap drops colliding tick labels instead of smearing them
			// together — critical on a narrow phone axis, especially with sparse
			// data where ECharts otherwise stacks day/month labels on top of
			// each other.
			axisLabel: { hideOverlap: true }
		},
		yAxis: {
			type: 'value',
			axisLabel: { formatter: (value: number) => formatPLN(value) }
		},
		grid: { ...gridConfig, containLabel: true },
		series: [
			{
				data: netWorthHistory.map((h) => [h.date, h.value]),
				type: 'line',
				smooth: true,
				areaStyle: {
					color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
						{ offset: 0, color: chartValueGradient[0] },
						{ offset: 1, color: chartValueGradient[1] }
					])
				},
				lineStyle: { color: chartValue, width: 2 },
				itemStyle: { color: chartValue }
			}
		]
	});

	const pieOption = $derived<EChartsOption>({
		title: {
			text: 'Alokacja Aktywów',
			left: 'center',
			textStyle: { fontSize: titleFontSize }
		},
		tooltip: {
			trigger: 'item',
			formatter: (params: unknown) => {
				const p = params as { name: string; value: number; percent: number };
				return `${p.name}: ${formatPLN(p.value)} (${formatNumber(p.percent, 1)}%)`;
			}
		},
		// On a phone the radial callout labels collide and get clipped at the
		// card edges, so swap them for a scrollable legend along the bottom.
		legend: $isMobile
			? { type: 'scroll', bottom: 0, left: 'center', textStyle: { fontSize: 11 } }
			: undefined,
		color: [...chartPalette],
		series: [
			{
				type: 'pie',
				radius: ['40%', '70%'],
				center: $isMobile ? ['50%', '42%'] : ['50%', '50%'],
				avoidLabelOverlap: true,
				label: { show: !$isMobile },
				labelLine: { show: !$isMobile },
				data: allocationSlices.map((a) => ({
					name:
						a.category === 'Inne'
							? 'Inne'
							: `${a.category} (${ownerName(owners, a.owner_user_id)})`,
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

	// Waterfall chart lifecycle: the container only mounts when at least
	// two snapshots exist (deltas need a previous baseline). Create the
	// chart once via `handle` so the ResizeObserver from createChart() is
	// disconnected on teardown — `chart.dispose()` alone would leak it.
	$effect(() => {
		if (!waterfallContainer) return;
		const handle = createChart(waterfallContainer);
		waterfallChart = handle.chart;
		return () => {
			handle.dispose();
			waterfallChart = undefined;
		};
	});

	$effect(() => {
		if (!waterfallChart) return;
		// Wipe any prior click handler — `setOption` is fine to call
		// repeatedly, but click handlers stack across reactive runs.
		waterfallChart.off('click');
		waterfallChart.on('click', (event: ECElementEvent) => {
			const idx = event.dataIndex as number;
			const step = waterfallSliced[idx];
			if (step && step.snapshotId) {
				goto(`/snapshots/${step.snapshotId}/edit`);
			}
		});
		waterfallChart.setOption(buildWaterfallOption(waterfallSliced, { isMobile: $isMobile }));
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

{#if waterfallSteps.length > 0}
	<div class="card preset-filled-surface-100-900 p-4 mt-4">
		<div bind:this={waterfallContainer} class="w-full h-[360px] cursor-pointer"></div>
	</div>
{/if}
