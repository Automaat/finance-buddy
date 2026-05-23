import type { EChartsOption, LineSeriesOption } from 'echarts';
import type { TopLevelFormatterParams } from 'echarts/types/dist/shared';

export interface YearlyProjection {
	year: number;
	age: number;
	annual_contribution: number;
	balance_end_of_year: number;
	cumulative_contributions: number;
	cumulative_returns: number;
	annual_limit: number;
	limit_utilized_pct: number;
	tax_savings: number;
	government_subsidies?: number;
	monthly_salary?: number;
	return_rate?: number;
}

export interface AccountSimulation {
	account_name: string;
	starting_balance: number;
	total_contributions: number;
	total_returns: number;
	total_tax_savings: number;
	total_subsidies?: number;
	final_balance: number;
	yearly_projections: YearlyProjection[];
}

const SERIES_COLORS = ['#5E81AC', '#81A1C1', '#88C0D0', '#8FBCBB', '#B48EAD', '#A3BE8C'];

export function buildRetirementProjectionOption(simulations: AccountSimulation[]): EChartsOption {
	const years = simulations.length > 0 ? simulations[0].yearly_projections.map((p) => p.year) : [];

	const series: LineSeriesOption[] = simulations.map((sim, idx) => ({
		name: sim.account_name,
		type: 'line',
		data: sim.yearly_projections.map((p) => p.balance_end_of_year),
		smooth: true,
		itemStyle: { color: SERIES_COLORS[idx % SERIES_COLORS.length] }
	}));

	return {
		title: { text: 'Projekcja wartości kont emerytalnych' },
		tooltip: {
			trigger: 'axis',
			formatter: (params: TopLevelFormatterParams) => {
				const items = Array.isArray(params) ? params : [params];
				let result = `<strong>Rok ${items[0].name}</strong><br/>`;
				items.forEach((param) => {
					const value = (param.value as number).toLocaleString('pl-PL');
					result += `${param.seriesName}: ${value} PLN<br/>`;
				});
				return result;
			}
		},
		legend: {
			data: simulations.map((s) => s.account_name),
			bottom: 0
		},
		grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true },
		xAxis: { type: 'category', data: years, name: 'Rok' },
		yAxis: {
			type: 'value',
			name: 'Wartość (PLN)',
			axisLabel: {
				formatter: (value: number) => `${(value / 1000).toFixed(0)}k`
			}
		},
		series
	};
}

export type WrapperKey = 'IKE' | 'IKZE' | 'PPK';

const WRAPPER_PREFIX: Record<WrapperKey, string> = {
	IKE: 'IKE ',
	IKZE: 'IKZE ',
	PPK: 'PPK '
};

export function wrapperFromAccountName(name: string): WrapperKey | null {
	if (name.startsWith(WRAPPER_PREFIX.IKZE)) return 'IKZE';
	if (name.startsWith(WRAPPER_PREFIX.IKE)) return 'IKE';
	if (name.startsWith(WRAPPER_PREFIX.PPK)) return 'PPK';
	return null;
}

export interface WrapperAggregates {
	ages: number[];
	IKE: number[];
	IKZE: number[];
	PPK: number[];
}

export function aggregateByWrapperOverAges(simulations: AccountSimulation[]): WrapperAggregates {
	const byAge = new Map<number, { IKE: number; IKZE: number; PPK: number }>();
	for (const sim of simulations) {
		const wrapper = wrapperFromAccountName(sim.account_name);
		if (!wrapper) continue;
		for (const p of sim.yearly_projections) {
			const bucket = byAge.get(p.age) ?? { IKE: 0, IKZE: 0, PPK: 0 };
			bucket[wrapper] += p.balance_end_of_year;
			byAge.set(p.age, bucket);
		}
	}
	const ages = Array.from(byAge.keys()).sort((a, b) => a - b);
	return {
		ages,
		IKE: ages.map((a) => byAge.get(a)?.IKE ?? 0),
		IKZE: ages.map((a) => byAge.get(a)?.IKZE ?? 0),
		PPK: ages.map((a) => byAge.get(a)?.PPK ?? 0)
	};
}

export function getTotalBalanceAtAge(simulations: AccountSimulation[], age: number): number | null {
	let total = 0;
	let found = false;
	for (const sim of simulations) {
		if (!wrapperFromAccountName(sim.account_name)) continue;
		const proj = sim.yearly_projections.find((p) => p.age === age);
		if (proj) {
			total += proj.balance_end_of_year;
			found = true;
		}
	}
	return found ? total : null;
}

const WRAPPER_COLORS: Record<WrapperKey, string> = {
	IKE: '#5E81AC',
	IKZE: '#88C0D0',
	PPK: '#A3BE8C'
};

export function buildRetirementByWrapperOption(
	simulations: AccountSimulation[],
	milestoneAges: number[]
): EChartsOption {
	const data = aggregateByWrapperOverAges(simulations);

	interface MilestoneMarkLine {
		xAxis: string;
		label: { formatter: string; position: 'insideEndTop' };
	}
	const markLineData: MilestoneMarkLine[] = milestoneAges
		.map((age) => {
			const idx = data.ages.indexOf(age);
			if (idx === -1) return null;
			const total = data.IKE[idx] + data.IKZE[idx] + data.PPK[idx];
			return {
				xAxis: String(age),
				label: {
					formatter: `${age} lat\n${(total / 1000).toFixed(0)}k`,
					position: 'insideEndTop' as const
				}
			};
		})
		.filter((d): d is MilestoneMarkLine => d !== null);

	const wrappers: WrapperKey[] = ['IKE', 'IKZE', 'PPK'];
	const series: LineSeriesOption[] = wrappers.map((w) => ({
		name: w,
		type: 'line',
		stack: 'total',
		areaStyle: {},
		smooth: true,
		showSymbol: false,
		data: data[w],
		itemStyle: { color: WRAPPER_COLORS[w] }
	}));

	if (series.length > 0 && markLineData.length > 0) {
		series[0].markLine = {
			symbol: ['none', 'none'],
			lineStyle: { type: 'dashed', color: '#B48EAD' },
			data: markLineData
		};
	}

	return {
		title: { text: 'Projekcja IKE + IKZE + PPK wg wieku' },
		tooltip: {
			trigger: 'axis',
			formatter: (params: TopLevelFormatterParams) => {
				const items = Array.isArray(params) ? params : [params];
				let total = 0;
				let result = `<strong>Wiek ${items[0].name}</strong><br/>`;
				items.forEach((p) => {
					const value = (p.value as number) || 0;
					total += value;
					result += `${p.seriesName}: ${value.toLocaleString('pl-PL', { maximumFractionDigits: 0 })} PLN<br/>`;
				});
				result += `<strong>Razem: ${total.toLocaleString('pl-PL', { maximumFractionDigits: 0 })} PLN</strong>`;
				return result;
			}
		},
		legend: { data: wrappers, bottom: 0 },
		grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true },
		xAxis: { type: 'category', data: data.ages.map(String), name: 'Wiek' },
		yAxis: {
			type: 'value',
			name: 'Wartość (PLN)',
			axisLabel: {
				formatter: (value: number) => `${(value / 1000).toFixed(0)}k`
			}
		},
		series
	};
}
