import type {
	CompanyValuation,
	CustomVestingEvent,
	EquityGrant,
	VestingFrequency
} from '$lib/types/salaries';

const FREQ_MONTHS: Record<VestingFrequency, number> = {
	monthly: 1,
	quarterly: 3,
	yearly: 12
};

function parseIsoDate(iso: string): Date {
	const [y, m, d] = iso.split('-').map(Number);
	return new Date(y, m - 1, d);
}

function monthsBetween(start: Date, end: Date): number {
	let months = (end.getFullYear() - start.getFullYear()) * 12 + (end.getMonth() - start.getMonth());
	if (end.getDate() < start.getDate()) months -= 1;
	return months;
}

interface ScheduleInput {
	totalShares: number;
	vestStartDate: Date;
	vestCliffMonths: number;
	vestTotalMonths: number;
	vestFrequencyMonths: number;
	customSchedule: CustomVestingEvent[] | null;
}

function vestedSharesAt(s: ScheduleInput, onDate: Date): number {
	if (onDate < s.vestStartDate) return 0;
	const elapsed = monthsBetween(s.vestStartDate, onDate);
	if (elapsed < s.vestCliffMonths) return 0;
	const capped = Math.min(elapsed, s.vestTotalMonths);
	if (s.customSchedule && s.customSchedule.length > 0) {
		let totalPct = 0;
		for (const ev of s.customSchedule) if (ev.month <= capped) totalPct += ev.pct;
		const vested = Math.floor((s.totalShares * totalPct) / 100);
		return Math.min(vested, s.totalShares);
	}
	if (s.vestTotalMonths <= 0) return s.totalShares;
	const freq = s.vestFrequencyMonths > 0 ? s.vestFrequencyMonths : 1;
	const monthsAfterCliff = capped - s.vestCliffMonths;
	const extraPeriods = Math.floor(monthsAfterCliff / freq);
	const vestingMonthCount = Math.min(s.vestCliffMonths + extraPeriods * freq, s.vestTotalMonths);
	return Math.floor((s.totalShares * vestingMonthCount) / s.vestTotalMonths);
}

function scheduleOf(g: EquityGrant): ScheduleInput {
	return {
		totalShares: g.total_shares,
		vestStartDate: parseIsoDate(g.vest_start_date),
		vestCliffMonths: g.vest_cliff_months,
		vestTotalMonths: g.vest_total_months,
		vestFrequencyMonths: FREQ_MONTHS[g.vest_frequency],
		customSchedule: g.vest_custom_schedule
	};
}

// Time-vested shares newly accruing in calendar year. Ignores liquidity-event
// gating — use isEffectivelyVested() to apply that gate.
export function vestedInYear(g: EquityGrant, year: number): number {
	const s = scheduleOf(g);
	const endOfYear = new Date(year, 11, 31);
	const endOfPrev = new Date(year - 1, 11, 31);
	const cur = vestedSharesAt(s, endOfYear);
	const prev = vestedSharesAt(s, endOfPrev);
	return Math.max(0, cur - prev);
}

// Whether the grant's vested shares are realisable in `year` — i.e. either
// no liquidity event is required, or one has fired on/before year end.
export function isEffectivelyVested(g: EquityGrant, year: number): boolean {
	if (!g.requires_liquidity_event) return true;
	if (!g.liquidity_event_date) return false;
	return parseIsoDate(g.liquidity_event_date) <= new Date(year, 11, 31);
}

export function latestValuationFor(
	valuations: CompanyValuation[],
	company: string
): CompanyValuation | null {
	let best: CompanyValuation | null = null;
	for (const v of valuations) {
		if (!v.is_active) continue;
		if (v.company !== company) continue;
		if (!best || v.date > best.date) best = v;
	}
	return best;
}

interface PerShareValue {
	base: number;
	low: number;
	high: number;
	currency: string;
}

// Per-share intrinsic value in valuation currency: max(FMV - strike, 0) for
// options, FMV for RSUs. Returns null when no valuation exists for the
// company.
export function perShareIntrinsicValue(
	g: EquityGrant,
	v: CompanyValuation | null
): PerShareValue | null {
	if (!v) return null;
	const fmv = v.fmv_per_share;
	const low = v.fmv_low ?? fmv;
	const high = v.fmv_high ?? fmv;
	if (g.type === 'option') {
		const strike = g.strike_price ?? 0;
		return {
			base: Math.max(fmv - strike, 0),
			low: Math.max(low - strike, 0),
			high: Math.max(high - strike, 0),
			currency: v.currency
		};
	}
	return { base: fmv, low, high, currency: v.currency };
}

export interface YearlyEquityComp {
	vestedPln: number;
	vestedLowPln: number;
	vestedHighPln: number;
	lockedPln: number;
	hasEquityWithoutFx: boolean;
}

// FX is borrowed across grants of the same currency, since the backend only
// attaches fx_rate when vested_shares_today > 0 — locked RSUs need their
// sibling option grant's rate to convert to PLN.
function buildFxByCurrency(grants: EquityGrant[]): Map<string, number> {
	const fx = new Map<string, number>([['PLN', 1]]);
	for (const g of grants) {
		if (g.fx_rate !== null && g.paper_value_currency) fx.set(g.paper_value_currency, g.fx_rate);
	}
	return fx;
}

// Compute the year's equity comp for one owner: realisable value + locked
// (LE-pending) sub-total. Pure function so it can be unit-tested directly,
// independent of the salary page wiring.
export function computeYearlyEquityComp(
	grants: EquityGrant[],
	valuations: CompanyValuation[],
	ownerUserId: number,
	year: number
): YearlyEquityComp {
	const fxByCurrency = buildFxByCurrency(grants);
	let vestedPln = 0;
	let vestedLowPln = 0;
	let vestedHighPln = 0;
	let lockedPln = 0;
	let hasEquityWithoutFx = false;

	for (const g of grants) {
		if (g.owner_user_id !== ownerUserId) continue;
		const shares = vestedInYear(g, year);
		if (shares <= 0) continue;
		const valuation = latestValuationFor(valuations, g.company);
		const perShare = perShareIntrinsicValue(g, valuation);
		if (!perShare) continue;
		const fx = fxByCurrency.get(perShare.currency);
		if (fx === undefined) {
			hasEquityWithoutFx = true;
			continue;
		}
		const base = shares * perShare.base * fx;
		const low = shares * perShare.low * fx;
		const high = shares * perShare.high * fx;
		if (isEffectivelyVested(g, year)) {
			vestedPln += base;
			vestedLowPln += low;
			vestedHighPln += high;
		} else {
			lockedPln += base;
		}
	}

	return { vestedPln, vestedLowPln, vestedHighPln, lockedPln, hasEquityWithoutFx };
}
