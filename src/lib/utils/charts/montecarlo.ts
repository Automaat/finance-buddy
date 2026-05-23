import type { EChartsOption, LineSeriesOption } from 'echarts';
import type { TopLevelFormatterParams } from 'echarts/types/dist/shared';

export interface MonteCarloBand {
	age: number;
	p5: number;
	p50: number;
	p95: number;
}

export interface MonteCarloResult {
	success_rate: number;
	bands: MonteCarloBand[];
	paths: number;
}

function fmtPLN(value: number): string {
	return value.toLocaleString('pl-PL', { maximumFractionDigits: 0 });
}

// Tooltip values come straight from the backend (numeric) but echarts'
// default tooltip renders as HTML — escape any string interpolation that
// could pick up user-provided values in the future.
function escapeHtml(value: string): string {
	return value
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;')
		.replace(/'/g, '&#39;');
}

// Fan chart: shaded P5–P95 band with the P50 median overlaid. Implemented
// as a stack — base = P5, ribbon = P95 - P5 (transparent fill on top).
export function buildMonteCarloFanOption(result: MonteCarloResult): EChartsOption {
	const ages = result.bands.map((b) => b.age);
	const p5 = result.bands.map((b) => b.p5);
	const p50 = result.bands.map((b) => b.p50);
	const ribbon = result.bands.map((b) => b.p95 - b.p5);

	const series: LineSeriesOption[] = [
		{
			name: 'P5',
			type: 'line',
			data: p5,
			stack: 'fan',
			lineStyle: { opacity: 0 },
			showSymbol: false,
			areaStyle: { color: 'transparent' },
			tooltip: { show: false }
		},
		{
			name: '90% przedział',
			type: 'line',
			data: ribbon,
			stack: 'fan',
			lineStyle: { opacity: 0 },
			showSymbol: false,
			areaStyle: { color: '#88C0D0', opacity: 0.35 }
		},
		{
			name: 'Mediana',
			type: 'line',
			data: p50,
			smooth: true,
			showSymbol: false,
			lineStyle: { color: '#5E81AC', width: 2 },
			itemStyle: { color: '#5E81AC' }
		}
	];

	return {
		title: { text: 'Symulacja Monte Carlo — projekcja salda' },
		tooltip: {
			trigger: 'axis',
			formatter: (params: TopLevelFormatterParams) => {
				const items = Array.isArray(params) ? params : [params];
				const age = items[0].name;
				const idx = items[0].dataIndex;
				const band = result.bands[idx as number];
				if (!band) return '';
				return (
					`<strong>Wiek ${escapeHtml(String(age))}</strong><br/>` +
					`P5: ${fmtPLN(band.p5)} PLN<br/>` +
					`P50 (mediana): ${fmtPLN(band.p50)} PLN<br/>` +
					`P95: ${fmtPLN(band.p95)} PLN`
				);
			}
		},
		legend: { data: ['Mediana', '90% przedział'], bottom: 0 },
		grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true },
		xAxis: { type: 'category', data: ages.map(String), name: 'Wiek' },
		yAxis: {
			type: 'value',
			name: 'Wartość (PLN)',
			axisLabel: {
				formatter: (v: number) => `${(v / 1000).toFixed(0)}k`
			}
		},
		series
	};
}
