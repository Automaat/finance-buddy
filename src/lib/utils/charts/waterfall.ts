import type { BarSeriesOption, EChartsOption } from 'echarts';
import type { TopLevelFormatterParams } from 'echarts/types/dist/shared';

export interface NetWorthPoint {
	date: string;
	snapshotId: number;
	value: number;
	assets: number;
	liabilities: number;
}

export interface WaterfallStep {
	date: string;
	snapshotId: number;
	startingNetWorth: number;
	endingNetWorth: number;
	assetDelta: number;
	liabilityDelta: number;
}

const COLOR_ASSET_POS = '#A3BE8C'; // green: assets grew
const COLOR_ASSET_NEG = '#BF616A'; // red: assets fell
const COLOR_LIAB_POS = '#BF616A'; // red: liabilities grew (worse)
const COLOR_LIAB_NEG = '#A3BE8C'; // green: liabilities shrank (better)
const COLOR_LINE = '#5E81AC';

// Tooltips render as HTML; escape any interpolated string in case future
// labels carry user content.
function escapeHtml(value: string): string {
	return value
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;')
		.replace(/'/g, '&#39;');
}

// buildWaterfallSteps derives month-over-month deltas from consecutive
// snapshots. A history of N points yields N-1 steps.
export function buildWaterfallSteps(history: NetWorthPoint[]): WaterfallStep[] {
	const out: WaterfallStep[] = [];
	for (let i = 1; i < history.length; i++) {
		const prev = history[i - 1];
		const curr = history[i];
		out.push({
			date: curr.date,
			snapshotId: curr.snapshotId,
			startingNetWorth: prev.value,
			endingNetWorth: curr.value,
			assetDelta: curr.assets - prev.assets,
			liabilityDelta: curr.liabilities - prev.liabilities
		});
	}
	return out;
}

function fmtPLN(value: number): string {
	const sign = value > 0 ? '+' : value < 0 ? '−' : '';
	const abs = Math.abs(value).toLocaleString('pl-PL', { maximumFractionDigits: 0 });
	return `${sign}${abs} PLN`;
}

function formatMonth(dateISO: string): string {
	const d = new Date(dateISO);
	if (Number.isNaN(d.getTime())) return dateISO;
	return d.toLocaleDateString('pl-PL', { year: '2-digit', month: 'short' });
}

export interface WaterfallChartOptions {
	maxMonths?: number; // crop to the most recent N steps (mobile = 6)
}

// buildWaterfallOption renders a grouped bar chart: per month, an Asset Δ
// bar and a Liability Δ bar, plus an overlaid line of running net worth.
// Color-coded so positive contributions are green, negative are red.
export function buildWaterfallOption(
	steps: WaterfallStep[],
	options: WaterfallChartOptions = {}
): EChartsOption {
	const sliced = options.maxMonths ? steps.slice(-options.maxMonths) : steps;

	const months = sliced.map((s) => formatMonth(s.date));
	const assetSeries: BarSeriesOption = {
		name: 'Δ Aktywa',
		type: 'bar',
		data: sliced.map((s) => ({
			value: s.assetDelta,
			itemStyle: { color: s.assetDelta >= 0 ? COLOR_ASSET_POS : COLOR_ASSET_NEG }
		}))
	};
	// Liability delta is shown as how it impacted net worth: a liability
	// increase is a red drag, a decrease (paying down debt) is green.
	const liabilitySeries: BarSeriesOption = {
		name: 'Δ Zobowiązania',
		type: 'bar',
		data: sliced.map((s) => ({
			value: -s.liabilityDelta,
			itemStyle: { color: s.liabilityDelta <= 0 ? COLOR_LIAB_NEG : COLOR_LIAB_POS }
		}))
	};
	const netWorthLine: BarSeriesOption = {
		name: 'Wartość netto',
		type: 'line',
		yAxisIndex: 1,
		smooth: true,
		showSymbol: true,
		data: sliced.map((s) => s.endingNetWorth),
		lineStyle: { color: COLOR_LINE, width: 2 },
		itemStyle: { color: COLOR_LINE }
	} as unknown as BarSeriesOption;

	return {
		title: { text: 'Wartość netto — wkład miesięczny' },
		tooltip: {
			trigger: 'axis',
			formatter: (params: TopLevelFormatterParams) => {
				const items = Array.isArray(params) ? params : [params];
				const idx = items[0].dataIndex as number;
				const step = sliced[idx];
				if (!step) return '';
				const month = escapeHtml(formatMonth(step.date));
				const netDelta = step.endingNetWorth - step.startingNetWorth;
				return (
					`<strong>${month}</strong><br/>` +
					`Saldo początkowe: ${step.startingNetWorth.toLocaleString('pl-PL', { maximumFractionDigits: 0 })} PLN<br/>` +
					`Δ Aktywa: ${fmtPLN(step.assetDelta)}<br/>` +
					`Δ Zobowiązania: ${fmtPLN(step.liabilityDelta)}<br/>` +
					`Saldo końcowe: ${step.endingNetWorth.toLocaleString('pl-PL', { maximumFractionDigits: 0 })} PLN<br/>` +
					`<strong>Zmiana netto: ${fmtPLN(netDelta)}</strong>`
				);
			}
		},
		legend: { data: ['Δ Aktywa', 'Δ Zobowiązania', 'Wartość netto'], bottom: 0 },
		grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true },
		xAxis: { type: 'category', data: months },
		yAxis: [
			{
				type: 'value',
				name: 'Wkład (PLN)',
				axisLabel: { formatter: (v: number) => `${(v / 1000).toFixed(0)}k` }
			},
			{
				type: 'value',
				name: 'Saldo (PLN)',
				position: 'right',
				axisLabel: { formatter: (v: number) => `${(v / 1000).toFixed(0)}k` }
			}
		],
		series: [assetSeries, liabilitySeries, netWorthLine]
	};
}
