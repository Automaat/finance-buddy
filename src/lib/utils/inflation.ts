import type { CpiPoint, CpiSeries } from '$lib/types/cpi';

/**
 * Linearly interpolate a fixed-base CPI index at a calendar date.
 * Mirrors backend logic in app/services/inflation.py for consistency.
 * Returns null if the series is empty.
 */
export function indexAtDate(series: CpiSeries, when: Date): number | null {
	const points = series.points;
	if (points.length === 0) return null;

	const sorted = [...points].sort((a, b) => a.year - b.year);
	const first = sorted[0];
	const last = sorted[sorted.length - 1];

	if (when.getFullYear() < first.year) return first.cumulative_index;
	if (when.getFullYear() >= last.year) return last.cumulative_index;

	const byYear = new Map<number, CpiPoint>();
	for (const p of sorted) byYear.set(p.year, p);

	const yearStart = new Date(when.getFullYear(), 0, 1);
	const nextYearStart = new Date(when.getFullYear() + 1, 0, 1);
	const span = nextYearStart.getTime() - yearStart.getTime();
	const fraction = (when.getTime() - yearStart.getTime()) / span;

	const start = byYear.get(when.getFullYear());
	const end = byYear.get(when.getFullYear() + 1);
	if (!start || !end) return null;
	return start.cumulative_index + (end.cumulative_index - start.cumulative_index) * fraction;
}

/**
 * Adjust `amount` from purchasing power on `fromDate` to `toDate`
 * using the supplied CPI series. Returns null if CPI data is missing.
 */
export function inflationAdjust(
	amount: number,
	fromDate: Date,
	toDate: Date,
	series: CpiSeries
): number | null {
	const fromIdx = indexAtDate(series, fromDate);
	const toIdx = indexAtDate(series, toDate);
	if (fromIdx == null || toIdx == null || fromIdx === 0) return null;
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
	series: CpiSeries
): RealChange | null {
	const previousInTodayPln = inflationAdjust(previousAmount, previousDate, currentDate, series);
	if (previousInTodayPln == null) return null;

	const nominalChangePln = currentAmount - previousAmount;
	const nominalChangePct = previousAmount === 0 ? 0 : (nominalChangePln / previousAmount) * 100;
	const realChangePln = currentAmount - previousInTodayPln;
	const realChangePct = previousInTodayPln === 0 ? 0 : (realChangePln / previousInTodayPln) * 100;
	return { nominalChangePln, nominalChangePct, realChangePln, realChangePct, previousInTodayPln };
}
