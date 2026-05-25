import { describe, it, expect } from 'vitest';
import { buildMonteCarloFanOption, type MonteCarloResult } from './montecarlo';

function makeResult(bands: Array<[number, number, number, number]>): MonteCarloResult {
	return {
		success_rate: 0.9,
		paths: 1000,
		bands: bands.map(([age, p5, p50, p95]) => ({
			age,
			p5,
			p50,
			p95,
			p5_real: p5,
			p50_real: p50,
			p95_real: p95,
			p5_net: p5,
			p50_net: p50,
			p95_net: p95,
			p5_net_real: p5,
			p50_net_real: p50,
			p95_net_real: p95,
			spending: 0,
			spending_real: 0
		})),
		assumptions: {
			expected_return: 6,
			volatility: 15,
			source: 'manual',
			inflation_mean: 0,
			inflation_volatility: 0
		}
	};
}

describe('buildMonteCarloFanOption', () => {
	it('emits three series: base, ribbon, median', () => {
		const opt = buildMonteCarloFanOption(
			makeResult([
				[60, 100, 200, 300],
				[61, 150, 280, 410]
			])
		);
		const series = opt.series as Array<{ name: string; data: number[] }>;
		expect(series).toHaveLength(3);
		expect(series.map((s) => s.name)).toEqual(['P5', '90% przedział', 'Mediana']);
	});

	it('stacks the ribbon as P95 - P5 above the P5 base', () => {
		const opt = buildMonteCarloFanOption(
			makeResult([
				[60, 100, 200, 300],
				[61, 150, 280, 410]
			])
		);
		const series = opt.series as Array<{ name: string; data: number[] }>;
		expect(series[0].data).toEqual([100, 150]);
		expect(series[1].data).toEqual([200, 260]);
		expect(series[2].data).toEqual([200, 280]);
	});

	it('x-axis is age category', () => {
		const opt = buildMonteCarloFanOption(makeResult([[60, 0, 0, 0]]));
		const xAxis = opt.xAxis as { data: string[]; name: string };
		expect(xAxis.data).toEqual(['60']);
		expect(xAxis.name).toBe('Wiek');
	});

	it('handles empty bands without crashing', () => {
		const opt = buildMonteCarloFanOption(makeResult([]));
		const series = opt.series as Array<{ data: number[] }>;
		expect(series[0].data).toEqual([]);
		expect(series[1].data).toEqual([]);
		expect(series[2].data).toEqual([]);
	});

	it('tooltip formatter renders the three percentiles', () => {
		const opt = buildMonteCarloFanOption(
			makeResult([
				[60, 1000, 2000, 3000],
				[61, 1500, 2800, 4100]
			])
		);
		const tooltip = opt.tooltip as unknown as {
			formatter: (params: Array<{ name: number; value: number; dataIndex: number }>) => string;
		};
		const html = tooltip.formatter([{ name: 61, value: 2800, dataIndex: 1 }]);
		expect(html).toContain('Wiek 61');
		expect(html).toContain('P5:');
		expect(html).toContain('mediana');
		expect(html).toContain('P95:');
		expect(html).toContain(`${(1500).toLocaleString('pl-PL')} PLN`);
	});
});
