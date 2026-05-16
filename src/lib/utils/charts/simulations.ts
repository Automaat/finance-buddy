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
