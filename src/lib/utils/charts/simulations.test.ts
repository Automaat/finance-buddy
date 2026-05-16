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
});
