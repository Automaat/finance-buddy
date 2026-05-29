import type { EChartsOption } from 'echarts';
import type { CallbackDataParams, TopLevelFormatterParams } from 'echarts/types/dist/shared';
import type { CpiSeries } from '$lib/types/cpi';

type AxisTooltipItem = CallbackDataParams & { axisValue: string };

// buildCumulativeInflationChartOption renders the CPI series as a cumulative
// price-level line (fixed-base index, left axis) plus year-over-year inflation
// bars (right axis). Mirrors the Nord palette and styling used across the
// metryki charts so the inflation section blends in.
export function buildCumulativeInflationChartOption(series: CpiSeries): EChartsOption {
	const sorted = [...series.points].sort((a, b) => a.year - b.year);
	const years = sorted.map((p) => String(p.year));
	const cumulative = sorted.map((p) => parseFloat(p.cumulative_index.toFixed(1)));
	const yoy = sorted.map((p) => parseFloat(p.yoy_rate.toFixed(1)));

	return {
		backgroundColor: 'transparent',
		title: {
			text: 'Skumulowana inflacja (CPI)',
			left: 'center',
			top: 10,
			textStyle: { color: '#2e3440', fontSize: 16, fontWeight: 'bold' }
		},
		tooltip: {
			trigger: 'axis',
			axisPointer: { type: 'shadow' },
			backgroundColor: 'rgba(255, 255, 255, 0.95)',
			borderColor: '#d8dee9',
			textStyle: { color: '#2e3440' },
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
			textStyle: { color: '#2e3440', fontSize: 14 }
		},
		grid: { left: 70, right: 60, bottom: 70, top: 70, containLabel: false },
		xAxis: {
			type: 'category',
			data: years,
			axisLabel: { color: '#2e3440', fontSize: 12 },
			axisLine: { lineStyle: { color: '#d8dee9', width: 2 } },
			boundaryGap: true
		},
		yAxis: [
			{
				type: 'value',
				name: 'Indeks',
				nameTextStyle: { color: '#2e3440', fontSize: 12, fontWeight: 'bold' },
				axisLabel: { color: '#2e3440', fontSize: 12 },
				axisLine: { lineStyle: { color: '#d8dee9', width: 2 } },
				splitLine: { lineStyle: { color: '#e5e9f0', type: 'dashed' } }
			},
			{
				type: 'value',
				name: 'r/r %',
				nameTextStyle: { color: '#2e3440', fontSize: 12, fontWeight: 'bold' },
				axisLabel: { color: '#2e3440', fontSize: 12, formatter: '{value}%' },
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
				itemStyle: { color: '#88c0d0', borderRadius: [4, 4, 0, 0] }
			},
			{
				name: 'Indeks cen (baza 100)',
				type: 'line',
				yAxisIndex: 0,
				data: cumulative,
				smooth: true,
				symbol: 'circle',
				symbolSize: 6,
				lineStyle: { width: 3, color: '#5e81ac' },
				itemStyle: { color: '#5e81ac' }
			}
		]
	};
}
