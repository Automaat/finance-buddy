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

	it('label position callback flips below the axis for negative ROI', () => {
		const option = buildYearlyRoiChartOption([], [], []);
		const series = option.series as Array<{
			label: { position: (p: { value: unknown }) => string };
		}>;
		expect(series[0].label.position({ value: 5 })).toBe('top');
		expect(series[0].label.position({ value: -3 })).toBe('bottom');
		expect(series[0].label.position({ value: 0 })).toBe('top');
	});

	it('tooltip formatter shows ROI per series and "brak danych" for nulls', () => {
		const option = buildYearlyRoiChartOption([], [], []);
		const tooltip = option.tooltip as unknown as {
			formatter: (p: Array<{ seriesName: string; value: number | null }>) => string;
		};
		const html = tooltip.formatter([
			{ seriesName: 'Akcje', value: 12 },
			{ seriesName: 'Obligacje', value: null },
			{ seriesName: 'PPK', value: 4 }
		]);
		expect(html).toContain('Akcje: 12%');
		expect(html).toContain('Obligacje: brak danych');
		expect(html).toContain('PPK: 4%');
	});
});

describe('chart formatter callbacks', () => {
	it('allocation xAxis formatter title-cases the category name', () => {
		const option = buildAllocationChartOption(allocationCategories);
		const xAxis = option.xAxis as { axisLabel: { formatter: (v: string) => string } };
		expect(xAxis.axisLabel.formatter('akcje')).toBe('Akcje');
		expect(xAxis.axisLabel.formatter('')).toBe('');
	});

	it('wrapper pie tooltip formats a single param', () => {
		const option = buildWrapperChartOption(wrappers);
		const tooltip = option.tooltip as unknown as {
			formatter: (p: { name: string; value: number; percent?: number }) => string;
		};
		const html = tooltip.formatter({ name: 'IKE', value: 12000, percent: 60 });
		expect(html).toContain('IKE');
		expect(html).toContain('12');
		expect(html).toContain('60.0%');
	});

	it('wrapper pie label formatter renders name and percent', () => {
		const option = buildWrapperChartOption(wrappers);
		const series = option.series as Array<{
			label: { formatter: (p: { name: string; percent?: number }) => string };
		}>;
		expect(series[0].label.formatter({ name: 'IKE', percent: 33.33 })).toContain('IKE');
		expect(series[0].label.formatter({ name: 'IKE', percent: 33.33 })).toContain('33.3%');
		expect(series[0].label.formatter({ name: 'IKE' })).toContain('0.0%');
	});

	it('wrapper pie tooltip handles array params (axis trigger fallback)', () => {
		const option = buildWrapperChartOption(wrappers);
		const tooltip = option.tooltip as unknown as {
			formatter: (p: Array<{ name: string; value: number; percent?: number }>) => string;
		};
		const html = tooltip.formatter([{ name: 'IKZE', value: 8000, percent: 40 }]);
		expect(html).toContain('IKZE');
	});

	it('investment trend tooltip computes returns and formats PLN', () => {
		const option = buildInvestmentTrendChartOption(trendSeries);
		const tooltip = option.tooltip as unknown as {
			formatter: (p: Array<{ axisValue: string; value: number; seriesName: string }>) => string;
		};
		const html = tooltip.formatter([
			{ axisValue: '2023-12-31', value: 2000, seriesName: 'Wpłaty' },
			{ axisValue: '2023-12-31', value: 2200, seriesName: 'Wartość portfela' }
		]);
		expect(html).toContain('2023-12-31');
		expect(html).toContain('Wartość portfela');
		expect(html).toContain('Wpłaty');
		expect(html).toContain('Zyski');
	});

	it('investment trend yAxis formatter renders k-shorthand', () => {
		const option = buildInvestmentTrendChartOption(trendSeries);
		const yAxis = option.yAxis as { axisLabel: { formatter: (v: number) => string } };
		expect(yAxis.axisLabel.formatter(15000)).toBe('15k');
		expect(yAxis.axisLabel.formatter(0)).toBe('0k');
	});

	it('wrapper trend tooltip and yAxis formatter cover the named-chart variant', () => {
		const option = buildWrapperTrendChartOption('IKE w czasie', trendSeries);
		const tooltip = option.tooltip as unknown as {
			formatter: (p: Array<{ axisValue: string; value: number; seriesName: string }>) => string;
		};
		const html = tooltip.formatter([
			{ axisValue: '2023-12-31', value: 2000, seriesName: 'Wpłaty' },
			{ axisValue: '2023-12-31', value: 2200, seriesName: 'Wartość portfela' }
		]);
		expect(html).toContain('Wartość');
		expect(html).toContain('Wpłaty');
		expect(html).toContain('Zyski');

		const yAxis = option.yAxis as { axisLabel: { formatter: (v: number) => string } };
		expect(yAxis.axisLabel.formatter(2500)).toBe('3k');
	});
});
