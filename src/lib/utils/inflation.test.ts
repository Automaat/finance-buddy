import { describe, it, expect } from 'vitest';
import { indexAtDate, inflationAdjust, realChange } from './inflation';
import type { CpiSeries } from '$lib/types/cpi';

const series: CpiSeries = {
	points: [
		{ year: 2020, yoy_rate: 100, cumulative_index: 100 },
		{ year: 2021, yoy_rate: 110, cumulative_index: 110 },
		{ year: 2022, yoy_rate: 120, cumulative_index: 132 }
	],
	base_year: 2020,
	latest_year: 2022,
	source: 'test'
};

describe('indexAtDate', () => {
	it('returns Jan 1 anchor for first day of year', () => {
		expect(indexAtDate(series, new Date(2021, 0, 1))).toBeCloseTo(110, 5);
	});

	it('interpolates linearly mid-year', () => {
		// Mid 2020 should be ~halfway between 100 and 110.
		const result = indexAtDate(series, new Date(2020, 6, 2));
		expect(result).toBeGreaterThan(104);
		expect(result).toBeLessThan(106);
	});

	it('clamps to latest known year past the end', () => {
		expect(indexAtDate(series, new Date(2099, 5, 1))).toBe(132);
	});

	it('clamps to earliest known year before start', () => {
		expect(indexAtDate(series, new Date(1900, 0, 1))).toBe(100);
	});

	it('returns null for empty series', () => {
		expect(indexAtDate({ ...series, points: [] }, new Date(2021, 0, 1))).toBeNull();
	});
});

describe('inflationAdjust', () => {
	it('compounds across full years', () => {
		const result = inflationAdjust(1000, new Date(2020, 0, 1), new Date(2022, 0, 1), series);
		expect(result).toBeCloseTo(1320, 5);
	});

	it('returns null when series is empty', () => {
		const empty = { ...series, points: [] };
		expect(inflationAdjust(1000, new Date(2020, 0, 1), new Date(2022, 0, 1), empty)).toBeNull();
	});
});

describe('realChange', () => {
	it('flags a raise that fails to beat inflation', () => {
		const result = realChange(1000, new Date(2020, 0, 1), 1100, new Date(2022, 0, 1), series);
		expect(result).not.toBeNull();
		expect(result!.nominalChangePln).toBe(100);
		expect(result!.previousInTodayPln).toBeCloseTo(1320, 5);
		expect(result!.realChangePln).toBeCloseTo(-220, 5);
		expect(result!.realChangePct).toBeLessThan(0);
	});

	it('confirms a raise that beats inflation', () => {
		const result = realChange(1000, new Date(2020, 0, 1), 1500, new Date(2022, 0, 1), series);
		expect(result).not.toBeNull();
		expect(result!.realChangePln).toBeCloseTo(180, 5);
		expect(result!.realChangePct).toBeGreaterThan(0);
	});
});
