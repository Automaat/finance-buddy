import type { EChartsOption } from 'echarts';
import type { CallbackDataParams, TopLevelFormatterParams } from 'echarts/types/dist/shared';
import type { CpiSeries } from '$lib/types/cpi';
import { chartInk, chartContribution, chartValue } from '$lib/utils/theme';
import { baseChartOption, chartTooltip, axisLine, splitLine } from './chartBase';

type AxisTooltipItem = CallbackDataParams & { axisValue: string };

// buildCumulativeInflationChartOption renders the CPI series as a cumulative
// price-level line (fixed-base index, left axis) plus year-over-year inflation
// bars (right axis). Shares the metryki base option + theme tokens so the
// inflation section blends in.
export function buildCumulativeInflationChartOption(series: CpiSeries): EChartsOption {
	const sorted = [...series.points].sort((a, b) => a.year - b.year);
	const years = sorted.map((p) => String(p.year));
	const cumulative = sorted.map((p) => parseFloat(p.cumulative_index.toFixed(1)));
	const yoy = sorted.map((p) => parseFloat(p.yoy_rate.toFixed(1)));

	return {
		...baseChartOption('Skumulowana inflacja (CPI)'),
		tooltip: {
			...chartTooltip(),
			axisPointer: { type: 'shadow' },
			formatter: function (params: TopLevelFormatterParams) {
				const items = (Array.isArray(params) ? params : [params]) as AxisTooltipItem[];
				const year = items[0]?.axisValue ?? '';
				return (
					`${year}<br/>` +
					items
						.map((p) => {
							const suffix = p.seriesName === 'Inflacja r/r' ? '%' : '';
							return `${p.marker ?? ''} ${p.seriesName}: <b>${p.value ?? '—'}${suffix}</b>`;
						})
						.join('<br/>')
				);
			}
		},
		legend: {
			data: ['Indeks cen (baza 100)', 'Inflacja r/r'],
			bottom: 10,
			textStyle: { color: chartInk, fontSize: 14 }
		},
		grid: { left: 70, right: 60, bottom: 70, top: 70, containLabel: false },
		xAxis: {
			type: 'category',
			data: years,
			axisLabel: { color: chartInk, fontSize: 12 },
			axisLine: axisLine(),
			boundaryGap: true
		},
		yAxis: [
			{
				type: 'value',
				name: 'Indeks',
				nameTextStyle: { color: chartInk, fontSize: 12, fontWeight: 'bold' },
				axisLabel: { color: chartInk, fontSize: 12 },
				axisLine: axisLine(),
				splitLine: splitLine()
			},
			{
				type: 'value',
				name: 'r/r %',
				nameTextStyle: { color: chartInk, fontSize: 12, fontWeight: 'bold' },
				axisLabel: { color: chartInk, fontSize: 12, formatter: '{value}%' },
				axisLine: { show: false },
				splitLine: { show: false }
			}
		],
		series: [
			{
				name: 'Inflacja r/r',
				type: 'bar',
				yAxisIndex: 1,
				data: yoy,
				barWidth: '45%',
				itemStyle: { color: chartContribution, borderRadius: [4, 4, 0, 0] }
			},
			{
				name: 'Indeks cen (baza 100)',
				type: 'line',
				yAxisIndex: 0,
				data: cumulative,
				smooth: true,
				symbol: 'circle',
				symbolSize: 6,
				lineStyle: { width: 3, color: chartValue },
				itemStyle: { color: chartValue }
			}
		]
	};
}
