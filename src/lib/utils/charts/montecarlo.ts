import type { EChartsOption, LineSeriesOption } from 'echarts';
import type { TopLevelFormatterParams } from 'echarts/types/dist/shared';

export interface MonteCarloBand {
	age: number;
	p5: number;
	p50: number;
	p95: number;
	p5_real: number;
	p50_real: number;
	p95_real: number;
	spending: number;
	spending_real: number;
}

export interface MonteCarloAllocationOut {
	stocks_pct: number;
	bonds_pct: number;
	cash_pct: number;
}

export interface MonteCarloAssumptions {
	expected_return: number;
	volatility: number;
	source: 'manual' | 'allocation';
	allocation?: MonteCarloAllocationOut;
	inflation_mean: number;
	inflation_volatility: number;
}

export interface MonteCarloResult {
	success_rate: number;
	bands: MonteCarloBand[];
	paths: number;
	assumptions: MonteCarloAssumptions;
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
// Pass {real: true} to plot the real (inflation-adjusted) bands instead
// of the nominal ones; tooltip labels follow the choice.
export function buildMonteCarloFanOption(
	result: MonteCarloResult,
	opts: { real?: boolean } = {}
): EChartsOption {
	const real = opts.real === true;
	const ages = result.bands.map((b) => b.age);
	const p5 = result.bands.map((b) => (real ? b.p5_real : b.p5));
	const p50 = result.bands.map((b) => (real ? b.p50_real : b.p50));
	const p95 = result.bands.map((b) => (real ? b.p95_real : b.p95));
	const ribbon = p95.map((v, i) => v - p5[i]);

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

	const suffix = real ? ' (realne)' : '';
	return {
		title: { text: `Symulacja Monte Carlo — projekcja salda${suffix}` },
		tooltip: {
			trigger: 'axis',
			formatter: (params: TopLevelFormatterParams) => {
				const items = Array.isArray(params) ? params : [params];
				const age = items[0].name;
				const idx = items[0].dataIndex;
				const band = result.bands[idx as number];
				if (!band) return '';
				const v5 = real ? band.p5_real : band.p5;
				const v50 = real ? band.p50_real : band.p50;
				const v95 = real ? band.p95_real : band.p95;
				return (
					`<strong>Wiek ${escapeHtml(String(age))}${suffix}</strong><br/>` +
					`P5: ${fmtPLN(v5)} PLN<br/>` +
					`P50 (mediana): ${fmtPLN(v50)} PLN<br/>` +
					`P95: ${fmtPLN(v95)} PLN`
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
