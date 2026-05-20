import type { CpiPoint, CpiSeries } from '$lib/types/cpi';

/**
 * Pre-built lookup over a CPI series. Build once with {@link buildCpiLookup}
 * and reuse for many `indexAtDate` / `inflationAdjust` calls — avoids
 * repeated sort/copy/map-build work inside loops (e.g. when computing a
 * dashed real-value series for every salary record on a chart).
 */
export interface CpiLookup {
	byYear: Map<number, number>;
	first: CpiPoint;
	last: CpiPoint;
}

export function buildCpiLookup(series: CpiSeries): CpiLookup | null {
	if (series.points.length === 0) return null;
	const sorted = [...series.points].sort((a, b) => a.year - b.year);
	const byYear = new Map<number, number>();
	for (const p of sorted) byYear.set(p.year, p.cumulative_index);
	return { byYear, first: sorted[0], last: sorted[sorted.length - 1] };
}

/**
 * Parse an ISO `YYYY-MM-DD` date as a local calendar date. The native
 * `new Date('2021-01-01')` constructor parses date-only strings as UTC,
 * which can shift the year boundary in non-UTC timezones — that would
 * make `indexAtDate` read the wrong CPI year. Use this for any string
 * coming from the API.
 */
export function parseIsoDate(value: string): Date {
	const [y, m, d] = value.split('-').map(Number);
	return new Date(y, (m ?? 1) - 1, d ?? 1);
}

function dayOfYear(when: Date): number {
	const start = new Date(when.getFullYear(), 0, 1);
	// Use Math.round on (when - start) / msPerDay; UTC offsets cancel out
	// because both timestamps are in the same local zone.
	const msPerDay = 24 * 60 * 60 * 1000;
	return Math.round((when.getTime() - start.getTime()) / msPerDay);
}

function daysInYear(year: number): number {
	const leap = year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
	return leap ? 366 : 365;
}

/**
 * Linearly interpolate a fixed-base CPI index at a calendar date.
 * Mirrors backend logic in app/services/inflation.py: `index[Y]` is the
 * end-of-year-Y price level, so a date inside year Y interpolates between
 * `index[Y-1]` (start of Y) and `index[Y]` (end of Y) pro-rata by
 * day-of-year. Returns null if the lookup is missing.
 */
export function indexAtDate(lookup: CpiLookup, when: Date): number {
	const year = when.getFullYear();
	if (year < lookup.first.year) return lookup.first.cumulative_index;
	if (year > lookup.last.year) return lookup.last.cumulative_index;

	const end = lookup.byYear.get(year) ?? lookup.last.cumulative_index;
	const start = lookup.byYear.get(year - 1) ?? lookup.first.cumulative_index;
	const fraction = dayOfYear(when) / daysInYear(year);
	return start + (end - start) * fraction;
}

/**
 * Adjust `amount` from purchasing power on `fromDate` to `toDate`
 * using a pre-built CPI lookup. Returns null if the lookup is null
 * (no CPI data loaded yet) or the source index is zero.
 */
export function inflationAdjust(
	amount: number,
	fromDate: Date,
	toDate: Date,
	lookup: CpiLookup | null
): number | null {
	if (lookup == null) return null;
	const fromIdx = indexAtDate(lookup, fromDate);
	const toIdx = indexAtDate(lookup, toDate);
	if (fromIdx === 0) return null;
	return amount * (toIdx / fromIdx);
}

/**
 * Compute nominal vs real change given previous and current salary records.
 */
export interface RealChange {
	nominalChangePln: number;
	nominalChangePct: number;
	realChangePln: number;
	realChangePct: number;
	previousInTodayPln: number;
}

export function realChange(
	previousAmount: number,
	previousDate: Date,
	currentAmount: number,
	currentDate: Date,
	lookup: CpiLookup | null
): RealChange | null {
	const previousInTodayPln = inflationAdjust(previousAmount, previousDate, currentDate, lookup);
	if (previousInTodayPln == null) return null;

	const nominalChangePln = currentAmount - previousAmount;
	const nominalChangePct = previousAmount === 0 ? 0 : (nominalChangePln / previousAmount) * 100;
	const realChangePln = currentAmount - previousInTodayPln;
	const realChangePct = previousInTodayPln === 0 ? 0 : (realChangePln / previousInTodayPln) * 100;
	return { nominalChangePln, nominalChangePct, realChangePln, realChangePct, previousInTodayPln };
}
