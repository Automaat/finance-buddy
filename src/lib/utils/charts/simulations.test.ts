import { describe, it, expect } from 'vitest';
import { buildRetirementProjectionOption, type AccountSimulation } from './simulations';

function makeSimulation(name: string, balances: number[]): AccountSimulation {
	return {
		account_name: name,
		starting_balance: balances[0] ?? 0,
		total_contributions: 0,
		total_returns: 0,
		total_tax_savings: 0,
		final_balance: balances[balances.length - 1] ?? 0,
		yearly_projections: balances.map((balance, idx) => ({
			year: 2025 + idx,
			age: 30 + idx,
			annual_contribution: 0,
			balance_end_of_year: balance,
			cumulative_contributions: 0,
			cumulative_returns: 0,
			annual_limit: 0,
			limit_utilized_pct: 0,
			tax_savings: 0
		}))
	};
}

describe('buildRetirementProjectionOption', () => {
	it('builds one line series per simulation', () => {
		const simulations = [
			makeSimulation('IKE Marcin', [1000, 2000, 3000]),
			makeSimulation('IKZE Ewa', [500, 1500, 2500])
		];
		const option = buildRetirementProjectionOption(simulations);
		const series = option.series as Array<{ name: string; type: string; data: number[] }>;

		expect(series).toHaveLength(2);
		expect(series[0].name).toBe('IKE Marcin');
		expect(series[0].type).toBe('line');
		expect(series[0].data).toEqual([1000, 2000, 3000]);
		expect(series[1].data).toEqual([500, 1500, 2500]);
	});

	it('derives x axis years from the first simulation', () => {
		const option = buildRetirementProjectionOption([makeSimulation('IKE Marcin', [1000, 2000])]);
		const xAxis = option.xAxis as { data: number[]; name: string };
		expect(xAxis.data).toEqual([2025, 2026]);
		expect(xAxis.name).toBe('Rok');
	});

	it('sets the legend from account names', () => {
		const option = buildRetirementProjectionOption([
			makeSimulation('IKE Marcin', [1000]),
			makeSimulation('PPK Ewa', [2000])
		]);
		const legend = option.legend as { data: string[] };
		expect(legend.data).toEqual(['IKE Marcin', 'PPK Ewa']);
	});

	it('sets the chart title', () => {
		const option = buildRetirementProjectionOption([]);
		const title = option.title as { text: string };
		expect(title.text).toBe('Projekcja wartości kont emerytalnych');
	});

	it('handles an empty simulation list', () => {
		const option = buildRetirementProjectionOption([]);
		const series = option.series as unknown[];
		const xAxis = option.xAxis as { data: number[] };
		expect(series).toEqual([]);
		expect(xAxis.data).toEqual([]);
	});

	it('formats y-axis labels as thousands', () => {
		const option = buildRetirementProjectionOption([makeSimulation('X', [1000])]);
		const yAxis = option.yAxis as { axisLabel: { formatter: (n: number) => string } };
		expect(yAxis.axisLabel.formatter(15000)).toBe('15k');
		expect(yAxis.axisLabel.formatter(1000)).toBe('1k');
		expect(yAxis.axisLabel.formatter(0)).toBe('0k');
	});

	it('tooltip formatter renders rows per series', () => {
		const option = buildRetirementProjectionOption([makeSimulation('IKE', [1000])]);
		const tooltip = option.tooltip as unknown as {
			formatter: (params: Array<{ name: number; seriesName: string; value: number }>) => string;
		};
		const html = tooltip.formatter([
			{ name: 2025, seriesName: 'IKE', value: 12000 },
			{ name: 2025, seriesName: 'IKZE', value: 7500 }
		]);
		expect(html).toContain('Rok 2025');
		expect(html).toContain('IKE');
		expect(html).toContain('IKZE');
		// Polish locale's thousand separator is environment-dependent; assert
		// the rendered numbers via toLocaleString to stay portable.
		expect(html).toContain(`${(12000).toLocaleString('pl-PL')} PLN`);
		expect(html).toContain(`${(7500).toLocaleString('pl-PL')} PLN`);
	});

	it('tooltip formatter accepts a single (non-array) param', () => {
		const option = buildRetirementProjectionOption([makeSimulation('IKE', [1000])]);
		const tooltip = option.tooltip as unknown as {
			formatter: (param: { name: number; seriesName: string; value: number }) => string;
		};
		const html = tooltip.formatter({ name: 2030, seriesName: 'IKE', value: 50000 });
		expect(html).toContain('Rok 2030');
		expect(html).toContain('IKE');
	});

	it('reuses the palette when there are more series than colors', () => {
		const sims = Array.from({ length: 10 }, (_, i) => makeSimulation(`acc${i}`, [i * 100]));
		const option = buildRetirementProjectionOption(sims);
		const series = option.series as Array<{ itemStyle: { color: string } }>;
		expect(series).toHaveLength(10);
		// 7th item should wrap back to the first palette colour
		expect(series[6].itemStyle.color).toBe(series[0].itemStyle.color);
	});
});
