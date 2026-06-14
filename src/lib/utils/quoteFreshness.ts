// Quote-freshness helpers for the snapshot form. Investment accounts are
// auto-valued from the latest stored price quote; these flag positions whose
// quote is missing or stale so the user can refresh before snapshotting.

export interface HoldingQuote {
	security: { name: string };
	quantity: string;
	latest_quote_date: string | null;
}

export interface StaleQuote {
	name: string;
	date: string | null;
	daysOld: number;
}

// round2 matches the snapshot value's numeric(15,2) column and the value
// input's step="0.01", so auto-calculated values pass native validation.
// The 1e-8 nudge on the scaled value absorbs IEEE-754 error so exact-half
// cents round away from zero (e.g. 1.005 -> 1.01, not 1.00).
export const round2 = (n: number): number => {
	const scaled = n * 100;
	return Math.round(scaled + Math.sign(scaled) * 1e-8) / 100;
};

export function daysSince(date: string | null, nowMs: number): number {
	if (!date) return Number.POSITIVE_INFINITY;
	const then = new Date(`${date}T00:00:00Z`).getTime();
	return Math.floor((nowMs - then) / 86_400_000);
}

export function staleQuotes(
	holdings: HoldingQuote[],
	nowMs: number,
	staleDays: number
): StaleQuote[] {
	return holdings
		.filter((h) => parseFloat(h.quantity) > 0)
		.map((h) => ({
			name: h.security.name,
			date: h.latest_quote_date,
			daysOld: daysSince(h.latest_quote_date, nowMs)
		}))
		.filter((h) => h.daysOld > staleDays);
}
