import { describe, it, expect } from 'vitest';
import { buildCumulativeInflationChartOption } from './inflation';
import type { CpiSeries } from '$lib/types/cpi';

const series: CpiSeries = {
	points: [
		{ year: 2024, yoy_rate: 3.05, cumulative_index: 114.34 },
		{ year: 2022, yoy_rate: 14.4, cumulative_index: 100 },
		{ year: 2023, yoy_rate: 11.06, cumulative_index: 111.06 }
	],
	base_year: 2022,
	latest_year: 2024,
	source: 'GUS'
};

describe('buildCumulativeInflationChartOption', () => {
	it('sorts points by year on the x-axis', () => {
		const option = buildCumulativeInflationChartOption(series);
		const xAxis = option.xAxis as { data: string[] };
		expect(xAxis.data).toEqual(['2022', '2023', '2024']);
	});

	it('plots cumulative index as a line and YoY as bars on a second axis', () => {
		const option = buildCumulativeInflationChartOption(series);
		const seriesOpts = option.series as Array<{
			name: string;
			type: string;
			yAxisIndex: number;
			data: number[];
		}>;

		const line = seriesOpts.find((s) => s.type === 'line');
		const bar = seriesOpts.find((s) => s.type === 'bar');

		expect(line?.yAxisIndex).toBe(0);
		expect(line?.data).toEqual([100, 111.1, 114.3]);
		expect(bar?.yAxisIndex).toBe(1);
		expect(bar?.data).toEqual([14.4, 11.1, 3]);
	});

	it('declares two y-axes (index + percent)', () => {
		const option = buildCumulativeInflationChartOption(series);
		expect(Array.isArray(option.yAxis)).toBe(true);
		expect((option.yAxis as unknown[]).length).toBe(2);
	});
});
