import { describe, it, expect } from 'vitest';
import { buildWaterfallOption, buildWaterfallSteps, type NetWorthPoint } from './waterfall';

function point(
	date: string,
	snapshotId: number,
	value: number,
	assets: number,
	liabilities: number
): NetWorthPoint {
	return { date, snapshotId, value, assets, liabilities };
}

describe('buildWaterfallSteps', () => {
	it('returns empty for fewer than two points', () => {
		expect(buildWaterfallSteps([])).toEqual([]);
		expect(buildWaterfallSteps([point('2025-01-31', 1, 100, 150, 50)])).toEqual([]);
	});

	it('computes month-over-month deltas', () => {
		const steps = buildWaterfallSteps([
			point('2025-01-31', 1, 100, 200, 100),
			point('2025-02-28', 2, 130, 240, 110),
			point('2025-03-31', 3, 150, 250, 100)
		]);
		expect(steps).toHaveLength(2);
		expect(steps[0]).toMatchObject({
			snapshotId: 2,
			startingNetWorth: 100,
			endingNetWorth: 130,
			assetDelta: 40,
			liabilityDelta: 10
		});
		expect(steps[1]).toMatchObject({
			snapshotId: 3,
			startingNetWorth: 130,
			endingNetWorth: 150,
			assetDelta: 10,
			liabilityDelta: -10
		});
	});
});

describe('buildWaterfallOption', () => {
	const steps = buildWaterfallSteps([
		point('2025-01-31', 1, 100, 200, 100),
		point('2025-02-28', 2, 130, 240, 110), // assets +40, liab +10
		point('2025-03-31', 3, 150, 250, 100) // assets +10, liab -10
	]);

	it('emits 3 series: asset delta, liability delta, net worth line', () => {
		const opt = buildWaterfallOption(steps);
		const series = opt.series as Array<{ name: string; type: string }>;
		expect(series).toHaveLength(3);
		expect(series.map((s) => s.name)).toEqual(['Δ Aktywa', 'Δ Zobowiązania', 'Wartość netto']);
		expect(series[2].type).toBe('line');
	});

	it('liability series inverts the sign so an increase reads as a negative contribution', () => {
		const opt = buildWaterfallOption(steps);
		const series = opt.series as Array<{ data: Array<{ value: number }> }>;
		expect(series[1].data[0].value).toBe(-10); // liability grew → drag
		expect(series[1].data[1].value).toBe(10); // liability shrank → boost
	});

	it('green for positive asset delta, red for negative', () => {
		const negSteps = buildWaterfallSteps([
			point('2025-01-31', 1, 100, 200, 100),
			point('2025-02-28', 2, 80, 180, 100) // assets -20
		]);
		const opt = buildWaterfallOption(negSteps);
		const assetData = (opt.series as Array<{ data: Array<{ itemStyle: { color: string } }> }>)[0]
			.data;
		const liabData = (opt.series as Array<{ data: Array<{ itemStyle: { color: string } }> }>)[1]
			.data;
		expect(assetData[0].itemStyle.color).toBe('#BF616A'); // red
		expect(liabData[0].itemStyle.color).toBe('#A3BE8C'); // no liab change → green branch
	});

	it('maxMonths crops to the most recent N steps', () => {
		const many = buildWaterfallSteps([
			point('2025-01-31', 1, 100, 200, 100),
			point('2025-02-28', 2, 110, 210, 100),
			point('2025-03-31', 3, 120, 220, 100),
			point('2025-04-30', 4, 130, 230, 100),
			point('2025-05-31', 5, 140, 240, 100),
			point('2025-06-30', 6, 150, 250, 100),
			point('2025-07-31', 7, 160, 260, 100)
		]);
		const opt = buildWaterfallOption(many, { maxMonths: 3 });
		const xAxis = opt.xAxis as { data: string[] };
		expect(xAxis.data).toHaveLength(3);
		const series = opt.series as Array<{ data: unknown[] }>;
		expect(series[0].data).toHaveLength(3);
	});

	it('tooltip renders monthly breakdown', () => {
		const opt = buildWaterfallOption(steps);
		const tooltip = opt.tooltip as unknown as {
			formatter: (params: Array<{ dataIndex: number }>) => string;
		};
		const html = tooltip.formatter([{ dataIndex: 0 }]);
		expect(html).toContain('Saldo początkowe');
		expect(html).toContain('Δ Aktywa');
		expect(html).toContain('Δ Zobowiązania');
		expect(html).toContain('Saldo końcowe');
	});

	it('tooltip returns empty string when dataIndex is out of range', () => {
		const opt = buildWaterfallOption(steps);
		const tooltip = opt.tooltip as unknown as {
			formatter: (params: Array<{ dataIndex: number }>) => string;
		};
		expect(tooltip.formatter([{ dataIndex: 999 }])).toBe('');
	});

	it('tooltip accepts a single (non-array) param', () => {
		const opt = buildWaterfallOption(steps);
		const tooltip = opt.tooltip as unknown as {
			formatter: (params: { dataIndex: number }) => string;
		};
		expect(tooltip.formatter({ dataIndex: 0 })).toContain('Saldo początkowe');
	});

	it('xAxis falls back to the raw date when it is not parsable', () => {
		const opt = buildWaterfallOption([
			{
				date: 'not-a-date',
				snapshotId: 1,
				startingNetWorth: 0,
				endingNetWorth: 100,
				assetDelta: 100,
				liabilityDelta: 0
			},
			{
				date: 'still-not-a-date',
				snapshotId: 2,
				startingNetWorth: 100,
				endingNetWorth: 200,
				assetDelta: 100,
				liabilityDelta: 0
			}
		]);
		const xAxis = opt.xAxis as { data: string[] };
		expect(xAxis.data).toEqual(['not-a-date', 'still-not-a-date']);
	});

	it('both yAxis formatters render k-shorthand', () => {
		const opt = buildWaterfallOption(steps);
		const yAxes = opt.yAxis as Array<{ axisLabel: { formatter: (v: number) => string } }>;
		expect(yAxes[0].axisLabel.formatter(12000)).toBe('12k');
		expect(yAxes[1].axisLabel.formatter(45000)).toBe('45k');
	});
});
