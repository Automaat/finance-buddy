import type { EChartsOption, LineSeriesOption } from 'echarts';
import type { TopLevelFormatterParams } from 'echarts/types/dist/shared';
import { chartAccent, chartPalette } from '$lib/utils/theme';

export interface MonteCarloBand {
	age: number;
	p5: number;
	p50: number;
	p95: number;
	p5_real: number;
	p50_real: number;
	p95_real: number;
	p5_net: number;
	p50_net: number;
	p95_net: number;
	p5_net_real: number;
	p50_net_real: number;
	p95_net_real: number;
	spending: number;
	spending_real: number;
}

export interface MonteCarloAllocationOut {
	stocks_pct: number;
	bonds_pct: number;
	cash_pct: number;
}

export interface MonteCarloAccountMixOut {
	taxable_pct: number;
	ike_pct: number;
	ikze_pct: number;
	zus_pct: number;
	taxable_gain_pct: number;
}

export interface MonteCarloAssumptions {
	expected_return: number;
	volatility: number;
	source: 'manual' | 'allocation';
	allocation?: MonteCarloAllocationOut;
	inflation_mean: number;
	inflation_volatility: number;
	account_mix?: MonteCarloAccountMixOut;
}

export interface MonteCarloTaxSummary {
	gross_withdrawals_total: number;
	tax_total: number;
	effective_lifetime_rate: number;
}

export interface MonteCarloResult {
	success_rate: number;
	bands: MonteCarloBand[];
	paths: number;
	assumptions: MonteCarloAssumptions;
	tax?: MonteCarloTaxSummary;
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

type BandKey =
	| 'p5'
	| 'p50'
	| 'p95'
	| 'p5_real'
	| 'p50_real'
	| 'p95_real'
	| 'p5_net'
	| 'p50_net'
	| 'p95_net'
	| 'p5_net_real'
	| 'p50_net_real'
	| 'p95_net_real';

function pickKeys(real: boolean, net: boolean): { p5: BandKey; p50: BandKey; p95: BandKey } {
	if (net && real) return { p5: 'p5_net_real', p50: 'p50_net_real', p95: 'p95_net_real' };
	if (net) return { p5: 'p5_net', p50: 'p50_net', p95: 'p95_net' };
	if (real) return { p5: 'p5_real', p50: 'p50_real', p95: 'p95_real' };
	return { p5: 'p5', p50: 'p50', p95: 'p95' };
}

// Fan chart: shaded P5–P95 band with the P50 median overlaid. Implemented
// as a stack — base = P5, ribbon = P95 - P5 (transparent fill on top).
// Pass {real: true} to plot real (inflation-adjusted) bands; pass {net:
// true} to plot net-of-tax bands (uses the AccountMix-driven net series);
// they compose.
export function buildMonteCarloFanOption(
	result: MonteCarloResult,
	opts: { real?: boolean; net?: boolean } = {}
): EChartsOption {
	const real = opts.real === true;
	const net = opts.net === true;
	const keys = pickKeys(real, net);
	const ages = result.bands.map((b) => b.age);
	const p5 = result.bands.map((b) => b[keys.p5]);
	const p50 = result.bands.map((b) => b[keys.p50]);
	const p95 = result.bands.map((b) => b[keys.p95]);
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
			areaStyle: { color: chartPalette[3], opacity: 0.35 },
			itemStyle: { color: chartPalette[3] }
		},
		{
			name: 'Mediana',
			type: 'line',
			data: p50,
			smooth: true,
			showSymbol: false,
			lineStyle: { color: chartAccent, width: 2 },
			itemStyle: { color: chartAccent }
		}
	];

	const suffixParts: string[] = [];
	if (real) suffixParts.push('realne');
	if (net) suffixParts.push('po podatku');
	const suffix = suffixParts.length > 0 ? ` (${suffixParts.join(', ')})` : '';
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
				const v5 = band[keys.p5];
				const v50 = band[keys.p50];
				const v95 = band[keys.p95];
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
