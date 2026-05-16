import { describe, it, expect } from 'vitest';
import {
	buildAllocationChartOption,
	buildWrapperChartOption,
	buildInvestmentTrendChartOption,
	buildWrapperTrendChartOption,
	buildYearlyRoiChartOption,
	computeYearlyRoi,
	type AllocationCategory,
	type AllocationWrapper,
	type TimeSeriesPoint
} from './metryki';

const allocationCategories: AllocationCategory[] = [
	{ category: 'akcje', current_percentage: 42.37, target_percentage: 60 },
	{ category: 'obligacje', current_percentage: 57.63, target_percentage: 40 }
];

const wrappers: AllocationWrapper[] = [
	{ wrapper: 'IKE', value: 12000 },
	{ wrapper: 'IKZE', value: 8000 }
];

const trendSeries: TimeSeriesPoint[] = [
	{ date: '2023-01-31', value: 1000, contributions: 1000 },
	{ date: '2023-12-31', value: 2200, contributions: 2000 }
];

describe('buildAllocationChartOption', () => {
	it('builds two bar series with rounded percentages', () => {
		const option = buildAllocationChartOption(allocationCategories);
		const series = option.series as Array<{ name: string; type: string; data: number[] }>;

		expect(series).toHaveLength(2);
		expect(series[0].name).toBe('Obecna');
		expect(series[0].type).toBe('bar');
		expect(series[0].data).toEqual([42.4, 57.6]);
		expect(series[1].name).toBe('Docelowa');
		expect(series[1].data).toEqual([60, 40]);
	});

	it('puts categories on the x axis', () => {
		const option = buildAllocationChartOption(allocationCategories);
		const xAxis = option.xAxis as { data: string[] };
		expect(xAxis.data).toEqual(['akcje', 'obligacje']);
	});

	it('handles an empty input', () => {
		const option = buildAllocationChartOption([]);
		const series = option.series as Array<{ data: number[] }>;
		expect(series[0].data).toEqual([]);
		expect(series[1].data).toEqual([]);
	});
});

describe('buildWrapperChartOption', () => {
	it('builds a single pie series from wrappers', () => {
		const option = buildWrapperChartOption(wrappers);
		const series = option.series as Array<{
			type: string;
			data: Array<{ name: string; value: number }>;
		}>;

		expect(series).toHaveLength(1);
		expect(series[0].type).toBe('pie');
		expect(series[0].data).toEqual([
			{ name: 'IKE', value: 12000 },
			{ name: 'IKZE', value: 8000 }
		]);
	});

	it('handles an empty input', () => {
		const option = buildWrapperChartOption([]);
		const series = option.series as Array<{ data: unknown[] }>;
		expect(series[0].data).toEqual([]);
	});
});

describe('buildInvestmentTrendChartOption', () => {
	it('builds contribution and value line series', () => {
		const option = buildInvestmentTrendChartOption(trendSeries);
		const series = option.series as Array<{ name: string; type: string; data: number[] }>;

		expect(series).toHaveLength(2);
		expect(series[0].name).toBe('Wpłaty');
		expect(series[0].data).toEqual([1000, 2000]);
		expect(series[1].name).toBe('Wartość portfela');
		expect(series[1].data).toEqual([1000, 2200]);
	});

	it('puts dates on the x axis', () => {
		const option = buildInvestmentTrendChartOption(trendSeries);
		const xAxis = option.xAxis as { data: string[] };
		expect(xAxis.data).toEqual(['2023-01-31', '2023-12-31']);
	});

	it('defaults missing values to zero', () => {
		const option = buildInvestmentTrendChartOption([{ date: '2023-01-31' }]);
		const series = option.series as Array<{ data: number[] }>;
		expect(series[0].data).toEqual([0]);
		expect(series[1].data).toEqual([0]);
	});

	it('handles an empty input', () => {
		const option = buildInvestmentTrendChartOption([]);
		const series = option.series as Array<{ data: number[] }>;
		expect(series[0].data).toEqual([]);
	});
});

describe('buildWrapperTrendChartOption', () => {
	it('uses the provided title', () => {
		const option = buildWrapperTrendChartOption('IKE w czasie', trendSeries);
		const title = option.title as { text: string };
		expect(title.text).toBe('IKE w czasie');
	});

	it('builds contribution and value line series', () => {
		const option = buildWrapperTrendChartOption('IKE w czasie', trendSeries);
		const series = option.series as Array<{ name: string; data: number[] }>;
		expect(series[0].data).toEqual([1000, 2000]);
		expect(series[1].data).toEqual([1000, 2200]);
	});

	it('handles an empty input', () => {
		const option = buildWrapperTrendChartOption('Puste', []);
		const series = option.series as Array<{ data: number[] }>;
		expect(series[0].data).toEqual([]);
	});
});

describe('computeYearlyRoi', () => {
	it('computes ROI per year using modified Dietz', () => {
		const series: TimeSeriesPoint[] = [
			{ date: '2022-01-31', value: 1000, contributions: 1000 },
			{ date: '2022-12-31', value: 1100, contributions: 1000 },
			{ date: '2023-12-31', value: 1300, contributions: 1100 }
		];
		const roi = computeYearlyRoi(series);
		expect(roi.get(2022)).toBe(10);
		expect(roi.get(2023)).toBeCloseTo(8.7, 1);
	});

	it('skips a first year that starts with a zero placeholder', () => {
		const series: TimeSeriesPoint[] = [
			{ date: '2022-01-31', value: 0, contributions: 0 },
			{ date: '2022-12-31', value: 500, contributions: 500 }
		];
		const roi = computeYearlyRoi(series);
		expect(roi.has(2022)).toBe(false);
	});

	it('handles an empty input', () => {
		expect(computeYearlyRoi([]).size).toBe(0);
	});
});

describe('buildYearlyRoiChartOption', () => {
	const stock: TimeSeriesPoint[] = [
		{ date: '2022-01-31', value: 1000, contributions: 1000 },
		{ date: '2022-12-31', value: 1100, contributions: 1000 }
	];
	const bond: TimeSeriesPoint[] = [
		{ date: '2023-01-31', value: 2000, contributions: 2000 },
		{ date: '2023-12-31', value: 2100, contributions: 2000 }
	];

	it('builds three bar series aligned across all years', () => {
		const option = buildYearlyRoiChartOption(stock, bond, []);
		const series = option.series as Array<{ name: string; type: string; data: unknown[] }>;
		const xAxis = option.xAxis as { data: string[] };

		expect(series.map((s) => s.name)).toEqual(['Akcje', 'Obligacje', 'PPK']);
		expect(series.every((s) => s.type === 'bar')).toBe(true);
		expect(xAxis.data).toEqual(['2022', '2023']);
		expect(series[0].data).toHaveLength(2);
	});

	it('handles all-empty inputs', () => {
		const option = buildYearlyRoiChartOption([], [], []);
		const xAxis = option.xAxis as { data: string[] };
		const series = option.series as Array<{ data: unknown[] }>;
		expect(xAxis.data).toEqual([]);
		expect(series[0].data).toEqual([]);
	});
});
