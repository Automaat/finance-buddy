import { describe, it, expect } from 'vitest';
import {
	buildCpiLookup,
	indexAtDate,
	inflationAdjust,
	parseIsoDate,
	realChange
} from './inflation';
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

const lookup = buildCpiLookup(series)!;

describe('parseIsoDate', () => {
	it('parses YYYY-MM-DD as a local calendar date (no UTC shift)', () => {
		const d = parseIsoDate('2021-01-01');
		expect(d.getFullYear()).toBe(2021);
		expect(d.getMonth()).toBe(0);
		expect(d.getDate()).toBe(1);
	});
});

describe('buildCpiLookup', () => {
	it('returns null for empty series', () => {
		expect(buildCpiLookup({ ...series, points: [] })).toBeNull();
	});

	it('sorts points and indexes by year', () => {
		const reversed: CpiSeries = { ...series, points: [...series.points].reverse() };
		const l = buildCpiLookup(reversed)!;
		expect(l.first.year).toBe(2020);
		expect(l.last.year).toBe(2022);
		expect(l.byYear.get(2021)).toBe(110);
	});
});

describe('indexAtDate', () => {
	it('returns idx[Y-1] (start of Y) for Jan 1 of year Y', () => {
		// Jan 1 2022 = end of 2021 prices = idx[2021] = 110.
		expect(indexAtDate(lookup, new Date(2022, 0, 1))).toBeCloseTo(110, 5);
	});

	it('interpolates linearly mid-year', () => {
		// Mid 2021 should be ~halfway between idx[2020]=100 and idx[2021]=110.
		const result = indexAtDate(lookup, new Date(2021, 6, 2));
		expect(result).toBeGreaterThan(104);
		expect(result).toBeLessThan(106);
	});

	it('clamps to latest known year past the end', () => {
		expect(indexAtDate(lookup, new Date(2099, 5, 1))).toBe(132);
	});

	it('clamps to earliest known year before start', () => {
		expect(indexAtDate(lookup, new Date(1900, 0, 1))).toBe(100);
	});
});

describe('inflationAdjust', () => {
	it('compounds across full years', () => {
		// Start of 2021 (= end of 2020) -> past end of 2022 (clamped): 110 -> 132 -> factor 1.32.
		const result = inflationAdjust(1000, new Date(2021, 0, 1), new Date(2023, 0, 1), lookup);
		expect(result).toBeCloseTo(1320, 5);
	});

	it('returns null when lookup is null', () => {
		expect(inflationAdjust(1000, new Date(2021, 0, 1), new Date(2023, 0, 1), null)).toBeNull();
	});
});

describe('realChange', () => {
	it('flags a raise that fails to beat inflation', () => {
		const result = realChange(1000, new Date(2021, 0, 1), 1100, new Date(2023, 0, 1), lookup);
		expect(result).not.toBeNull();
		expect(result!.nominalChangePln).toBe(100);
		expect(result!.previousInTodayPln).toBeCloseTo(1320, 5);
		expect(result!.realChangePln).toBeCloseTo(-220, 5);
		expect(result!.realChangePct).toBeLessThan(0);
	});

	it('confirms a raise that beats inflation', () => {
		const result = realChange(1000, new Date(2021, 0, 1), 1500, new Date(2023, 0, 1), lookup);
		expect(result).not.toBeNull();
		expect(result!.realChangePln).toBeCloseTo(180, 5);
		expect(result!.realChangePct).toBeGreaterThan(0);
	});

	it('returns null when lookup is null', () => {
		expect(realChange(1000, new Date(2021, 0, 1), 1100, new Date(2023, 0, 1), null)).toBeNull();
	});
});
