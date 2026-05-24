import { describe, it, expect } from 'vitest';
import {
	aggregateByWrapperOverAges,
	buildRetirementByWrapperOption,
	buildRetirementProjectionOption,
	getTotalBalanceAtAge,
	wrapperFromAccountName,
	type AccountSimulation
} from './simulations';

function makeSimulation(name: string, balances: number[], startAge = 30): AccountSimulation {
	return {
		account_name: name,
		starting_balance: balances[0] ?? 0,
		total_contributions: 0,
		total_returns: 0,
		total_tax_savings: 0,
		final_balance: balances[balances.length - 1] ?? 0,
		yearly_projections: balances.map((balance, idx) => ({
			year: 2025 + idx,
			age: startAge + idx,
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

describe('wrapperFromAccountName', () => {
	it('maps IKE/IKZE/PPK prefixes correctly', () => {
		expect(wrapperFromAccountName('IKE (Marcin)')).toBe('IKE');
		expect(wrapperFromAccountName('IKZE (Marcin)')).toBe('IKZE');
		expect(wrapperFromAccountName('PPK (Marcin)')).toBe('PPK');
	});

	it('returns null for non-retirement accounts', () => {
		expect(wrapperFromAccountName('Rachunek maklerski (Marcin)')).toBeNull();
		expect(wrapperFromAccountName('')).toBeNull();
	});

	it('disambiguates IKZE before IKE so the longer prefix wins', () => {
		expect(wrapperFromAccountName('IKZE (Ewa)')).toBe('IKZE');
	});
});

describe('aggregateByWrapperOverAges', () => {
	it('sums balances per wrapper at each age across owners', () => {
		const sims: AccountSimulation[] = [
			makeSimulation('IKE (Marcin)', [100, 200, 300], 60),
			makeSimulation('IKE (Ewa)', [50, 100, 150], 60),
			makeSimulation('IKZE (Marcin)', [1000, 2000, 3000], 60),
			makeSimulation('PPK (Marcin)', [10, 20, 30], 60)
		];
		const out = aggregateByWrapperOverAges(sims);
		expect(out.ages).toEqual([60, 61, 62]);
		expect(out.IKE).toEqual([150, 300, 450]);
		expect(out.IKZE).toEqual([1000, 2000, 3000]);
		expect(out.PPK).toEqual([10, 20, 30]);
	});

	it('ignores non-retirement accounts', () => {
		const sims: AccountSimulation[] = [
			makeSimulation('Rachunek maklerski (Marcin)', [9999], 60),
			makeSimulation('IKE (Marcin)', [100], 60)
		];
		const out = aggregateByWrapperOverAges(sims);
		expect(out.IKE).toEqual([100]);
		expect(out.IKZE).toEqual([0]);
		expect(out.PPK).toEqual([0]);
	});

	it('returns empty arrays for an empty input', () => {
		const out = aggregateByWrapperOverAges([]);
		expect(out.ages).toEqual([]);
		expect(out.IKE).toEqual([]);
		expect(out.IKZE).toEqual([]);
		expect(out.PPK).toEqual([]);
	});

	it('uses zero when a wrapper has no entry at an age', () => {
		// IKE present at 60, PPK only at 61 — both ages should show.
		const sims: AccountSimulation[] = [
			{
				account_name: 'IKE (Marcin)',
				starting_balance: 0,
				total_contributions: 0,
				total_returns: 0,
				total_tax_savings: 0,
				final_balance: 0,
				yearly_projections: [
					{
						year: 2025,
						age: 60,
						annual_contribution: 0,
						balance_end_of_year: 100,
						cumulative_contributions: 0,
						cumulative_returns: 0,
						annual_limit: 0,
						limit_utilized_pct: 0,
						tax_savings: 0
					}
				]
			},
			{
				account_name: 'PPK (Marcin)',
				starting_balance: 0,
				total_contributions: 0,
				total_returns: 0,
				total_tax_savings: 0,
				final_balance: 0,
				yearly_projections: [
					{
						year: 2026,
						age: 61,
						annual_contribution: 0,
						balance_end_of_year: 50,
						cumulative_contributions: 0,
						cumulative_returns: 0,
						annual_limit: 0,
						limit_utilized_pct: 0,
						tax_savings: 0
					}
				]
			}
		];
		const out = aggregateByWrapperOverAges(sims);
		expect(out.ages).toEqual([60, 61]);
		expect(out.IKE).toEqual([100, 0]);
		expect(out.PPK).toEqual([0, 50]);
	});
});

describe('getTotalBalanceAtAge', () => {
	it('returns the sum of retirement balances at a given age', () => {
		const sims: AccountSimulation[] = [
			makeSimulation('IKE (Marcin)', [100, 200], 60),
			makeSimulation('IKZE (Marcin)', [500, 700], 60),
			makeSimulation('PPK (Marcin)', [50, 80], 60),
			makeSimulation('Rachunek maklerski (Marcin)', [9999, 9999], 60)
		];
		expect(getTotalBalanceAtAge(sims, 60)).toBe(650);
		expect(getTotalBalanceAtAge(sims, 61)).toBe(980);
	});

	it('returns null when no projection covers the age', () => {
		const sims = [makeSimulation('IKE (Marcin)', [100, 200], 60)];
		expect(getTotalBalanceAtAge(sims, 70)).toBeNull();
	});
});

describe('buildRetirementByWrapperOption', () => {
	it('emits three stacked series for IKE/IKZE/PPK', () => {
		const sims = [
			makeSimulation('IKE (Marcin)', [100, 200], 60),
			makeSimulation('IKZE (Marcin)', [50, 75], 60),
			makeSimulation('PPK (Marcin)', [10, 20], 60)
		];
		const option = buildRetirementByWrapperOption(sims, []);
		const series = option.series as Array<{ name: string; stack: string; data: number[] }>;
		expect(series).toHaveLength(3);
		expect(series.map((s) => s.name)).toEqual(['IKE', 'IKZE', 'PPK']);
		expect(series.every((s) => s.stack === 'total')).toBe(true);
		expect(series[0].data).toEqual([100, 200]);
	});

	it('adds markLines only for milestone ages within the projection range', () => {
		const sims = [makeSimulation('IKE (Marcin)', [100, 200, 300], 60)];
		const option = buildRetirementByWrapperOption(sims, [60, 65, 70]);
		const series = option.series as Array<{ markLine?: { data: Array<{ xAxis: string }> } }>;
		const lines = series[0].markLine?.data ?? [];
		// Only age 60 falls inside [60, 62]; 65 and 70 must be dropped.
		expect(lines.map((l) => l.xAxis)).toEqual(['60']);
	});

	it('uses age as x-axis category', () => {
		const sims = [makeSimulation('IKE (Marcin)', [100, 200], 60)];
		const option = buildRetirementByWrapperOption(sims, []);
		const xAxis = option.xAxis as { data: string[]; name: string };
		expect(xAxis.data).toEqual(['60', '61']);
		expect(xAxis.name).toBe('Wiek');
	});

	it('tooltip formatter sums series values and renders rows per wrapper', () => {
		const sims = [
			makeSimulation('IKE (Marcin)', [100, 200], 60),
			makeSimulation('IKZE (Marcin)', [50, 75], 60),
			makeSimulation('PPK (Marcin)', [10, 20], 60)
		];
		const option = buildRetirementByWrapperOption(sims, []);
		const tooltip = option.tooltip as unknown as {
			formatter: (p: Array<{ name: string; seriesName: string; value: number }>) => string;
		};
		const html = tooltip.formatter([
			{ name: '60', seriesName: 'IKE', value: 100 },
			{ name: '60', seriesName: 'IKZE', value: 50 },
			{ name: '60', seriesName: 'PPK', value: 10 }
		]);
		expect(html).toContain('Wiek 60');
		expect(html).toContain('IKE');
		expect(html).toContain('IKZE');
		expect(html).toContain('PPK');
		expect(html).toContain('Razem');
	});

	it('tooltip formatter accepts a single non-array param', () => {
		const sims = [makeSimulation('IKE (Marcin)', [100], 60)];
		const option = buildRetirementByWrapperOption(sims, []);
		const tooltip = option.tooltip as unknown as {
			formatter: (p: { name: string; seriesName: string; value: number }) => string;
		};
		const html = tooltip.formatter({ name: '60', seriesName: 'IKE', value: 100 });
		expect(html).toContain('Wiek 60');
		expect(html).toContain('IKE');
	});

	it('tooltip formatter coerces null/undefined values to 0', () => {
		const sims = [makeSimulation('IKE (Marcin)', [0], 60)];
		const option = buildRetirementByWrapperOption(sims, []);
		const tooltip = option.tooltip as unknown as {
			formatter: (p: Array<{ name: string; seriesName: string; value: unknown }>) => string;
		};
		const html = tooltip.formatter([{ name: '60', seriesName: 'IKE', value: null }]);
		expect(html).toContain('0 PLN');
	});

	it('yAxis formatter renders k-shorthand for retirement-by-wrapper chart', () => {
		const sims = [makeSimulation('IKE (Marcin)', [100], 60)];
		const option = buildRetirementByWrapperOption(sims, []);
		const yAxis = option.yAxis as { axisLabel: { formatter: (v: number) => string } };
		expect(yAxis.axisLabel.formatter(125000)).toBe('125k');
		expect(yAxis.axisLabel.formatter(0)).toBe('0k');
	});
});
